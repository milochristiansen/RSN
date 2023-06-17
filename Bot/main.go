package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	"fmt"
	"io/ioutil"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/twitch"

	"github.com/milochristiansen/sessionlogger"

	esb "github.com/dnsge/twitch-eventsub-bindings"
	esf "github.com/dnsge/twitch-eventsub-framework"
)

const (
	Channel = "#milochristiansen"
	ChannelID = "216915573"
	BotName = "mechanicalmilo"
)

func main() {
	l := sessionlogger.NewMasterLogger()

	l.I.Println("Bot starting...")
	
	client := new(IRC)

	l.I.Println("Creating message handler.")
	client.OnMessage = func(i *IRCMsg) {
		fmt.Println(i.Raw)

		if i.Command == "PRIVMSG" {
			switch i.Params[len(i.Params)-1] {
			case "!discord":
				client.Say(Channel, "Get in the Discord for memes, alerts, and random BS: https://discord.gg/ayrXgwNTJR")
			case "!steam":
				client.Say(Channel, "My Steam URL: https://steamcommunity.com/id/MiloChristiansen/")
			case "!bot":
				client.Say(Channel, "Yes, I'm the bot. Yes, I'm custom. Yes, I'm probably buggy as hell.")
			}
		}
	}

	l.I.Println("Starting timer loop.")
	go func(){
		l := sessionlogger.NewSessionLogger("timer-loop")
		lastPlug := time.Now().Add(-(time.Minute * 30))
		for {
			time.Sleep(time.Minute)
			// If the last message was within a few minutes:
			if time.Now().Sub(client.LastMsg) < (time.Minute * 10) {
				// If the last plug was more than an hour ago.
				if time.Now().Sub(lastPlug) > time.Hour {
					l.I.Println("Sending Discord plug.")
					client.Say(Channel, "Join the Discord for memes, live notifications, game suggestions, and general craziness: https://discord.gg/ayrXgwNTJR")
					lastPlug = time.Now()
				}
			}
		}
	}()

	go func() {
		l := sessionlogger.NewSessionLogger("irc")
		l.I.Println("Starting connection loop.")
		for {
			tkn, err := getToken("token-bot.json")
			if err != nil {
				l.E.Println(err)
				l.I.Println("Sleeping for 1m")
				time.Sleep(time.Minute)
				continue
			}

			err = client.Connect(BotName, "oauth:"+tkn, Channel)
			if err != nil {
				l.E.Println(err)
				l.I.Println("Sleeping for 1m")
				time.Sleep(time.Minute)
				continue
			}
		}
	}()

	var desiredSubs = map[string]any{
		"channel.channel_points_custom_reward_redemption.add": esb.ConditionChannelPointsRewardRedemptionAdd{BroadcasterUserID: ChannelID},
		"channel.cheer": esb.ConditionChannelCheer{BroadcasterUserID: ChannelID},
		"channel.follow": map[string]any{"broadcaster_user_id": ChannelID, "moderator_user_id": ChannelID},
		"channel.raid": esb.ConditionChannelRaid{ToBroadcasterUserID: ChannelID},
		"channel.subscribe": esb.ConditionChannelSubscribe{BroadcasterUserID: ChannelID},
		"channel.subscription.gift": esb.ConditionChannelSubscriptionGift{BroadcasterUserID: ChannelID},
		"channel.subscription.message": esb.ConditionChannelSubscriptionMessage{BroadcasterUserID: ChannelID},
		"stream.offline": esb.ConditionStreamOffline{BroadcasterUserID: ChannelID},
		"stream.online": esb.ConditionStreamOnline{BroadcasterUserID: ChannelID},
	}
	// If nothing is listed, uses "1"
	var subVersions = map[string]string{
		"channel.follow": "2",
	}

	// Get a list of all current eventsub subscriptions, then go down the list unsubscribing anything we don't want
	// and finally subscribing anything we want but don't have.
	l.I.Println("Managing event subscriptions.")
	tkn, err := AppToken.Token()
	if err != nil {
		l.E.Println("EventSub Error:", err)
		os.Exit(1)
	}
	esclient := esf.NewSubClient(esf.NewStaticCredentials(ClientID, tkn.AccessToken))
	subs, err := esclient.GetSubscriptions(context.Background(), esf.StatusAny)
	if err != nil {
		l.E.Println("EventSub Error:", err)
		os.Exit(1)
	}
	for _, sub := range subs.Data {
		if _, ok := desiredSubs[sub.Type]; ok {
			// We have it and we want it!
			l.I.Printf("Already subscribed to %v event.\n", sub.Type)
			desiredSubs[sub.Type] = nil
		} else {
			// We have it and we don't want it :(
			l.I.Printf("Unsubscribing from %v event.\n", sub.Type)
			err := esclient.Unsubscribe(context.Background(), sub.ID)
			if err != nil {
				l.E.Println("EventSub Unsubscribe Error:", err)
			}
		}
	}
	for sub, cond := range desiredSubs {
		if cond != nil {
			// We want it and we don't have it
			l.I.Printf("Subscribing to %v event.\n", sub)
			_, err := esclient.Subscribe(context.Background(), &esf.SubRequest{
				Type: sub,
				Secret: EventSubSecret,
				Callback: "https://httpscolonslashslashwww.com/twitch/webhook",
				Condition: cond,
				Version: subVersions[sub],
			})
			if err != nil {
				l.E.Println("EventSub Subscribe Error:", err)
			}
		}
	}

	// Handle event webhooks.
	l.I.Println("Creating webhook handlers.")
	handler := esf.NewSubHandler(true, []byte(EventSubSecret))
	handler.HandleChannelSubscribe = func(h *esb.ResponseHeaders, event *esb.EventChannelSubscribe) {
		l := sessionlogger.NewSessionLogger("webhook-sub")
		if event.IsGift {
			// I don't care who got a gift. Screw them. Kappa
			return
		}
		SendEventMsg("sub", map[string]any{"Name": event.UserName, "Months": 0})
		l.I.Printf("%v just subscribed!\n", event.UserName)
	}
	handler.HandleChannelSubscriptionGift = func(h *esb.ResponseHeaders, event *esb.EventChannelSubscriptionGift) {
		l := sessionlogger.NewSessionLogger("webhook-giftsub")
		SendEventMsg("gift", map[string]any{"Name": event.UserName, "Count": event.Total})
		l.I.Printf("%v just gifted %v subs!\n", event.UserName, event.Total)
	}
	handler.HandleChannelSubscriptionMessage = func(h *esb.ResponseHeaders, event *esb.EventChannelSubscriptionMessage) {
		l := sessionlogger.NewSessionLogger("webhook-resub")
		SendEventMsg("sub", map[string]any{"Name": event.UserName, "Months": event.DurationMonths})
		l.I.Printf("%v just resubscribed for %v months!\n", event.UserName, event.DurationMonths)
	}
	handler.HandleChannelCheer = func(h *esb.ResponseHeaders, event *esb.EventChannelCheer) {
		l := sessionlogger.NewSessionLogger("webhook-cheer")
		SendEventMsg("bits", map[string]any{"Name": event.UserName, "Bits": event.Bits})
		l.I.Printf("%v just cheered %v bits!\n", event.UserName, event.Bits)
	}
	handler.HandleChannelFollow = func(h *esb.ResponseHeaders, event *esb.EventChannelFollow) {
		l := sessionlogger.NewSessionLogger("webhook-follow")
		client.Say(Channel, fmt.Sprintf("Thank you for the follow %v!", event.UserName))
		SendEventMsg("follow", map[string]any{"Name": event.UserName})
		l.I.Printf("%v just followed!\n", event.UserName)
	}
	handler.HandleChannelRaid = func(h *esb.ResponseHeaders, event *esb.EventChannelRaid) {
		l := sessionlogger.NewSessionLogger("webhook-raid")
		SendEventMsg("raid", map[string]any{"Name": event.FromBroadcasterUserName, "Viewers": event.Viewers})
		l.I.Printf("%v just arrived with %v raiders!\n", event.FromBroadcasterUserName, event.Viewers)
	}
	handler.HandleChannelPointsRewardRedemptionAdd = func(h *esb.ResponseHeaders, event *esb.EventChannelPointsRewardRedemptionAdd) {
		l := sessionlogger.NewSessionLogger("webhook-redeem")
		SendEventMsg("points", map[string]any{"Name": event.UserName, "Reward": event.Reward.Title})
		l.I.Printf("%v just redeemed %v!\n", event.UserName, event.Reward.Title)
	}

	// Maybe use these to stop/start the IRC bot sending messages?
	handler.HandleStreamOnline = func(h *esb.ResponseHeaders, event *esb.EventStreamOnline) {}
	handler.HandleStreamOffline = func(h *esb.ResponseHeaders, event *esb.EventStreamOffline) {}

	http.Handle("/twitch/webhook", handler)
	http.HandleFunc("/twitch/sse", SSEHandler)

	l.I.Println("Starting HTTP server.")
	err = http.ListenAndServe(":80", nil)
	if err != nil {
		l.E.Println(err)
	}
}

