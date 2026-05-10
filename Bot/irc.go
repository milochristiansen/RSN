package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/textproto"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// There exist plenty of IRC libraries, but I don't like any of the ones I have run into. Roll our own scuffed BS
// partly because NIH is fun, and partly to learn more about IRC.

// IRC manages a connection to the twitch IRC servers. If the connection is lost, you can simply reconnect, there is no need
type IRC struct {
	LastMsg        time.Time

	OnMessage func(*IRCMsg) // Called for all incoming messages except PING

	messages chan sendPair
	initMsgs sync.Once
	conn     net.Conn
	done     chan struct{}
	shutdown atomic.Uint32
}

const (
	ircAddr = "irc.chat.twitch.tv:6667"
)

// Connect to a twitch IRC server and attempt to authenticate. Does not return until the connection closes or is closed.
func (irc *IRC) Connect(id, token, channel string) error {
	conn, err := net.Dial("tcp", ircAddr)
	if err != nil {
		return err
	}

	// Reset the shutdown infrastructure.
	irc.shutdown.Store(0)
	irc.done = make(chan struct{})
	irc.conn = conn

	// And setup cleanup.
	defer irc.Shutdown()

	buffconn := bufio.NewReader(conn)
	proto := textproto.NewReader(buffconn)

	// Start write loop handler
	go func() {
		irc.initMessages()

		buf := new(bytes.Buffer)
		for {
			select {
			case <-irc.done:
				return
			case pair := <-irc.messages:
				buf.Reset()
				fmt.Fprintf(buf, "%s", pair.Msg.Command)
				for i := range pair.Msg.Params {
					if i == len(pair.Msg.Params) - 1 && strings.Contains(pair.Msg.Params[i], " ") {
						fmt.Fprintf(buf, " :%s", pair.Msg.Params[i])
						continue
					}
					fmt.Fprintf(buf, " %s", pair.Msg.Params[i])
				}

				_, err := fmt.Fprintf(conn, "%s\r\n", buf.Bytes())
				pair.Conf <- err
			}
		}
	}()

	// Send login messages
	irc.Send(&IRCMsg{Command: "PASS", Params: []string{token}})
	irc.Send(&IRCMsg{Command: "NICK", Params: []string{id}})
	irc.Send(&IRCMsg{Command: "JOIN", Params: []string{channel}})

	// Comm loop
	for {
		line, err := proto.ReadLine()
		if err != nil {
			return err
		}

		msg := parseMsg(line)
		if msg == nil {
			continue
		}

		switch msg.Command {
		case "PING":
			err := irc.Send(&IRCMsg{Command: "PONG", Params: msg.Params})
			if err != nil {
				return err
			}
		case "PRIVMSG":
			irc.LastMsg = time.Now()
			fallthrough
		default:
			if irc.OnMessage != nil {
				irc.OnMessage(msg)
			}
		}
	}
}

type sendPair struct {
	Msg *IRCMsg
	Conf chan error
}

// Send queues an IRCMsg and waits for it to be sent. Any error when sending is returned.
func (irc *IRC) Send(msg *IRCMsg) error {
	irc.initMessages()

	v := sendPair{msg, make(chan error, 1)}
	irc.messages <- v
	return <-v.Conf
}

// Say is a simple version of Send that handles the common PRIVMSG case.
func (irc *IRC) Say(channel, msg string) error {
	return irc.Send(&IRCMsg{Command: "PRIVMSG", Params: []string{channel, msg}})
}

func (irc *IRC) initMessages() {
	irc.initMsgs.Do(func() {
		irc.messages = make(chan sendPair)
	})
}

func (irc *IRC) Shutdown() {
	// Atomically transition from 0 (not shutdown) to 1 (shutdown).
	// If the value was already 1, another goroutine already shut down, so return.
	if !irc.shutdown.CompareAndSwap(0, 1) {
		return
	}
	if irc.done != nil {
		close(irc.done)
	}
	if irc.conn != nil {
		irc.conn.Close()
	}
}

type IRCMsg struct {
	Command string
	Params []string
	Raw string
}

func parseMsg(line string) *IRCMsg {
	parts := strings.Fields(line)
	index := 0

	// Just drop the tags if any exist, I don't care.
	if strings.HasPrefix(parts[index], "@") {
		index++
	}

	if index >= len(parts) {
		return nil
	}

	// Don't care about the source either.
	if strings.HasPrefix(parts[index], ":") {
		index++
	}

	if index >= len(parts) {
		return nil
	}

	// Now we are getting to the actual meat of the message.
	rtn := &IRCMsg{Raw: line}

	rtn.Command = parts[index]
	index++

	if index >= len(parts) {
		return rtn
	}

	var params []string
	for i, part := range parts[index:] {
		if strings.HasPrefix(part, ":") {
			part = strings.Join(parts[index+i:], " ")
			part = strings.TrimPrefix(part, ":")
			params = append(params, part)
			break
		}

		params = append(params, part)
	}
	rtn.Params = params

	return rtn
}
