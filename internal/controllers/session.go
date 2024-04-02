package controllers

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/Aaron-json/WsSession/internal/pkg/code"
	"github.com/Aaron-json/WsSession/internal/pkg/pool"
	"github.com/go-chi/chi/v5"
)

type Session struct {
	Name    string
	clients []*Client
}

var SessionPool = pool.NewPool[string, Session]()

const MAX_USERS_PER_SESSION = 5

func init() {
	go func() {
		ticker := time.NewTicker(time.Second).C
		for range ticker {
			fmt.Println("goroutines:", runtime.NumGoroutine(), "Sessions", SessionPool.Size())
		}
	}()
}

func CreateNewSession(w http.ResponseWriter, r *http.Request) {
	sessionName := chi.URLParam(r, "sessionName")
	type InitialRes struct {
		Code string `json:"code"`
	}
	code := code.Generate()

	c, err := NewClient(w, r, code)
	if err != nil {
		fmt.Println("creating client err", err)
		return
	}
	err = SessionPool.Store(code, Session{
		Name:    sessionName,
		clients: append(make([]*Client, 0, MAX_USERS_PER_SESSION), c),
	})
	if err != nil {
		c.Conn.Close()
		fmt.Println("Error adding new session to pool: ", err)
		return
	}
	c.Conn.WriteJSON(&InitialRes{
		Code: code,
	})
	c.Start()
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
	c, err := NewClient(w, r, sessionID)
	if err != nil { // NewClient() will write err message to response
		log.Println("Join Session: ", err)
		return
	}
	ses.clients = append(ses.clients, c)
	err = SessionPool.Update(sessionID, ses)
	if err != nil {
		log.Println("Could not join the session pool:", err)
		c.Conn.Close()
		return
	}
	c.Start()
}
