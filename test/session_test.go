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
	url1 := url.URL{Scheme: "ws", Host: HOST, Path: "/new-session/My session"}
	client1, _, err := websocket.DefaultDialer.Dial(url1.String(), nil)
	type CodeRes struct {
		Code string `json:"code"`
	}
	if err != nil {
		t.Fatal(err)
	}
	code := CodeRes{}
	err = client1.ReadJSON(&code)

	if err != nil {
		t.Fatal(err)
	}

	url2 := url.URL{Scheme: "ws", Host: HOST, Path: fmt.Sprint("/join-session/", code.Code)}
	client2, _, err := websocket.DefaultDialer.Dial(url2.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go Write(client1, code.Code, wg)
	go Listen(client2, wg)
	wg.Wait()
}

func Listen(c *websocket.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		packet := controllers.Packet{}
		err := c.ReadJSON(&packet)
		if err != nil {
			break
		}
		if packet.Type == controllers.MEMBER_LEFT {
			c.Close()
		}
		log.Println(packet)
	}
}

func Write(c *websocket.Conn, code string, wg *sync.WaitGroup) {
	defer wg.Done()
	defer c.Close()
	ticker := time.NewTicker(time.Millisecond * 50)
	timer := time.NewTimer(time.Second * 5)
	for {
		select {
		case <-ticker.C:
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
		case <-timer.C:
			ticker.Stop()
			return
		}
	}
}
