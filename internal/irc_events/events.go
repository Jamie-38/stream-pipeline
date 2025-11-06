package ircevents

import "encoding/json"

type Event interface {
	Kind() string
	Key() string
	Marshal() ([]byte, error)
}

// type PrivMsg struct {
// 	UserID    string
// 	ChannelID string
// 	Text      string
// }

type PrivMsg struct {
	UserID       string // e.g., "464309918" (stable, from tags)
	UserLogin    string // e.g., "someviewer" (mutable, from prefix)
	ChannelID    string // e.g., "12345678" (stable, from tags)
	ChannelLogin string // e.g., "coolstreamer" (from params)
	Text         string
}

type JoinPart struct {
	UserID    string
	ChannelID string
	Op        string
}

func (msg PrivMsg) Kind() string {
	return "privmsg"
}

func (msg PrivMsg) Key() string {
	return msg.ChannelID
}

func (msg PrivMsg) Marshal() ([]byte, error) {
	return json.Marshal(msg)
}
