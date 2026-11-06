package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestClefDecide(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/accounts/abc123/ai/run/@cf/cloudflare/clef" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("authorization header = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, `{"success":true,"result":{"answers":{"respond":{"type":"noul","noul":0.97}},"usage":{"input_tokens":42}}}`); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	cfg := DecisionConfig{
		Model:   "@cf/cloudflare/clef",
		BaseURL: server.URL,
		Timeout: time.Second,
	}
	client := newClefClient("abc123", "test-key", cfg)
	answers, usage, err := client.decide(context.Background(), map[string]any{}, map[string]Question{})
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	probability, err := answerProbability(answers["respond"])
	if err != nil {
		t.Fatalf("answerProbability: %v", err)
	}
	if probability != 0.97 {
		t.Fatalf("probability = %v", probability)
	}
	if usage["input_tokens"] != float64(42) {
		t.Fatalf("usage = %#v", usage)
	}
}

func TestDeepSeekReply(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, `{"model":"deepseek-flash","choices":[{"message":{"content":"  hi\r\nthere  ","reasoning_content":"thinking"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	cfg := ReplyConfig{
		Model:        "deepseek-flash",
		BaseURL:      server.URL,
		MaxTokens:    100,
		MaxChatChars: 480,
		Temperature:  0.5,
		Timeout:      time.Second,
	}
	client := newDeepSeekClient("test-key", "MechanicalMilo", cfg)
	result, err := client.reply(context.Background(), "system", "user")
	if err != nil {
		t.Fatalf("reply: %v", err)
	}
	if result.Text != "hi there" {
		t.Fatalf("text = %q", result.Text)
	}
	if result.Usage.PromptTokens != 10 || result.Usage.CompletionTokens != 5 {
		t.Fatalf("usage = %#v", result.Usage)
	}
}

func TestDeepSeekEmptyReply(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, `{"choices":[{"message":{"content":""},"finish_reason":"length"}]}`); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	cfg := ReplyConfig{
		Model:        "deepseek-flash",
		BaseURL:      server.URL,
		MaxTokens:    100,
		MaxChatChars: 480,
		Timeout:      time.Second,
	}
	client := newDeepSeekClient("test-key", "MechanicalMilo", cfg)
	_, err := client.reply(context.Background(), "system", "user")
	if err == nil || !strings.Contains(err.Error(), "max_tokens") {
		t.Fatalf("err = %v, want max_tokens hint", err)
	}
}

func TestPostJSONRetries(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			if _, err := io.WriteString(w, `{"error":{"message":"slow down"}}`); err != nil {
				t.Errorf("write response: %v", err)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, `{"ok":true}`); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	data, err := postJSON(ctx, &http.Client{}, server.URL, nil, map[string]string{"a": "b"}, "test")
	if err != nil {
		t.Fatalf("postJSON: %v", err)
	}
	if string(data) != `{"ok":true}` {
		t.Fatalf("data = %q", data)
	}
	if attempts.Load() != 2 {
		t.Fatalf("attempts = %d, want 2", attempts.Load())
	}
}

func TestPostJSONDoesNotRetry401(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := io.WriteString(w, `{"error":{"message":"bad key"}}`); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := postJSON(ctx, &http.Client{}, server.URL, nil, map[string]string{"a": "b"}, "test")
	if err == nil || !strings.Contains(err.Error(), "bad key") {
		t.Fatalf("err = %v, want bad key", err)
	}
	if attempts.Load() != 1 {
		t.Fatalf("attempts = %d, want 1", attempts.Load())
	}
}
