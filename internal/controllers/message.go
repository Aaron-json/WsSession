package controllers

import (
	"encoding/json"

	"github.com/Aaron-json/WsSession/internal/pkg/client"
)

type MessageType int

const (
	SEND        MessageType = 0
	SEND_ERR    MessageType = 1
	NEW_MEMBER  MessageType = 2
	ERR         MessageType = 3
	MEMBER_LEFT MessageType = 4
)

type Packet struct {
	SessionID string      `json:"sessionID"`
	Message   Message     `json:"message"`
	Type      MessageType `json:"type"`
}

type Message struct {
	Id        string  `json:"id"`
	Data      *string `json:"data"` // control messages do not have data
	Timestamp int64   `json:"timestamp"`
	From      string  `json:"from"`
	Username  string  `json:"username"`
}

func HandleMessage(c *client.Client, data []byte) {
	p := Packet{}
	err := json.Unmarshal(data, &p)
	if err != nil {
		p.Type = SEND_ERR
		json.NewEncoder(c).Encode(&p)
		return
	}
	switch p.Type {
	case SEND:
		Broadcast(c, &p)
	}
}

// Sends a message to all other clients in the session
func Broadcast(c *client.Client, p *Packet) {
	ses, err := SessionPool.Get(p.SessionID)
	if err != nil {
		p.Type = SEND_ERR
		json.NewEncoder(c).Encode(p)
		return
	}
	ses.mu.RLock()
	defer ses.mu.RUnlock()
	// send to all members in the session
	for _, member := range ses.clients {
		if member.CID != c.CID {
			json.NewEncoder(member).Encode(p)
		}
	}
}
