package main

import (
	"fmt"
)

// buildDecisionState mirrors the probe's state shape exactly. Clef is sensitive
// to this shape: keep chat_so_far as strings, keep latest_message as a small
// object, and keep stream context as flat scalars (no nested channel object).
func buildDecisionState(cfg *AIConfig, window []ChatMessage, trigger ChatMessage) map[string]any {
	chatSoFar := make([]string, 0, len(window))
	for _, message := range window {
		if message.Seq == trigger.Seq {
			continue
		}
		chatSoFar = append(chatSoFar, fmt.Sprintf("[%s] %s: %s", message.Time.Format("15:04"), message.DisplayName, message.Text))
	}

	state := map[string]any{
		"bot":         cfg.BotName,
		"streamer":    cfg.Streamer,
		"chat_so_far": chatSoFar,
		"latest_message": map[string]string{
			"time":   trigger.Time.Format("15:04"),
			"author": trigger.DisplayName,
			"text":   trigger.Text,
		},
	}

	if cfg.Channel.Game != "" {
		state["stream_game"] = cfg.Channel.Game
	}
	if cfg.Channel.Title != "" {
		state["stream_title"] = cfg.Channel.Title
	}
	return state
}