// SSE
// =====================================================================================================================

var Broker = struct {
	Messages chan []byte // Incoming events to send to clients
	NewClients chan chan []byte // New client connections
	ClosingClients chan chan []byte // Closed client connections
	clients map[chan []byte]bool // Client connections registry
}{
	Messages:       make(chan []byte, 1),
	NewClients:     make(chan chan []byte),
	ClosingClients: make(chan chan []byte),
	clients:        make(map[chan []byte]bool),
}

func SendEventMsg(typ string, body any) error {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "event: %s\ndata: ", typ)
	err := json.NewEncoder(buf).Encode(body)
	if err != nil {
		return err
	}
	fmt.Fprintf(buf, "\n")
	Broker.Messages <- buf.Bytes()
	return nil
}

func init() {
	const dur = 10 * time.Second
	go func() {
		timeout := time.NewTicker(dur)
		for {
			select {
			case msg := <- Broker.Messages:
				// New message, send to all clients.
				for client := range Broker.clients {
					client <- msg
				}
				timeout.Reset(dur)
			case client := <- Broker.NewClients:
				Broker.clients[client] = true
				timeout.Reset(dur)
			case client := <- Broker.ClosingClients:
				delete(Broker.clients, client)
				timeout.Reset(dur)
			case <-timeout.C:
				for client := range Broker.clients {
					client <- []byte(": ping")
				}
			}
		}
	}()
}

