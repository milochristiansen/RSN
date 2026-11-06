package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// AIConfig is compiled into the binary. There is no external config file and
// no environment override; changing any value is a source edit and a rebuild.
type AIConfig struct {
	Enabled bool

	// Also written statically into ai_prompts.go.
	BotName  string
	Streamer string
	Channel  ChannelConfig // used in the Clef state; not part of the reply prompt

	Context  ContextConfig
	Decision DecisionConfig
	Reply    ReplyConfig
	Limits   LimitsConfig
}

type ChannelConfig struct {
	Game  string
	Title string
}

type ContextConfig struct {
	Minutes     float64
	MaxMessages int
	MaxChars    int
}

type Question struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions,omitempty"`
	Criteria     map[string]string `json:"criteria,omitempty"`
}

type DecisionConfig struct {
	Model     string
	BaseURL   string
	Threshold float64
	Gate      string
	Timeout   time.Duration
	Questions map[string]Question
}

type ReplyConfig struct {
	Model           string
	BaseURL         string
	MaxTokens       int
	MaxChatChars    int
	Temperature     float64
	ReasoningEffort string
	Timeout         time.Duration
}

type LimitsConfig struct {
	QueueSize              int
	MaxMessageAge          time.Duration
	MaxRepliesPerHour      int
	MaxGateCallsPerMinute  int
	CircuitBreakerFailures int
	CircuitBreakerPause    time.Duration
}

func defaultAIConfig() AIConfig {
	return AIConfig{
		Enabled:  true,
		BotName:  "MechanicalMilo",
		Streamer: "Milo",
		Channel:  ChannelConfig{Game: "", Title: ""},
		Context: ContextConfig{
			Minutes:     10,
			MaxMessages: 80,
			MaxChars:    6000,
		},
		Decision: DecisionConfig{
			Model:     "@cf/cloudflare/clef",
			BaseURL:   "https://api.cloudflare.com/client/v4",
			Threshold: 0.60,
			Gate:      "respond",
			Timeout:   10 * time.Second,
			Questions: map[string]Question{
				"respond": {
					Type:         "noul",
					Instructions: decisionRespondPrompt,
				},
				"intent": {
					Type:         "choice",
					Instructions: decisionIntentPrompt,
					Criteria: map[string]string{
						"bot_request":   "Addressed to the bot, or asking the bot to say/do something.",
						"streamer_chat": "Aimed at the streamer, not the bot.",
						"viewer_chat":   "Aimed at another viewer or general chat.",
						"reaction":      "A reaction, joke, or comment that does not expect a bot reply.",
						"other":         "Anything else.",
					},
				},
			},
		},
		Reply: ReplyConfig{
			Model:           "deepseek-flash",
			BaseURL:         "https://api.deepseek.com",
			MaxTokens:       900,
			MaxChatChars:    480,
			Temperature:     1.2,
			ReasoningEffort: "low",
			Timeout:         30 * time.Second,
		},
		Limits: LimitsConfig{
			QueueSize:              32,
			MaxMessageAge:          45 * time.Second,
			MaxRepliesPerHour:      360,
			MaxGateCallsPerMinute:  120,
			CircuitBreakerFailures: 5,
			CircuitBreakerPause:    5 * time.Minute,
		},
	}
}

func validateAIConfig(cfg AIConfig) error {
	if strings.TrimSpace(cfg.BotName) == "" {
		return errors.New("ai: bot name must not be empty")
	}
	if strings.TrimSpace(cfg.Streamer) == "" {
		return errors.New("ai: streamer name must not be empty")
	}
	if cfg.Decision.Threshold < 0 || cfg.Decision.Threshold > 1 {
		return fmt.Errorf("ai: threshold %.3f must be between 0 and 1", cfg.Decision.Threshold)
	}
	if cfg.Context.Minutes <= 0 {
		return errors.New("ai: context minutes must be positive")
	}
	if cfg.Context.MaxMessages < 1 {
		return errors.New("ai: context max messages must be at least 1")
	}
	if cfg.Context.MaxChars < 100 {
		return errors.New("ai: context max chars must be at least 100")
	}
	if strings.TrimSpace(cfg.Decision.Model) == "" {
		return errors.New("ai: decision model must not be empty")
	}
	if strings.TrimSpace(cfg.Decision.BaseURL) == "" {
		return errors.New("ai: decision base URL must not be empty")
	}
	if cfg.Decision.Timeout <= 0 {
		return errors.New("ai: decision timeout must be positive")
	}
	gate, ok := cfg.Decision.Questions[cfg.Decision.Gate]
	if !ok {
		return fmt.Errorf("ai: gate question %q is not defined", cfg.Decision.Gate)
	}
	if gate.Type != "noul" {
		return fmt.Errorf("ai: gate question %q must be type noul", cfg.Decision.Gate)
	}
	for id, question := range cfg.Decision.Questions {
		switch question.Type {
		case "noul", "choice", "score":
		default:
			return fmt.Errorf("ai: question %q has unsupported type %q", id, question.Type)
		}
	}
	if strings.TrimSpace(cfg.Reply.Model) == "" {
		return errors.New("ai: reply model must not be empty")
	}
	if strings.TrimSpace(cfg.Reply.BaseURL) == "" {
		return errors.New("ai: reply base URL must not be empty")
	}
	if cfg.Reply.MaxTokens < 1 {
		return errors.New("ai: reply max tokens must be positive")
	}
	if cfg.Reply.MaxChatChars < 40 {
		return errors.New("ai: reply max chat chars must be at least 40")
	}
	if cfg.Reply.Temperature < 0 || cfg.Reply.Temperature > 2 {
		return fmt.Errorf("ai: reply temperature %.2f must be between 0 and 2", cfg.Reply.Temperature)
	}
	switch cfg.Reply.ReasoningEffort {
	case "", "low", "high", "max":
	default:
		return fmt.Errorf("ai: reasoning effort %q must be low, high, max, or empty", cfg.Reply.ReasoningEffort)
	}
	if cfg.Reply.Timeout <= 0 {
		return errors.New("ai: reply timeout must be positive")
	}
	if cfg.Limits.QueueSize < 1 {
		return errors.New("ai: queue size must be positive")
	}
	if cfg.Limits.MaxMessageAge <= 0 {
		return errors.New("ai: max message age must be positive")
	}
	if cfg.Limits.MaxRepliesPerHour < 1 {
		return errors.New("ai: max replies per hour must be positive")
	}
	if cfg.Limits.MaxGateCallsPerMinute < 1 {
		return errors.New("ai: max gate calls per minute must be positive")
	}
	if cfg.Limits.CircuitBreakerFailures < 1 {
		return errors.New("ai: circuit breaker failures must be positive")
	}
	if cfg.Limits.CircuitBreakerPause <= 0 {
		return errors.New("ai: circuit breaker pause must be positive")
	}
	return nil
}
