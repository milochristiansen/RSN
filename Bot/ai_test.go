package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestIRCLogin(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"normal", ":alice!alice@alice.tmi.twitch.tv PRIVMSG #milochristiansen :hello", "alice"},
		{"no user part", ":alice PRIVMSG #milochristiansen :hello", "alice"},
		{"tags prefix", "@display-name=Alice;id=abc :alice!alice@alice.tmi.twitch.tv PRIVMSG #milochristiansen :hello", "alice"},
		{"no prefix", "PING :tmi.twitch.tv", ""},
		{"empty", "", ""},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := ircLogin(test.raw); got != test.want {
				t.Fatalf("ircLogin(%q) = %q, want %q", test.raw, got, test.want)
			}
		})
	}
}

func TestChatMessageFromIRC(t *testing.T) {
	now := time.Date(2026, 11, 5, 20, 50, 0, 0, time.UTC)
	cases := []struct {
		name          string
		msg           *IRCMsg
		wantOK        bool
		wantLogin     string
		wantText      string
		wantBot       bool
		wantBroadcast bool
	}{
		{
			name:      "viewer",
			msg:       &IRCMsg{Command: "PRIVMSG", Params: []string{Channel, "milo what's a vod?"}, Raw: ":alice!alice@alice.tmi.twitch.tv PRIVMSG #milochristiansen :milo what's a vod?"},
			wantOK:    true,
			wantLogin: "alice",
			wantText:  "milo what's a vod?",
		},
		{
			name:      "bot",
			msg:       &IRCMsg{Command: "PRIVMSG", Params: []string{Channel, "hi"}, Raw: ":mechanicalmilo!mechanicalmilo@mechanicalmilo.tmi.twitch.tv PRIVMSG #milochristiansen :hi"},
			wantOK:    true,
			wantLogin: "mechanicalmilo",
			wantText:  "hi",
			wantBot:   true,
		},
		{
			name:          "broadcaster",
			msg:           &IRCMsg{Command: "PRIVMSG", Params: []string{Channel, "!ai off"}, Raw: ":milochristiansen!milochristiansen@milochristiansen.tmi.twitch.tv PRIVMSG #milochristiansen :!ai off"},
			wantOK:        true,
			wantLogin:     "milochristiansen",
			wantText:      "!ai off",
			wantBroadcast: true,
		},
		{
			name:   "wrong channel",
			msg:    &IRCMsg{Command: "PRIVMSG", Params: []string{"#other", "hi"}, Raw: ":alice!alice@alice.tmi.twitch.tv PRIVMSG #other :hi"},
			wantOK: false,
		},
		{
			name:   "short params",
			msg:    &IRCMsg{Command: "PRIVMSG", Params: []string{Channel}, Raw: ":alice!alice@alice.tmi.twitch.tv PRIVMSG #milochristiansen"},
			wantOK: false,
		},
		{
			name:   "nil",
			msg:    nil,
			wantOK: false,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got, ok := chatMessageFromIRC(test.msg, now)
			if ok != test.wantOK {
				t.Fatalf("ok = %v, want %v", ok, test.wantOK)
			}
			if !ok {
				return
			}
			if got.Login != test.wantLogin || got.DisplayName != test.wantLogin {
				t.Fatalf("login/display = %q/%q, want %q", got.Login, got.DisplayName, test.wantLogin)
			}
			if got.Text != test.wantText {
				t.Fatalf("text = %q, want %q", got.Text, test.wantText)
			}
			if got.IsBot != test.wantBot {
				t.Fatalf("IsBot = %v, want %v", got.IsBot, test.wantBot)
			}
			if got.IsBroadcaster != test.wantBroadcast {
				t.Fatalf("IsBroadcaster = %v, want %v", got.IsBroadcaster, test.wantBroadcast)
			}
			if !got.Time.Equal(now) {
				t.Fatalf("time = %v, want %v", got.Time, now)
			}
		})
	}
}

