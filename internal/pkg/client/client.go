package client

import (
	"errors"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Represents a connection to a user.
// Do not copy
type Client struct {
	// protect change of client state.
	mu sync.RWMutex
	// state of the client. writes to a closed channel are ignored and produce an error.
	open          bool
	SID           string
	CID           string
	conn          *websocket.Conn
	send          chan []byte
	handleClose   func(*Client)
	handleMessage func(*Client, []byte)
}
type ClientConfig struct {
	// used to notify  when this client is closed
	HandleClose func(*Client)
	// Called on on every message received by this client.
	HandleMessage func(*Client, []byte)
	// session id
	SID        string
	InitialMsg []byte
}

// If an error is encountered when creating client, the method writes an error
// to the http client automatically and returns a nil pointer and an error.
func NewClient(w http.ResponseWriter, r *http.Request, conf ClientConfig) (*Client, error) {
	var upgrader = &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	r.Body.Close()
	if err != nil {
		return nil, err
	}
	c := &Client{
		conn:          conn,
		CID:           uuid.NewString(),
		SID:           conf.SID,
		send:          make(chan []byte, 15),
		open:          false,
		mu:            sync.RWMutex{},
		handleClose:   conf.HandleClose,
		handleMessage: conf.HandleMessage,
	}
	return c, nil
}

// Attempts to close the client as soon as possible.
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
	close(c.send)

	c.handleClose(c)
}

func (c *Client) listener() {
	defer c.End()
	for {
		msgType, data, err := c.conn.ReadMessage()
		if err != nil || msgType == websocket.CloseMessage {
			return
		}
		c.handleMessage(c, data)
	}
}

func (c *Client) writer() {
	for p := range c.send {
		c.conn.WriteMessage(websocket.TextMessage, p)
	}
}

// Begins receiving from and writing to the connection. Takes an optional first message parameter.
// The message is sent before the client starts listening and accepting writing to the connection.
// Useful for performing an action before other messages can be sent and received. ex. sending an initial control message.
func (c *Client) Start(msg []byte) {
	if c.open {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	go c.listener()
	go c.writer()
	// send initial
	if msg != nil {
		c.send <- msg
	}
	c.open = true
}

// Implements the Writer interface.
// This method either sends the whole message, or nothing. ie on successful write, n == len(p).
// Messeges sent before Start() or after End() methods will return an error and len == 0 and will be ignored.
func (c *Client) Write(p []byte) (n int, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.open {
		c.send <- p
		return len(p), nil
	}
	return 0, errors.New("client has been closed")
}
