package main

import (
	"strings"
	"unicode"
)

// cleanChatReply turns model output into one safe Twitch chat line. Collapsing
// all whitespace is also what prevents a reply from injecting IRC line breaks.
func cleanChatReply(text, botName string, maxChars int) string {
	cleaned := strings.Join(strings.Fields(text), " ")
	cleaned = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, cleaned)
	cleaned = strings.Join(strings.Fields(cleaned), " ")

	prefix := strings.ToLower(botName) + ":"
	if strings.HasPrefix(strings.ToLower(cleaned), prefix) {
		cleaned = strings.TrimSpace(cleaned[len(prefix):])
	}

	runes := []rune(cleaned)
	if maxChars > 0 && len(runes) > maxChars {
		if maxChars == 1 {
			return string(runes[:1])
		}
		cleaned = string(runes[:maxChars-1]) + "…"
	}
	return cleaned
}
