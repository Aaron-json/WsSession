package controllers

import (
	"errors"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Represents a connection to a user
type Client struct {
	// id of the session we are currently listening to
	SID  string
	Conn *websocket.Conn
	// id of the client (specific member)
	CID  string
	Send chan *Packet
	open bool
	mu   sync.RWMutex
}

// If an error is encountered when creating client, the method writes an error
// to the http client automatically.
func NewClient(w http.ResponseWriter, r *http.Request, sessionID string) (*Client, error) {

	var upgrader = &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	r.Body.Close()
	if err != nil {
		return nil, err
	}
	c := &Client{
		Conn: conn,
		CID:  uuid.NewString(),
		SID:  sessionID,
		Send: make(chan *Packet, 15),
		open: false,
		mu:   sync.RWMutex{},
	}
	return c, nil
}

func (c *Client) End() error {
	// close connection and channel to make sure the listen and write
	// goroutines stop blocking.
	c.mu.Lock()
	c.open = false
	c.Conn.Close()
	close(c.Send)
	c.mu.Unlock()
	ses, err := SessionPool.Get(c.SID)
	if err != nil {
		return errors.New("Session does not exist")
	}
	idx := slices.IndexFunc(ses.clients, func(v *Client) bool {
		return v.CID == c.CID
	})
	if idx == -1 {
		return errors.New("Client is not in the session")
	}
	if len(ses.clients) == 1 {
		SessionPool.Delete(c.SID)
		return nil
	}
	ses.clients = slices.Delete(ses.clients, idx, idx+1)
	SessionPool.Update(c.SID, ses)
	for _, member := range ses.clients {
		member.SendMessage(&Packet{
			SessionID: member.SID,
			Type:      MEMBER_LEFT,
			Message: Message{
				Id:        uuid.NewString(),
				Timestamp: time.Now().Unix(),
				From:      member.CID,
				Data:      nil,
			},
		})
	}
	return nil
}

func (c *Client) Read() {
	defer c.End()
	for {
		p := Packet{}
		err := c.Conn.ReadJSON(&p)
		if err != nil {
			// TODO: send back the packet to the user with error field set
			break
		}
		switch p.Type {
		case SEND:
			SendMessage(&p, c)
		}
	}
}

func (c *Client) Start() {
	c.open = true
	go c.Read()
	go c.Write()
}

func (c *Client) Write() {
	for p := range c.Send {
		c.Conn.WriteJSON(p)
	}
}

func (c *Client) SendMessage(p *Packet) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.open {
		c.Send <- p
	}
}
