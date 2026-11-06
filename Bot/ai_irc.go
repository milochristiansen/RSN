package main

import (
	"strings"
	"time"
)

// chatMessageFromIRC adapts the existing IRC message to the AI layer. The IRC
// connection and parser are not modified: Raw keeps the source prefix and
// Params keeps the channel and text.
func chatMessageFromIRC(i *IRCMsg, now time.Time) (ChatMessage, bool) {
	if i == nil || i.Command != "PRIVMSG" || len(i.Params) < 2 {
		return ChatMessage{}, false
	}
	if !strings.EqualFold(i.Params[0], Channel) {
		return ChatMessage{}, false
	}
	text := strings.TrimSpace(i.Params[len(i.Params)-1])
	if text == "" {
		return ChatMessage{}, false
	}
	login := ircLogin(i.Raw)
	if login == "" {
		return ChatMessage{}, false
	}
	return ChatMessage{
		Login:         login,
		DisplayName:   login,
		Text:          text,
		Time:          now,
		IsBot:         strings.EqualFold(login, BotName),
		IsBroadcaster: strings.EqualFold(login, strings.TrimPrefix(Channel, "#")),
	}, true
}

// ircLogin reads the sender nick out of ":nick!nick@host". It tolerates an
// optional @tags prefix defensively, even though this bot does not request
// tags from Twitch.
func ircLogin(raw string) string {
	line := raw
	if strings.HasPrefix(line, "@") {
		if i := strings.IndexByte(line, ' '); i >= 0 {
			line = line[i+1:]
		}
	}
	if !strings.HasPrefix(line, ":") {
		return ""
	}
	rest := line[1:]
	if end := strings.IndexAny(rest, "!@ "); end > 0 {
		return rest[:end]
	}
	return ""
}
