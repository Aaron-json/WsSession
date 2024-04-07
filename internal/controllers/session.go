package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/Aaron-json/WsSession/internal/pkg/client"
	"github.com/Aaron-json/WsSession/internal/pkg/code"
	"github.com/Aaron-json/WsSession/internal/pkg/pool"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type HandshakeRes struct {
	SessionCode string `json:"sessionCode,omitempty"`
	StatusCode  int    `json:"statusCode"`
	Status      string `json:"status"`
}
type Session struct {
	Name    string
	clients []*client.Client
	mu      sync.RWMutex
}

const (
	MAX_SESSIONS          int = 500
	MAX_USERS_PER_SESSION int = 5
)

var SessionPool = pool.NewPool[string, *Session](MAX_SESSIONS)

func CreateNewSession(w http.ResponseWriter, r *http.Request) {
	sessionName := chi.URLParam(r, "sessionName")

	c, err := client.NewClient(w, r, client.ClientConfig{
		// init without sesssion code
		HandleClose:   func(c *client.Client) { HandleClientClose(c) },
		HandleMessage: HandleMessage,
	})
	if err != nil {
		// handshake unsuccessful
		return
	}
	newSession := &Session{
		Name:    sessionName,
		clients: append(make([]*client.Client, 0, MAX_USERS_PER_SESSION), c),
		mu:      sync.RWMutex{},
	}
	res := HandshakeRes{}
	var sessionCode string
	for {
		// find unused code
		sessionCode = code.Generate(7)
		err = SessionPool.Store(sessionCode, newSession)
		if err == nil {
			c.SID = sessionCode
			res.SessionCode = sessionCode
			res.StatusCode = 200
			res.Status = "Success"
			break
		} else if err == pool.DUPLICATE_KEY {
			continue
		} else if err == pool.MAX_CAPACITY {
			res.StatusCode = 507
			res.Status = "Server is full"
			break
		} else {
			res.StatusCode = 500
			res.Status = "Server Error"
			break
		}
	}

	resJson, err := json.Marshal(&res)
	if err != nil {
		c.End()
	} else if res.StatusCode != 200 {
		c.Start(resJson)
		// give user time to read the control message
		time.Sleep(time.Second * 5)
		c.End()
	} else {
		c.Start(resJson)
	}
}

func JoinSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")
	c, err := client.NewClient(w, r, client.ClientConfig{
		HandleClose:   func(c *client.Client) { HandleClientClose(c) },
		HandleMessage: HandleMessage,
		SID:           sessionID,
	})
	if err != nil { // handshake failed
		return
	}
	res := HandshakeRes{}
	ses, err := SessionPool.Get(sessionID)
	if err != nil {
		if err == pool.KEY_NOT_FOUND {
			res.StatusCode = 404
			res.Status = "Could not find session"
		} else {
			res.StatusCode = 500
			res.Status = "Server Error"
		}
	} else {
		ses.mu.Lock()
		if len(ses.clients) == MAX_USERS_PER_SESSION {
			res.StatusCode = 507
			res.Status = "Session is full"
		} else {
			// no error when retrieving session
			ses.clients = append(ses.clients, c)
			// no error adding to pool
			res.StatusCode = 200
			res.Status = "Success"
		}
		ses.mu.Unlock()
	}

	resJson, err := json.Marshal(&res)
	if err != nil {
		c.End()
	} else if res.StatusCode != 200 {
		c.Start(resJson)
		// give user time to read the control message
		time.Sleep(time.Second * 5)
		c.End()
	} else {
		c.Start(resJson)
	}
}

func HandleClientClose(c *client.Client) error {
	ses, err := SessionPool.Get(c.SID)
	if err != nil {
		return errors.New("Session does not exist")
	}
	ses.mu.Lock()
	defer ses.mu.Unlock()
	idx := slices.IndexFunc(ses.clients, func(v *client.Client) bool {
		return v.CID == c.CID
	})
	if idx == -1 {
		return errors.New("client is not in the session")
	}
	if len(ses.clients) == 1 {
		SessionPool.Delete(c.SID)
		return nil
	}
	ses.clients = slices.Delete(ses.clients, idx, idx+1)
	p := &Packet{
		SessionID: c.SID,
		Type:      MEMBER_LEFT,
		Message: Message{
			Id:        uuid.NewString(),
			Timestamp: time.Now().Unix(),
			From:      c.CID,
			Data:      nil,
		},
	}
	for _, member := range ses.clients {
		if member.CID != c.CID {
			json.NewEncoder(member).Encode(p)
		}
	}
	return nil
}
