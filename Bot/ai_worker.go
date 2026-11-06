package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/milochristiansen/sessionlogger"
)

const (
	dedupeTTL   = 5 * time.Second
	sendTimeout = 5 * time.Second
)

var errAIStopped = errors.New("ai stopped")

type AI struct {
	cfg  AIConfig
	log  *sessionlogger.Logger
	send func(string) error

	history   *aiHistory
	clef      *clefClient
	deepseek  *deepSeekClient
	questions map[string]Question

	queue     chan ChatMessage
	stop      chan struct{}
	wg        sync.WaitGroup
	closeOnce sync.Once

	enabled     atomic.Bool
	failures    atomic.Int32
	pausedUntil atomic.Int64

	mu              sync.Mutex
	dedupe          map[string]time.Time
	gateTimes       []time.Time
	replyTimes      []time.Time
	lastProbability float64
	lastError       string

	gateCalls    atomic.Int64
	repliesSent  atomic.Int64
	ignored      atomic.Int64
	dropped      atomic.Int64
	errorCount   atomic.Int64
	rateLimited  atomic.Int64
	circuitTrips atomic.Int64
}

func NewAI(cfg AIConfig, log *sessionlogger.Logger, send func(string) error) (*AI, error) {
	if err := validateAIConfig(cfg); err != nil {
		return nil, err
	}
	if log == nil {
		log = sessionlogger.NewSessionLogger("ai")
	}
	if send == nil {
		return nil, errors.New("ai: send function must not be nil")
	}
	if CloudflareAccountID == "" || CloudflareAPIKey == "" {
		return nil, errors.New("ai: missing Cloudflare credentials in keys.go")
	}
	if DeepSeekAPIKey == "" {
		return nil, errors.New("ai: missing DeepSeek credentials in keys.go")
	}

	ai := &AI{
		cfg:        cfg,
		log:        log,
		send:       send,
		history:    newAIHistory(),
		clef:       newClefClient(CloudflareAccountID, CloudflareAPIKey, cfg.Decision),
		deepseek:   newDeepSeekClient(DeepSeekAPIKey, cfg.BotName, cfg.Reply),
		questions:  cfg.Decision.Questions,
		queue:      make(chan ChatMessage, cfg.Limits.QueueSize),
		stop:       make(chan struct{}),
		dedupe:     make(map[string]time.Time),
		gateTimes:  make([]time.Time, 0, cfg.Limits.MaxGateCallsPerMinute),
		replyTimes: make([]time.Time, 0, cfg.Limits.MaxRepliesPerHour),
	}
	ai.enabled.Store(cfg.Enabled)
	ai.wg.Add(1)
	go ai.worker()
	return ai, nil
}

func (a *AI) Close() {
	a.closeOnce.Do(func() {
		close(a.stop)
		done := make(chan struct{})
		go func() {
			a.wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			a.log.W.Println("ai: worker did not stop in time")
		}
	})
}

// Consider is called from the IRC read loop. It only parses, records history,
// checks limits, and does a non-blocking queue send; it never performs I/O.
func (a *AI) Consider(i *IRCMsg) {
	select {
	case <-a.stop:
		return
	default:
	}
	message, ok := chatMessageFromIRC(i, time.Now())
	if !ok || message.IsBot {
		return
	}
	if !a.dedupeOK(message) {
		a.ignored.Add(1)
		return
	}

	message = a.history.Append(message)
	if !a.enabled.Load() {
		a.ignored.Add(1)
		return
	}
	if strings.HasPrefix(message.Text, "!") {
		a.ignored.Add(1)
		return
	}
	if a.isPaused() {
		a.ignored.Add(1)
		return
	}
	if !a.gateAllowed() {
		a.rateLimited.Add(1)
		a.log.I.Printf("skip seq=%d reason=gate_cap", message.Seq)
		return
	}

	select {
	case a.queue <- message:
	default:
		a.dropped.Add(1)
		a.log.I.Printf("drop seq=%d reason=queue_full", message.Seq)
	}
}

func isAICommand(text string) bool {
	fields := strings.Fields(text)
	return len(fields) > 0 && strings.EqualFold(fields[0], "!ai")
}

