package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/Aaron-json/WsSession/internal/pkg/client"
	"github.com/Aaron-json/WsSession/internal/pkg/code"
	"github.com/Aaron-json/WsSession/internal/pkg/pool"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type InitialRes struct {
	Code   string `json:"code,omitempty"`
	Status string `json:"status"`
}
type Session struct {
	Name    string
	clients []*client.Client
}

var SessionPool = pool.NewPool[string, Session]()

const MAX_USERS_PER_SESSION = 5

func CreateNewSession(w http.ResponseWriter, r *http.Request) {
	sessionName := chi.URLParam(r, "sessionName")

	code := code.Generate()
	c, err := client.NewClient(w, r, client.ClientConfig{
		HandleClose:   func(c *client.Client) { HandleClientClose(c) },
		HandleMessage: HandleMessage,
		SID:           code,
	})
	if err != nil {
		return
	}
	// create the session in the pool
	err = SessionPool.Store(code, Session{
		Name:    sessionName,
		clients: append(make([]*client.Client, 0, MAX_USERS_PER_SESSION), c),
	})
	if err != nil {
		// could not add to pool.
		return
	}
	initialMsg, err := json.Marshal(&InitialRes{
		Code:   code,
		Status: "Opened",
	})
	if err != nil {
		return
	}
	c.Start(initialMsg)
}

func JoinSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")
	ses, err := SessionPool.Get(sessionID)
	if err != nil { // if exists
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Session does not exist"))
		return
	}
	if len(ses.clients) == MAX_USERS_PER_SESSION {
		w.WriteHeader(http.StatusInsufficientStorage)
		return
	}
	c, err := client.NewClient(w, r, client.ClientConfig{
		HandleClose:   func(c *client.Client) { HandleClientClose(c) },
		HandleMessage: HandleMessage,
		SID:           sessionID,
	})
	if err != nil { // NewClient() will write err message to response
		return
	}
	ses.clients = append(ses.clients, c)
	err = SessionPool.Update(sessionID, ses)
	if err != nil {
		return
	}
	initialMsg, err := json.Marshal(&InitialRes{
		Status: "Opened",
	})
	if err != nil {
		return
	}
	c.Start(initialMsg)
}

func HandleClientClose(c *client.Client) error {
	ses, err := SessionPool.Get(c.SID)
	if err != nil {
		return errors.New("Session does not exist")
	}
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
	SessionPool.Update(c.SID, ses)
	for _, member := range ses.clients {
		if member.CID == c.CID {
			continue
		}
		json.NewEncoder(member).Encode(&Packet{
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