func TestBuildDecisionState(t *testing.T) {
	cfg := defaultAIConfig()
	base := time.Date(2026, 11, 5, 20, 48, 0, 0, time.UTC)
	first := ChatMessage{Seq: 1, Login: "alice", DisplayName: "alice", Text: "good morning chat", Time: base}
	trigger := ChatMessage{Seq: 2, Login: "carol", DisplayName: "carol", Text: "milo what's a vod?", Time: base.Add(2 * time.Minute)}

	state := buildDecisionState(&cfg, []ChatMessage{first, trigger}, trigger)
	chat, ok := state["chat_so_far"].([]string)
	if !ok || len(chat) != 1 || chat[0] != "[20:48] alice: good morning chat" {
		t.Fatalf("chat_so_far = %#v", state["chat_so_far"])
	}
	latest, ok := state["latest_message"].(map[string]string)
	if !ok || latest["author"] != "carol" || latest["text"] != "milo what's a vod?" {
		t.Fatalf("latest_message = %#v", state["latest_message"])
	}
	if _, ok := state["channel"]; ok {
		t.Fatal("state must not contain a nested channel object")
	}
	if _, ok := state["stream_game"]; ok {
		t.Fatal("empty stream_game should be omitted")
	}

	cfg.Channel = ChannelConfig{Game: "Turing Complete", Title: "building chips"}
	state = buildDecisionState(&cfg, []ChatMessage{first, trigger}, trigger)
	if state["stream_game"] != "Turing Complete" || state["stream_title"] != "building chips" {
		t.Fatalf("flat stream context missing: %#v", state)
	}
}

func TestCleanChatReply(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		maxChars int
		want     string
	}{
		{"whitespace", "Hello\r\nworld  !", 480, "Hello world !"},
		{"bot prefix", "MechanicalMilo: hi there", 480, "hi there"},
		{"control", "a\x00b", 480, "ab"},
		{"empty", "   ", 480, ""},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := cleanChatReply(test.input, "MechanicalMilo", test.maxChars); got != test.want {
				t.Fatalf("cleanChatReply = %q, want %q", got, test.want)
			}
		})
	}

	long := strings.Repeat("x", 20)
	got := cleanChatReply(long, "MechanicalMilo", 10)
	if len([]rune(got)) != 10 || !strings.HasSuffix(got, "…") {
		t.Fatalf("truncation = %q (runes %d)", got, len([]rune(got)))
	}
}

func TestHistoryWindow(t *testing.T) {
	history := newAIHistory()
	base := time.Date(2026, 11, 5, 20, 0, 0, 0, time.UTC)
	first := history.Append(ChatMessage{DisplayName: "alice", Text: "first", Time: base})
	trigger := history.Append(ChatMessage{DisplayName: "carol", Text: "trigger", Time: base.Add(5 * time.Minute)})
	history.Append(ChatMessage{DisplayName: "dave", Text: "later", Time: base.Add(6 * time.Minute)})

	window := history.Window(trigger, 10, 80, 6000)
	if len(window) != 2 {
		t.Fatalf("window length = %d, want 2: %#v", len(window), window)
	}
	if window[0].Text != "first" || window[1].Text != "trigger" {
		t.Fatalf("window order = %#v", window)
	}
	if window[0].Seq != first.Seq || window[1].Seq != trigger.Seq {
		t.Fatalf("window seqs = %#v", window)
	}

	if got := history.FormatTranscript(window); got != "[20:00] alice: first\n[20:05] carol: trigger" {
		t.Fatalf("transcript = %q", got)
	}
}

func TestAnswerProbability(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want float64
	}{
		{"float", `{"type":"noul","noul":0.983}`, 0.983},
		{"bool true", `{"type":"noul","noul":true}`, 1},
		{"bool false", `{"type":"noul","noul":false}`, 0},
		{"object", `{"type":"noul","noul":{"probability":0.5}}`, 0.5},
		{"fallback field", `{"type":"noul","probability":0.25}`, 0.25},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got, err := answerProbability(json.RawMessage(test.raw))
			if err != nil {
				t.Fatalf("answerProbability: %v", err)
			}
			if got != test.want {
				t.Fatalf("answerProbability = %v, want %v", got, test.want)
			}
		})
	}

	if _, err := answerProbability(json.RawMessage(`{"type":"choice","noul":0.5}`)); err == nil {
		t.Fatal("expected an error for a non-noul answer")
	}
}

func TestDefaultAIConfigValid(t *testing.T) {
	if err := validateAIConfig(defaultAIConfig()); err != nil {
		t.Fatalf("default config is invalid: %v", err)
	}
}

func TestIsAICommand(t *testing.T) {
	cases := map[string]bool{
		"!ai":         true,
		"!ai on":      true,
		"!AI OFF":     true,
		"!ai\tstatus": true,
		"!airplane":   false,
		"hello !ai":   false,
		"":            false,
	}
	for text, want := range cases {
		if got := isAICommand(text); got != want {
			t.Fatalf("isAICommand(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestPromptsAreStatic(t *testing.T) {
	prompts := map[string]string{
		"decisionRespondPrompt": decisionRespondPrompt,
		"decisionIntentPrompt":  decisionIntentPrompt,
		"replySystemPrompt":     replySystemPrompt,
	}
	for name, text := range prompts {
		if strings.ContainsAny(text, "{}") {
			t.Fatalf("%s contains placeholder braces", name)
		}
	}
}
