package client

import (
	"errors"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type MessageHandler func(*Client, *Packet)
type CloseHandler func(*Client)

type Client struct {
	mu            sync.RWMutex
	open          bool
	conn          *websocket.Conn
	sendCh        chan *Packet
	HandleClose   CloseHandler
	HandleMessage MessageHandler
}
type Packet struct {
	Type int
	Data []byte
}

const (
	TextMessage   = 1
	BinaryMessage = 2
	CloseMessage  = 8
	PingMessage   = 9
	PongMessage   = 10
)

var upgrader = &websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// If an error is encountered when creating client, the method writes an error
// to the http client automatically and returns a nil pointer and an error.
func NewClient(w http.ResponseWriter, r *http.Request) (*Client, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	r.Body.Close()
	if err != nil {
		return nil, err
	}
	c := &Client{
		conn:   conn,
		sendCh: make(chan *Packet, 15),
		open:   false,
		mu:     sync.RWMutex{},
	}
	return c, nil
}

// Attempts to close the connection as soon as possible.
// Called automatically when the session is closed or when a close message is
// sent to this client. Can also be invoked manually. End must NOT be called
// more than once.
func (c *Client) End() {
	// close connection and channel to make sure the listen and write
	// goroutines stop blocking.
	if !c.open {
		return
	}
	c.mu.Lock()
	c.open = false
	c.mu.Unlock()

	c.conn.Close()
	close(c.sendCh)

	if c.HandleClose != nil {
		c.HandleClose(c)
	}
}

func (c *Client) listener() {
	defer c.End()
	for {
		msgType, data, err := c.conn.ReadMessage()
		if err != nil || msgType == websocket.CloseMessage {
			return
		}
		p := &Packet{
			Type: msgType,
			Data: data,
		}
		if c.HandleMessage != nil {
			c.HandleMessage(c, p)
		}
	}
}

func (c *Client) writer() {
	for p := range c.sendCh {
		c.conn.WriteMessage(p.Type, p.Data)
	}
}

// Begins receiving from and writing to the connection. Takes an optional first message parameter.
// The message is sent before the client starts listening and accepting writing to the connection.
func (c *Client) Start(p *Packet) {
	if c.open {
		// already opened
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	go c.listener()
	go c.writer()
	if p != nil {
		c.sendCh <- p
	}
	c.open = true
}

// This method either sends the whole message, or nothing. ie on successful write, n == len(p).
// Messeges sent before Start() or after End() methods will return an error and len == 0 and will be ignored.
func (c *Client) Send(p *Packet) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.open {
		c.sendCh <- p
		return nil
	}
	return errors.New("client has been closed")
}
