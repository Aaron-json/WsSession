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
	"github.com/google/uuid"
)

type HandshakeMessage struct {
	SessionName string   `json:"sessionName,omitempty"`
	SessionCode string   `json:"sessionCode,omitempty"`
	StatusCode  int      `json:"statusCode"`
	Status      string   `json:"status"`
	Members     []string `json:"members,omitempty"`
}
type Session struct {
	Code    string
	Name    string
	Members []Member
	mu      sync.RWMutex
}

type Member struct {
	Id      string
	Session *Session
	Client  *client.Client
}

const (
	MAX_SESSIONS          int = 500
	MAX_USERS_PER_SESSION int = 5
)

var SessionPool = pool.NewPool[string, *Session](MAX_SESSIONS)

func CreateNewSession(w http.ResponseWriter, r *http.Request) {
	sessionName := r.PathValue("sessionName")

	c, err := client.NewClient(w, r)
	if err != nil {
		// handshake unsuccessful.
		return
	}
	newSession := NewSession(sessionName, "") // have not chosen code yet
	newMember := NewMember(newSession, c)
	newSession.Members = append(newSession.Members, newMember)
	res := HandshakeMessage{}
	var sessionCode string
	for {
		// find unused code
		sessionCode = code.Generate(7)
		err = SessionPool.Store(sessionCode, newSession)
		if err == nil {
			newSession.Code = sessionCode
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
		return
	}
	c.Start(&client.Packet{
		Type: client.TextMessage,
		Data: resJson,
	})
	if res.StatusCode != 200 {
		// give user time to read the control message
		time.Sleep(time.Second * 2)
		c.End()
	}
}

func JoinSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionID")
	c, err := client.NewClient(w, r)
	if err != nil { // handshake failed
		return
	}
	handshakeMsg := HandshakeMessage{}
	var newMember Member
	ses, err := SessionPool.Get(sessionID)
	if err != nil {
		if err == pool.KEY_NOT_FOUND {
			handshakeMsg.StatusCode = 404
			handshakeMsg.Status = "Could not find session"
		} else {
			handshakeMsg.StatusCode = 500
			handshakeMsg.Status = "Server Error"
		}
	} else {
		ses.mu.Lock()
		if len(ses.Members) == MAX_USERS_PER_SESSION {
			handshakeMsg.StatusCode = 507
			handshakeMsg.Status = "Session is full"
		} else {
			// no error when retrieving session
			currentMembers := getMemberIds(ses)
			newMember = NewMember(ses, c)
			ses.Members = append(ses.Members, newMember)
			// no error adding to pool
			handshakeMsg.StatusCode = 200
			handshakeMsg.Status = "Success"
			handshakeMsg.Members = currentMembers
			handshakeMsg.SessionName = ses.Name
		}
		ses.mu.Unlock()
	}

	resJson, err := json.Marshal(&handshakeMsg)
	if err != nil {
		c.End()
		return
	}
	c.Start(&client.Packet{
		Type: client.TextMessage,
		Data: resJson,
	})
	if handshakeMsg.StatusCode != 200 {
		// close connection after timeout if error occured
		time.Sleep(time.Second * 2)
		c.End()
		return
	}
	BroadcastJSON(newMember, &ControlMessage{
		Control: MEMBER_JOINED,
		From:    newMember.Id,
	})
}

func RemoveMemberFromSession(mem Member) error {
	mem.Session.mu.Lock()
	defer mem.Session.mu.Unlock()
	prevLen := len(mem.Session.Members)
	mem.Session.Members = slices.DeleteFunc(mem.Session.Members, func(v Member) bool {
		return v.Id == mem.Id
	})
	if prevLen == len(mem.Session.Members) {
		return errors.New("member not found")
	}
	if len(mem.Session.Members) == 0 {
		SessionPool.Delete(mem.Session.Code)
		return nil
	}
	return nil
}

func HandleClientClose(mem Member) error {
	err := RemoveMemberFromSession(mem)
	if err != nil {
		return err
	}
	msg := ControlMessage{
		From:    mem.Id,
		Control: MEMBER_LEFT,
	}
	BroadcastJSON(mem, &msg)
	return nil
}

func NewSession(name string, code string) *Session {
	ses := &Session{
		Code:    code,
		Name:    name,
		Members: make([]Member, 0, MAX_USERS_PER_SESSION),
		mu:      sync.RWMutex{},
	}
	return ses
}

// Creates a new member object. Does not add the member to the session or make any modification
// to the input.
func NewMember(ses *Session, c *client.Client) Member {
	mem := Member{
		Id:      uuid.NewString(),
		Session: ses,
		Client:  c,
	}
	mem.Client.HandleMessage = func(c *client.Client, p *client.Packet) {
		HandleMessage(mem, p)
	}
	mem.Client.HandleClose = func(c *client.Client) {
		HandleClientClose(mem)
	}
	return mem
}

// Gets the ids of the members in a session. Synchronisation is left up to the
// caller/.
func getMemberIds(ses *Session) []string {
	memberIds := make([]string, len(ses.Members))
	for i := range len(ses.Members) {
		memberIds[i] = ses.Members[i].Id
	}
	return memberIds
}
