package controllers

type MessageType int

const (
	SEND        MessageType = 0
	SEND_ERR    MessageType = 1
	NEW_MEMBER  MessageType = 2
	ERR         MessageType = 3
	MEMBER_LEFT MessageType = 4
)

type Message struct {
	Id        string  `json:"id"`
	Data      *string `json:"data"` // control messages do not have data
	Timestamp int64   `json:"timestamp"`
	From      string  `json:"from"`
	Username  string  `json:"username"`
}

func SendMessage(p *Packet, c *Client) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.open {
		// sender's client is destroyed before message is sent
		return
	}
	ses, err := SessionPool.Get(p.SessionID)
	if err != nil {
		p.Type = SEND_ERR
		c.Send <- p
		return
	}
	// send to all members in the session
	for _, member := range ses.clients {
		if member.CID != c.CID {
			member.SendMessage(p)
		}
	}
}
