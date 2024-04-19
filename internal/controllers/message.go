package controllers

import (
	"encoding/json"

	"github.com/Aaron-json/WsSession/internal/pkg/client"
)

const (
	MEMBER_JOINED = 0
	MEMBER_LEFT   = 1
)

type ControlMessage struct {
	Control int    `json:"controlValue"`
	From    string `json:"from"` // member that generated the control message ex. member that left
}
type Message struct {
	Data string `json:"data,omitempty"`
	From string `json:"from"`
}

func HandleMessage(Sender Member, p *client.Packet) {
	if p.Type == client.TextMessage {
		// text messages are modified to include a "from" field
		msg := Message{}
		err := json.Unmarshal(p.Data, &msg)
		if err != nil {
			return
		}
		// add from field
		msg.From = Sender.Id
		BroadcastJSON(Sender, &msg)
	} else {
		// binary packets and custom message types are transmitted as is
		Broadcast(Sender, p)
	}
}

// Sends a message to all other clients in the session. Handles synchronization
// using a read lock on the session.
func Broadcast(Sender Member, p *client.Packet) {
	Sender.Session.mu.RLock()
	defer Sender.Session.mu.RUnlock()
	BroadcastUnsync(Sender, p)
}

// Broadcasts message to members without any synchronization. The caller is responsible
// for handling synchronization. Most callers should consider using Broadcast() unless
// they need to broadcast within an existing lock.
func BroadcastUnsync(Sender Member, p *client.Packet) {
	for _, member := range Sender.Session.Members {
		if member.Id != Sender.Id {
			member.Client.Send(p)
		}
	}
}

// Covnenience wrapper around Broadcast() that marshals the given value into JSON and broadcasts it.
func BroadcastJSON(Sender Member, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	Broadcast(Sender, &client.Packet{Data: data, Type: client.TextMessage})
	return nil
}

// Covnenience wrapper around BroadcastUnsync() that marshals the given value into JSON and broadcasts it
// as a text message without synchronization. Most callers should consider using BroadcastJSON().
func BroadcastUnsyncJSON(Sender Member, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	BroadcastUnsync(Sender, &client.Packet{Data: data, Type: client.TextMessage})
	return nil
}
