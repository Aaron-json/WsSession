package controllers

import (
	"encoding/json"

	"github.com/Aaron-json/WsSession/internal/pkg/client"
)

type ControlValue int

const (
	SEND_ERR      ControlValue = 0
	MEMBER_JOINED ControlValue = 1
	ERR           ControlValue = 2
	MEMBER_LEFT   ControlValue = 3
)

type ControlMessage struct {
	Control ControlValue `json:"controlValue"`
	From    string       `json:"from"` // member that generated the control message ex. member that left
}
type Message struct {
	Data string `json:"data,omitempty"`
	From string `json:"from"`
}

func HandleMessage(Sender Member, p *client.Packet) {
	if p.Type == client.BinaryMessage {
		// binary packets are transmitted as is
		Broadcast(Sender, p)
	} else if p.Type == client.TextMessage {
		msg := Message{}
		err := json.Unmarshal(p.Data, &msg)
		if err != nil {
			return
		}
		msg.From = Sender.Id
		data, err := json.Marshal(&msg)
		if err != nil {
			return
		}
		Broadcast(Sender, &client.Packet{
			Data: data,
			Type: client.TextMessage,
		})
	}
}

// Sends a message to all other clients in the session
func Broadcast(Sender Member, p *client.Packet) {
	Sender.Session.mu.RLock()
	defer Sender.Session.mu.RUnlock()
	for _, member := range Sender.Session.Members {
		if member.Id != Sender.Id {
			member.Client.Send(p)
		}
	}
}

// Covnenience method that marshals the given value into JSON and broadcasts.
func BroadcastText(Sender Member, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	Broadcast(Sender, &client.Packet{Data: data, Type: client.TextMessage})
	return nil
}
