package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/textproto"
	"strings"
	"time"
)

// There exist plenty of IRC libraries, but I don't like any of the ones I have run into. Roll our own scuffed BS
// partly because NIH is fun, and partly to learn more about IRC.

// IRC manages a connection to the twitch IRC servers. If the connection is lost, you can simply reconnect, there is no need
type IRC struct {
	LastMsg        time.Time

	OnMessage func(*IRCMsg) // Called for all incoming messages except PING

	messages chan sendPair
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
	defer conn.Close()

	buffconn := bufio.NewReader(conn)
	proto := textproto.NewReader(buffconn)

	// Start write loop handler
	done := make(chan bool)
	go func() {
		if irc.messages == nil {
			irc.messages = make(chan sendPair)
		}

		buf := new(bytes.Buffer)
		for {
			select {
			case <-done:
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
			close(done)
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
				close(done)
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

// Send
func (irc *IRC) Send(msg *IRCMsg) error {
	if irc.messages == nil {
		irc.messages = make(chan sendPair)
	}

	v := sendPair{msg, make(chan error, 1)}
	irc.messages <- v
	return <-v.Conf
}

func (irc *IRC) Say(channel, msg string) error {
	return irc.Send(&IRCMsg{Command: "PRIVMSG", Params: []string{channel, msg}})
}

type IRCMsg struct {
	Command string
	Params []string
	Raw string
}

func parseMsg(line string) *IRCMsg {
	parts := strings.Split(line, " ")
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