func SSEHandler(w http.ResponseWriter, r *http.Request) {
	l := sessionlogger.NewSessionLogger("/twitch/sse")
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Event streaming unsupported.", http.StatusInternalServerError)
		l.E.Println("ResponseWriter doesn't support streaming.")
		return
	}

	messages := make(chan []byte, 1)
	Broker.NewClients <- messages
	defer func(){
		l.I.Println("Closing event send loop.")
		Broker.ClosingClients <- messages
	}()

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	l.I.Println("Starting event send loop.")
	for {
		select {
		case <-r.Context().Done():
			break
		case ev := <-messages:
			fmt.Fprintf(w, "%s\n\n", ev)
			f.Flush()
		}
	}
}

// Auth
// =====================================================================================================================

var AuthData = &struct{
	Config   oauth2.Config
	Context  context.Context
	Lock sync.Mutex
}{
	Context: context.Background(),
	Config: oauth2.Config{
		ClientID:     ClientID,
		ClientSecret: ClientSecret,
		Endpoint:     twitch.Endpoint,
		RedirectURL:  "http://localhost:3000", // TODO: Write in my own oauth endpoint instead of using the twitch CLI.
	},
}

var AppToken = (&clientcredentials.Config{
	ClientID:     ClientID,
	ClientSecret: ClientSecret,
	TokenURL: twitch.Endpoint.TokenURL,
}).TokenSource(context.Background())

// Three tokens:
// IRC Token: User token for MechanicalMilo account, has chat perms.
//	chat:edit chat:read
// "I need it but I don't use it" Token: User token for my account, has the perms for the EventSub events I need.
//	bits:read channel:read:redemptions channel:read:subscriptions moderator:read:followers
// Bot token, take 2: App token, no special perms

func getToken(path string) (string, error) {
	AuthData.Lock.Lock()
	defer AuthData.Lock.Unlock()

	tokenR, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	// Parse the token
	token := &oauth2.Token{}
	err = json.Unmarshal(tokenR, token)
	if err != nil {
		return "", err
	}

	// Refresh the token if needed.
	source := AuthData.Config.TokenSource(AuthData.Context, token)

	token, err = source.Token()
	if err != nil {
		return "", err
	}

	// Save the possibly refreshed token
	tokenB, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	ioutil.WriteFile(path, tokenB, 0666)
	return token.AccessToken, nil
}

