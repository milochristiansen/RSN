package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const maxResponseBytes = 4 << 20

var (
	clefModelRE   = regexp.MustCompile(`^[A-Za-z0-9@._/-]+$`)
	clefAccountRE = regexp.MustCompile(`^[A-Za-z0-9]+$`)
)

type clefResponse struct {
	Success *bool `json:"success"`
	Result  struct {
		Answers map[string]json.RawMessage `json:"answers"`
		Usage   map[string]any             `json:"usage"`
	} `json:"result"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type clefClient struct {
	accountID string
	apiKey    string
	cfg       DecisionConfig
	http      *http.Client
}

func newClefClient(accountID, apiKey string, cfg DecisionConfig) *clefClient {
	return &clefClient{
		accountID: accountID,
		apiKey:    apiKey,
		cfg:       cfg,
		http:      &http.Client{},
	}
}

func (c *clefClient) decide(ctx context.Context, state map[string]any, questions map[string]Question) (map[string]json.RawMessage, map[string]any, error) {
	if !clefAccountRE.MatchString(c.accountID) {
		return nil, nil, fmt.Errorf("clef: invalid Cloudflare account ID")
	}
	if !clefModelRE.MatchString(c.cfg.Model) {
		return nil, nil, fmt.Errorf("clef: unsupported model name %q", c.cfg.Model)
	}

	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/accounts/" + c.accountID + "/ai/run/" + c.cfg.Model
	payload := struct {
		State     map[string]any      `json:"state"`
		Questions map[string]Question `json:"questions"`
	}{
		State:     state,
		Questions: questions,
	}
	data, err := postJSON(ctx, c.http, url, map[string]string{
		"Authorization": "Bearer " + c.apiKey,
	}, payload, "clef")
	if err != nil {
		return nil, nil, err
	}

	var response clefResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil, fmt.Errorf("clef: decode response: %w", err)
	}
	if response.Success != nil && !*response.Success {
		if len(response.Errors) > 0 && response.Errors[0].Message != "" {
			return nil, nil, fmt.Errorf("clef: API error: %s", response.Errors[0].Message)
		}
		return nil, nil, errors.New("clef: API returned success=false")
	}
	if response.Result.Answers == nil {
		return nil, nil, errors.New("clef: response has no answers")
	}
	usage := response.Result.Usage
	if usage == nil {
		usage = map[string]any{}
	}
	return response.Result.Answers, usage, nil
}

// answerProbability reads a noul answer. The API normally returns a float, but
// accept a bool or an object too so a small API shape change does not take the
// bot offline.
func answerProbability(raw json.RawMessage) (float64, error) {
	var answer struct {
		Type        string          `json:"type"`
		Noul        json.RawMessage `json:"noul"`
		Probability json.RawMessage `json:"probability"`
	}
	if err := json.Unmarshal(raw, &answer); err != nil {
		return 0, fmt.Errorf("decode answer: %w", err)
	}
	if answer.Type != "" && answer.Type != "noul" {
		return 0, fmt.Errorf("answer type %q is not noul", answer.Type)
	}
	value := answer.Noul
	if len(value) == 0 {
		value = answer.Probability
	}
	if len(value) == 0 {
		return 0, errors.New("noul answer has no value")
	}

	var decoded any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return 0, fmt.Errorf("decode noul value: %w", err)
	}
	switch typed := decoded.(type) {
	case float64:
		return typed, nil
	case bool:
		if typed {
			return 1, nil
		}
		return 0, nil
	case map[string]any:
		for _, key := range []string{"true", "prob", "probability"} {
			switch value := typed[key].(type) {
			case float64:
				return value, nil
			case bool:
				if value {
					return 1, nil
				}
				return 0, nil
			}
		}
	}
	return 0, fmt.Errorf("unsupported noul value %s", string(value))
}

type deepSeekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type deepSeekRequest struct {
	Model           string            `json:"model"`
	Messages        []deepSeekMessage `json:"messages"`
	MaxTokens       int               `json:"max_tokens"`
	Temperature     float64           `json:"temperature"`
	ReasoningEffort string            `json:"reasoning_effort,omitempty"`
	Stream          bool              `json:"stream"`
}

type deepSeekUsage struct {
	PromptTokens        int `json:"prompt_tokens"`
	CompletionTokens    int `json:"completion_tokens"`
	PromptTokensDetails struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"prompt_tokens_details"`
	CompletionTokensDetails struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"completion_tokens_details"`
}

type deepSeekResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage deepSeekUsage `json:"usage"`
}

type deepSeekResult struct {
	Model        string
	Text         string
	RawText      string
	Reasoning    string
	FinishReason string
	Usage        deepSeekUsage
}

type deepSeekClient struct {
	apiKey  string
	botName string
	cfg     ReplyConfig
	http    *http.Client
}

func newDeepSeekClient(apiKey, botName string, cfg ReplyConfig) *deepSeekClient {
	return &deepSeekClient{
		apiKey:  apiKey,
		botName: botName,
		cfg:     cfg,
		http:    &http.Client{},
	}
}

func (c *deepSeekClient) reply(ctx context.Context, systemPrompt, userPrompt string) (deepSeekResult, error) {
	payload := deepSeekRequest{
		Model: c.cfg.Model,
		Messages: []deepSeekMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:       c.cfg.MaxTokens,
		Temperature:     c.cfg.Temperature,
		ReasoningEffort: c.cfg.ReasoningEffort,
		Stream:          false,
	}
	data, err := postJSON(ctx, c.http, strings.TrimRight(c.cfg.BaseURL, "/")+"/chat/completions", map[string]string{
		"Authorization": "Bearer " + c.apiKey,
	}, payload, c.cfg.Model)
	if err != nil {
		return deepSeekResult{}, err
	}

	var response deepSeekResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return deepSeekResult{}, fmt.Errorf("%s: decode response: %w", c.cfg.Model, err)
	}
	if len(response.Choices) == 0 {
		return deepSeekResult{}, fmt.Errorf("%s: response has no choices", c.cfg.Model)
	}
	choice := response.Choices[0]
	rawText := strings.TrimSpace(choice.Message.Content)
	if rawText == "" {
		hint := ""
		if choice.FinishReason == "length" {
			hint = " (hit max_tokens while reasoning; raise Reply.MaxTokens)"
		}
		return deepSeekResult{}, fmt.Errorf("%s: returned no text%s", c.cfg.Model, hint)
	}
	text := cleanChatReply(rawText, c.botName, c.cfg.MaxChatChars)
	if text == "" {
		return deepSeekResult{}, fmt.Errorf("%s: returned only unusable text", c.cfg.Model)
	}
	model := response.Model
	if model == "" {
		model = c.cfg.Model
	}
	return deepSeekResult{
		Model:        model,
		Text:         text,
		RawText:      rawText,
		Reasoning:    choice.Message.ReasoningContent,
		FinishReason: choice.FinishReason,
		Usage:        response.Usage,
	}, nil
}

func postJSON(ctx context.Context, client *http.Client, url string, headers map[string]string, payload any, label string) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal request: %w", label, err)
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("%s: build request: %w", label, err)
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json")
		for key, value := range headers {
			request.Header.Set(key, value)
		}

		response, err := client.Do(request)
		if err != nil {
			lastErr = fmt.Errorf("%s: request failed: %w", label, err)
			if attempt == 2 || ctx.Err() != nil {
				return nil, lastErr
			}
			if sleepErr := sleepContext(ctx, retryDelay("", attempt)); sleepErr != nil {
				return nil, lastErr
			}
			continue
		}

		data, readErr := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
		closeErr := response.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("%s: read response: %w", label, readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("%s: close response: %w", label, closeErr)
		}

		if response.StatusCode < 200 || response.StatusCode >= 300 {
			lastErr = fmt.Errorf("%s: HTTP %d: %s", label, response.StatusCode, errorMessage(data))
			if attempt == 2 || !retryableStatus(response.StatusCode) {
				return nil, lastErr
			}
			if sleepErr := sleepContext(ctx, retryDelay(response.Header.Get("Retry-After"), attempt)); sleepErr != nil {
				return nil, lastErr
			}
			continue
		}
		return data, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("%s: request failed", label)
	}
	return nil, lastErr
}

func retryableStatus(status int) bool {
	switch status {
	case http.StatusRequestTimeout, http.StatusTooManyRequests,
		http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func retryDelay(retryAfter string, attempt int) time.Duration {
	if retryAfter != "" {
		seconds, err := strconv.ParseFloat(strings.TrimSpace(retryAfter), 64)
		if err == nil {
			delay := time.Duration(seconds * float64(time.Second))
			if delay < 500*time.Millisecond {
				delay = 500 * time.Millisecond
			}
			if delay > 15*time.Second {
				delay = 15 * time.Second
			}
			return delay
		}
	}
	delay := time.Duration(1<<uint(attempt)) * time.Second
	if delay > 8*time.Second {
		delay = 8 * time.Second
	}
	return delay
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func errorMessage(data []byte) string {
	if len(strings.TrimSpace(string(data))) == 0 {
		return "(empty response)"
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return truncateText(string(data), 400)
	}
	if errorsValue, ok := parsed["errors"].([]any); ok && len(errorsValue) > 0 {
		if first, ok := errorsValue[0].(map[string]any); ok {
			if message, ok := first["message"].(string); ok && message != "" {
				return message
			}
		}
	}
	if errorValue, ok := parsed["error"].(map[string]any); ok {
		if message, ok := errorValue["message"].(string); ok && message != "" {
			return message
		}
	}
	if message, ok := parsed["error"].(string); ok && message != "" {
		return message
	}
	if message, ok := parsed["message"].(string); ok && message != "" {
		return message
	}
	return truncateText(string(data), 400)
}

func truncateText(text string, limit int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= limit {
		return string(runes)
	}
	if limit < 1 {
		return ""
	}
	return string(runes[:limit-1]) + "…"
}
