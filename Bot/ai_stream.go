package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// StreamInfo is the small, slow-changing stream context that is included in
// the DeepSeek user message. It is intentionally not part of the static system
// prompt so the system prompt stays cacheable.
type StreamInfo struct {
	Title    string
	Category string
}

func cleanStreamText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func (a *AI) SetStreamInfo(title, category string) {
	info := &StreamInfo{
		Title:    cleanStreamText(title),
		Category: cleanStreamText(category),
	}
	a.streamInfo.Store(info)
	a.log.I.Printf("stream info: category=%q title=%q", info.Category, info.Title)
}

func (a *AI) currentStreamInfo() StreamInfo {
	if info := a.streamInfo.Load(); info != nil {
		return *info
	}
	return StreamInfo{}
}

// buildReplyUserPrompt adds the compact stream information block in front of
// the transcript. The system prompt remains fully static.
func buildReplyUserPrompt(info StreamInfo, transcript string) string {
	if info.Title == "" && info.Category == "" {
		return transcript
	}

	var builder strings.Builder
	builder.WriteString("Stream information:\n")
	if info.Category != "" {
		builder.WriteString("- category: ")
		builder.WriteString(info.Category)
		builder.WriteByte('\n')
	}
	if info.Title != "" {
		builder.WriteString("- title: ")
		builder.WriteString(info.Title)
		builder.WriteByte('\n')
	}
	builder.WriteString("\n")
	builder.WriteString(transcript)
	return builder.String()
}

// fetchChannelInfo gets the initial title/category. EventSub channel.update
// keeps it current afterward.
func fetchChannelInfo(ctx context.Context) (StreamInfo, error) {
	token, err := AppToken.Token()
	if err != nil {
		return StreamInfo{}, fmt.Errorf("channel info: app token: %w", err)
	}

	endpoint := "https://api.twitch.tv/helix/channels?broadcaster_id=" + url.QueryEscape(ChannelID)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return StreamInfo{}, fmt.Errorf("channel info: build request: %w", err)
	}
	request.Header.Set("Client-Id", ClientID)
	request.Header.Set("Authorization", "Bearer "+token.AccessToken)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return StreamInfo{}, fmt.Errorf("channel info: request failed: %w", err)
	}
	data, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	closeErr := response.Body.Close()
	if readErr != nil {
		return StreamInfo{}, fmt.Errorf("channel info: read response: %w", readErr)
	}
	if closeErr != nil {
		return StreamInfo{}, fmt.Errorf("channel info: close response: %w", closeErr)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return StreamInfo{}, fmt.Errorf("channel info: HTTP %d: %s", response.StatusCode, truncateText(string(data), 400))
	}

	var payload struct {
		Data []struct {
			Title    string `json:"title"`
			GameName string `json:"game_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return StreamInfo{}, fmt.Errorf("channel info: decode response: %w", err)
	}
	if len(payload.Data) == 0 {
		return StreamInfo{}, nil
	}
	return StreamInfo{
		Title:    cleanStreamText(payload.Data[0].Title),
		Category: cleanStreamText(payload.Data[0].GameName),
	}, nil
}
