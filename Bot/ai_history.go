package main

import (
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type ChatMessage struct {
	Seq           uint64 // assigned by aiHistory.Append
	Login         string
	DisplayName   string
	Text          string
	Time          time.Time
	IsBot         bool
	IsBroadcaster bool
}

type aiHistory struct {
	mu   sync.RWMutex
	seq  uint64
	msgs []ChatMessage
}

func newAIHistory() *aiHistory {
	return &aiHistory{}
}

// Append assigns a sequence number, stores the message, and trims the hard
// storage cap. The returned copy is the one callers should enqueue.
func (h *aiHistory) Append(message ChatMessage) ChatMessage {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.seq++
	message.Seq = h.seq
	h.msgs = append(h.msgs, message)
	if len(h.msgs) > 2000 {
		h.msgs = h.msgs[len(h.msgs)-2000:]
	}
	return message
}

// Window returns messages up to and including trigger, inside the rolling time
// window, capped by count and rendered size. The trigger is always kept.
func (h *aiHistory) Window(trigger ChatMessage, minutes float64, maxMessages, maxChars int) []ChatMessage {
	h.mu.RLock()
	defer h.mu.RUnlock()

	cutoff := trigger.Time.Add(-time.Duration(minutes * float64(time.Minute)))
	selected := make([]ChatMessage, 0, maxMessages)
	for _, message := range h.msgs {
		if message.Seq > trigger.Seq {
			continue
		}
		if message.Time.Before(cutoff) {
			continue
		}
		selected = append(selected, message)
	}
	if len(selected) > maxMessages {
		selected = selected[len(selected)-maxMessages:]
	}

	// Trim from the oldest end by rough rendered size, always keeping the
	// trigger (the newest message in the selection).
	trimmed := make([]ChatMessage, 0, len(selected))
	total := 0
	for i := len(selected) - 1; i >= 0; i-- {
		message := selected[i]
		cost := utf8.RuneCountInString(message.DisplayName) + utf8.RuneCountInString(message.Text) + 12
		if len(trimmed) > 0 && total+cost > maxChars {
			break
		}
		trimmed = append(trimmed, message)
		total += cost
	}
	for left, right := 0, len(trimmed)-1; left < right; left, right = left+1, right-1 {
		trimmed[left], trimmed[right] = trimmed[right], trimmed[left]
	}
	return trimmed
}

func (h *aiHistory) FormatTranscript(messages []ChatMessage) string {
	var builder strings.Builder
	for i, message := range messages {
		if i > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString("[")
		builder.WriteString(message.Time.Format("15:04"))
		builder.WriteString("] ")
		builder.WriteString(message.DisplayName)
		builder.WriteString(": ")
		builder.WriteString(message.Text)
	}
	return builder.String()
}

func (h *aiHistory) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.msgs = nil
}

func (h *aiHistory) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.msgs)
}
