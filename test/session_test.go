package test

import (
	"bufio"
	"encoding/json"
	"errors"
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

type TestClient struct {
	conn *websocket.Conn
	id   string
}

func TestMain(m *testing.M) {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal(err)
	}
	HOST = fmt.Sprintf("127.0.0.1:%s", os.Getenv("PORT"))
	os.Exit(m.Run())
}

func TestSessions(t *testing.T) {
	nSessions := 200 // number of concurrent sessions
	testsWg := &sync.WaitGroup{}
	testsWg.Add(nSessions)
	loggersWg := &sync.WaitGroup{}
	loggersWg.Add(2)
	errCh := make(chan []byte, 10)
	outCh := make(chan []byte, 10)
	go Logger(errCh, "session_test_err.log", loggersWg)
	go Logger(outCh, "session_test.log", loggersWg)
	t.Log("Starting session test")
	for range nSessions {
		go testSession(outCh, errCh, testsWg)
	}
	t.Log("waiting for tests")
	testsWg.Wait()
	close(errCh)
	close(outCh)
	t.Log("waiting for loggers")
	// wait for error logger to write and close the file
	loggersWg.Wait()
}
func testSession(stdout chan []byte, stderr chan []byte, wg *sync.WaitGroup) {
	defer wg.Done()
	nClients := 5
	conns := make([]TestClient, 0, nClients)
	listenWg := &sync.WaitGroup{}
	ownerConn, code, err := createSession()
	if err != nil {
		stderr <- []byte(err.Error())
		return
	}
	listenWg.Add(1)
	go Listen(ownerConn, stdout, stderr, listenWg)
	conns = append(conns, ownerConn)
	// wait for connection listeners to be ready to close
	for range nClients - 1 { // owner is already in session
		// create connections to join the connection
		client, err := joinSession(code)
		if err != nil {
			stderr <- []byte(err.Error())
			continue
		}
		listenWg.Add(1)
		go Listen(client, stdout, stderr, listenWg)

		conns = append(conns, client)
	}
	for i, conn := range conns {
		stdout <- []byte(fmt.Sprint("Testing client: ", i+1))
		Write(conn, code, stdout, stderr)
		// wait for listeners to log messages
		time.Sleep(time.Second * 2)
	}
	listenWg.Wait()
}

func Logger(logCh chan []byte, filename string, wg *sync.WaitGroup) {
	defer wg.Done()
	fp, err := os.Create(filename)
	if err != nil {
		panic("Could not create log file.")
	}
	defer fp.Close()
	bw := bufio.NewWriter(fp)
	defer bw.Flush()
	for val := range logCh {
		bw.Write(val)
		bw.WriteByte('\n')
	}
}

func Listen(c TestClient, stdout chan []byte, stderr chan []byte, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			stderr <- []byte(fmt.Sprint("Error listening on client id: ", c.id, " Error: ", err.Error()))
			break
		}
		if msgType == websocket.CloseMessage {
			break
		}
		stdout <- data
	}
}

func Write(c TestClient, code string, stdout chan []byte, stderr chan []byte) {
	defer func() {
		c.conn.Close()
	}()
	msgTicker := time.NewTicker(time.Millisecond * 500)
	endTimer := time.NewTimer(time.Second * 2)
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
			err := c.conn.WriteJSON(&p)
			if err != nil {
				stderr <- []byte(fmt.Sprint("Error writing on client id: ", c.id, " Error: ", err.Error()))

			}
		case <-endTimer.C:
			msgTicker.Stop()
			return
		}
	}
}

// Returns the client after creating the session, the session code
// of an error.
func createSession() (TestClient, string, error) {
	client, err := WsDial("/new-session/TestSession")
	if err != nil {
		return TestClient{}, "", err
	}
	res := controllers.HandshakeRes{}
	// read control message containing the session code+
	_, data, err := client.conn.ReadMessage()
	if err != nil {
		client.conn.Close()
		return TestClient{}, "", err
	}
	err = json.Unmarshal(data, &res)
	if err != nil {
		client.conn.Close()
		return TestClient{}, "", err
	}
	if res.StatusCode != 200 {
		client.conn.Close()
		return TestClient{}, "", errors.New(res.Status)
	}
	return client, res.SessionCode, nil

}

func joinSession(code string) (TestClient, error) {
	client, err := WsDial(fmt.Sprint("/join-session/", code))
	if err != nil {
		return TestClient{}, err
	}
	res := controllers.HandshakeRes{}
	// read control message containing the session code+
	_, data, err := client.conn.ReadMessage()
	if err != nil {
		client.conn.Close()
		return TestClient{}, err
	}
	err = json.Unmarshal(data, &res)
	if err != nil {
		client.conn.Close()
		return TestClient{}, err
	}
	if res.StatusCode != 200 {
		return TestClient{}, errors.New(res.Status)
	}
	return client, nil
}

func WsDial(endpoint string) (TestClient, error) {
	url2 := url.URL{Scheme: "ws", Host: HOST, Path: endpoint}
	conn, _, err := websocket.DefaultDialer.Dial(url2.String(), nil)
	if err != nil {
		return TestClient{}, err
	}
	return TestClient{
		conn: conn,
		id:   uuid.NewString(),
	}, nil
}