func (a *AI) HandleCommand(i *IRCMsg) bool {
	if i == nil || i.Command != "PRIVMSG" || len(i.Params) < 2 {
		return false
	}
	text := strings.TrimSpace(i.Params[len(i.Params)-1])
	if !isAICommand(text) {
		return false
	}
	login := ircLogin(i.Raw)
	if !strings.EqualFold(login, strings.TrimPrefix(Channel, "#")) {
		a.log.W.Printf("ignoring !ai from %q", login)
		return true
	}

	fields := strings.Fields(text)
	if len(fields) == 1 {
		a.sendAsync(a.statusLine())
		return true
	}

	switch strings.ToLower(fields[1]) {
	case "on":
		a.enabled.Store(true)
		a.log.I.Println("AI enabled by broadcaster")
		a.sendAsync("AI enabled.")
	case "off":
		a.enabled.Store(false)
		a.log.I.Println("AI disabled by broadcaster")
		a.sendAsync("AI disabled.")
	case "status":
		a.sendAsync(a.statusLine())
	case "reset":
		a.failures.Store(0)
		a.pausedUntil.Store(0)
		a.log.I.Println("AI circuit breaker reset by broadcaster")
		a.sendAsync("AI breaker reset.")
	case "clear":
		a.history.Clear()
		a.log.I.Println("AI history cleared by broadcaster")
		a.sendAsync("AI history cleared.")
	default:
		a.sendAsync("Usage: !ai on|off|status|reset|clear")
	}
	return true
}

func (a *AI) sendAsync(text string) {
	select {
	case <-a.stop:
		return
	default:
	}
	go func() {
		if err := a.sendWithTimeout(text); err != nil && !errors.Is(err, errAIStopped) {
			a.log.E.Println("say:", err)
		}
	}()
}

func (a *AI) sendWithTimeout(text string) error {
	text = cleanChatReply(text, a.cfg.BotName, a.cfg.Reply.MaxChatChars)
	if text == "" {
		return errors.New("refusing to send an empty chat message")
	}
	result := make(chan error, 1)
	go func() {
		result <- a.send(text)
	}()
	select {
	case err := <-result:
		return err
	case <-time.After(sendTimeout):
		return errors.New("say timed out")
	case <-a.stop:
		return errAIStopped
	}
}

func (a *AI) worker() {
	defer a.wg.Done()
	for {
		select {
		case <-a.stop:
			return
		default:
		}
		select {
		case <-a.stop:
			return
		case message := <-a.queue:
			a.process(message)
		}
	}
}

func (a *AI) process(message ChatMessage) {
	age := time.Since(message.Time)
	if age > a.cfg.Limits.MaxMessageAge {
		a.dropped.Add(1)
		a.log.I.Printf("drop seq=%d reason=stale age=%s", message.Seq, age.Round(time.Millisecond))
		return
	}
	if !a.enabled.Load() || a.isPaused() {
		a.ignored.Add(1)
		return
	}

	window := a.history.Window(message, a.cfg.Context.Minutes, a.cfg.Context.MaxMessages, a.cfg.Context.MaxChars)
	if len(window) == 0 || window[len(window)-1].Seq != message.Seq {
		// History can be cleared while a message is queued. Keep the trigger in
		// the prompt even in that edge case.
		window = append(window, message)
	}
	state := buildDecisionState(&a.cfg, window, message)

	gateStarted := time.Now()
	gateCtx, gateCancel := context.WithTimeout(context.Background(), a.cfg.Decision.Timeout)
	answers, usage, err := a.clef.decide(gateCtx, state, a.questions)
	gateCancel()
	if err != nil {
		a.recordFailure("clef", err)
		a.log.E.Println("clef:", err)
		return
	}
	gateMS := time.Since(gateStarted).Milliseconds()

	gateRaw, ok := answers[a.cfg.Decision.Gate]
	if !ok {
		err := fmt.Errorf("missing gate question %q", a.cfg.Decision.Gate)
		a.recordFailure("clef", err)
		a.log.E.Println("clef:", err)
		return
	}
	probability, err := answerProbability(gateRaw)
	if err != nil {
		a.recordFailure("clef", err)
		a.log.E.Println("clef:", err)
		return
	}
	if probability < 0 || probability > 1 {
		a.log.W.Printf("clef: probability %.3f outside [0,1] for seq=%d", probability, message.Seq)
		probability = math.Max(0, math.Min(1, probability))
	}
	a.recordSuccess()
	a.gateCalls.Add(1)
	a.setLastProbability(probability)

	wantsReply := probability >= a.cfg.Decision.Threshold
	a.log.I.Printf("decision seq=%d user=%s p=%.3f threshold=%.2f wants=%t gate_ms=%d input_tokens=%v",
		message.Seq, message.Login, probability, a.cfg.Decision.Threshold, wantsReply, gateMS, usage["input_tokens"])
	if !wantsReply {
		return
	}
	if !a.enabled.Load() {
		return
	}
	if !a.replyAllowed() {
		a.rateLimited.Add(1)
		a.log.I.Printf("skip seq=%d reason=hourly_reply_cap", message.Seq)
		return
	}

	// The system prompt is fully static. The transcript is the only per-message
	// input, and it is sent as the user message so the system prompt remains a
	// stable prefix for prompt caching.
	transcript := a.history.FormatTranscript(window)
	systemPrompt := replySystemPrompt
	userPrompt := transcript

	replyStarted := time.Now()
	replyCtx, replyCancel := context.WithTimeout(context.Background(), a.cfg.Reply.Timeout)
	result, err := a.deepseek.reply(replyCtx, systemPrompt, userPrompt)
	replyCancel()
	replyMS := time.Since(replyStarted).Milliseconds()
	if err != nil {
		a.recordFailure("deepseek", err)
		a.log.E.Println("deepseek:", err)
		return
	}
	a.recordSuccess()

	select {
	case <-a.stop:
		return
	default:
	}
	if !a.enabled.Load() {
		return
	}
	if err := a.sendWithTimeout(result.Text); err != nil {
		if errors.Is(err, errAIStopped) {
			return
		}
		a.errorCount.Add(1)
		a.setLastError("say: " + err.Error())
		a.log.E.Println("say:", err)
		return
	}
	a.recordReply()
	a.history.Append(ChatMessage{
		Login:       BotName,
		DisplayName: a.cfg.BotName,
		Text:        result.Text,
		Time:        time.Now(),
		IsBot:       true,
	})
	a.log.I.Printf("reply seq=%d user=%s reply_ms=%d in=%d out=%d chars=%d text=%q",
		message.Seq, message.Login, replyMS, result.Usage.PromptTokens, result.Usage.CompletionTokens,
		len([]rune(result.Text)), result.Text)
}

