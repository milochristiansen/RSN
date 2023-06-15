module twitchbot

go 1.20

require (
	github.com/dnsge/twitch-eventsub-bindings v1.1.0
	github.com/dnsge/twitch-eventsub-framework v1.2.3
	github.com/gempir/go-twitch-irc/v4 v4.0.0
	github.com/gorilla/websocket v1.4.2
	github.com/milochristiansen/sessionlogger v0.0.0-20220826144442-e29e359dbf4f
	github.com/pajlada/go-twitch-pubsub v0.0.4
	golang.org/x/oauth2 v0.8.0
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/mozillazg/go-httpheader v0.3.0 // indirect
	github.com/teris-io/shortid v0.0.0-20201117134242-e59966efd125 // indirect
	golang.org/x/net v0.10.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)

replace github.com/pajlada/go-twitch-pubsub => /home/milo/Projects/Servers/TwitchBot/App/pubsub/
