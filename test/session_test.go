package test

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Aaron-json/WsSession/internal/controllers"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

var HOST string

func TestMain(m *testing.M) {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal(err)
	}
	HOST = fmt.Sprintf("127.0.0.1:%s", os.Getenv("PORT"))
	os.Exit(m.Run())
}

func TestCreateSession(t *testing.T) {
	nClients := 5
	conns := make([]*websocket.Conn, 0, nClients)
	url1 := url.URL{Scheme: "ws", Host: HOST, Path: "/new-session/TestSession"}
	ownerConn, _, err := websocket.DefaultDialer.Dial(url1.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	res := controllers.InitialRes{}
	err = ownerConn.ReadJSON(&res)
	if err != nil {
		t.Fatal(err)
	}
	conns = append(conns, ownerConn)
	for range nClients - 1 { // the owner is already in the session
		// create connections to join the connection
		url2 := url.URL{Scheme: "ws", Host: HOST, Path: fmt.Sprint("/join-session/", res.Code)}
		client, _, err := websocket.DefaultDialer.Dial(url2.String(), nil)
		if err != nil {
			t.Fatal(err)
		}
		conns = append(conns, client)
	}

	wg := &sync.WaitGroup{} // wait for listeners to close
	wg.Add(nClients)
	for _, conn := range conns {
		// in total listener should log [(nClients - 1) * messagesSent] messages for every message sent
		// by the writer. Messeges should reduce with every write since the connection is closed after we are
		// done writing to it.
		go Listen(conn, wg)
	}
	for i, conn := range conns {
		t.Log("Testing client: ", i+1)
		Write(conn, res.Code)
		// give readers time to print all the written messages they received
		time.Sleep(time.Second * 2)
	}
	wg.Wait()
}

func Listen(c *websocket.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		msgType, data, err := c.ReadMessage()
		if err != nil || msgType == websocket.CloseMessage {
			break
		}
		log.Println(string(data))
	}
}

func Write(c *websocket.Conn, code string) {
	defer func() {
		c.Close()
	}()
	msgTicker := time.NewTicker(time.Millisecond * 500)
	endTimer := time.NewTimer(time.Second * 4)
	for {
		select {
		case <-msgTicker.C:
			msg := "This is my message"
			p := controllers.Packet{
				SessionID: code,
				Type:      controllers.SEND,
				Message: controllers.Message{
					Timestamp: time.Now().Unix(),
					Data:      &msg,
					Id:        uuid.NewString(),
				},
			}
			c.WriteJSON(&p)
		case <-endTimer.C:
			msgTicker.Stop()
			return
		}
	}
}