func (a *AI) dedupeOK(message ChatMessage) bool {
	now := time.Now()
	key := strings.ToLower(message.Login) + "\x00" + message.Text

	a.mu.Lock()
	defer a.mu.Unlock()
	for existing, seen := range a.dedupe {
		if now.Sub(seen) > dedupeTTL {
			delete(a.dedupe, existing)
		}
	}
	if seen, ok := a.dedupe[key]; ok && now.Sub(seen) <= dedupeTTL {
		return false
	}
	a.dedupe[key] = now
	return true
}

func (a *AI) gateAllowed() bool {
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()

	a.gateTimes = pruneTimes(a.gateTimes, now.Add(-time.Minute))
	if len(a.gateTimes) >= a.cfg.Limits.MaxGateCallsPerMinute {
		return false
	}
	a.gateTimes = append(a.gateTimes, now)
	return true
}

func (a *AI) replyAllowed() bool {
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()

	a.replyTimes = pruneTimes(a.replyTimes, now.Add(-time.Hour))
	return len(a.replyTimes) < a.cfg.Limits.MaxRepliesPerHour
}

func (a *AI) recordReply() {
	a.mu.Lock()
	a.replyTimes = append(a.replyTimes, time.Now())
	a.mu.Unlock()
	a.repliesSent.Add(1)
}

func pruneTimes(times []time.Time, cutoff time.Time) []time.Time {
	n := 0
	for _, stamp := range times {
		if stamp.After(cutoff) {
			times[n] = stamp
			n++
		}
	}
	return times[:n]
}

func (a *AI) isPaused() bool {
	until := a.pausedUntil.Load()
	return until > 0 && time.Now().UnixNano() < until
}

func (a *AI) recordFailure(stage string, err error) {
	a.errorCount.Add(1)
	a.setLastError(stage + ": " + err.Error())
	failures := a.failures.Add(1)
	if int(failures) >= a.cfg.Limits.CircuitBreakerFailures {
		until := time.Now().Add(a.cfg.Limits.CircuitBreakerPause)
		a.pausedUntil.Store(until.UnixNano())
		a.failures.Store(0)
		a.circuitTrips.Add(1)
		a.log.E.Printf("circuit breaker: pausing AI until %s", until.Format(time.RFC3339))
	}
}

func (a *AI) recordSuccess() {
	a.failures.Store(0)
}

func (a *AI) setLastError(message string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastError = message
}

func (a *AI) setLastProbability(probability float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastProbability = probability
}

func (a *AI) statusLine() string {
	a.mu.Lock()
	probability := a.lastProbability
	lastError := a.lastError
	a.mu.Unlock()

	state := "off"
	if a.enabled.Load() {
		state = "on"
	}
	if a.isPaused() {
		state += "/paused"
	}
	line := fmt.Sprintf("AI %s | p=%.3f | q=%d/%d | gates=%d replies=%d dropped=%d errors=%d",
		state, probability, len(a.queue), cap(a.queue), a.gateCalls.Load(), a.repliesSent.Load(),
		a.dropped.Load(), a.errorCount.Load())
	if lastError != "" {
		line += " | last=" + lastError
	}
	return truncateText(line, 450)
}
