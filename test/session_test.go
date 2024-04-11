package test

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Aaron-json/WsSession/internal/controllers"
	"github.com/Aaron-json/WsSession/internal/pkg/client"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

var HOST string
var testFileData []byte
var testFileName = "sample_file.jpg"

type TestClient struct {
	conn *websocket.Conn
	id   string
}

func TestMain(m *testing.M) {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal(err)
	}
	fp, err := os.Open(testFileName)
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()
	testFileData, err = io.ReadAll(fp)
	testFileName = ".jpg"
	if err != nil {
		log.Panic(err)
	}
	HOST = fmt.Sprint("127.0.0.1:", os.Getenv("PORT"))
	m.Run()
}

func TestSessions(t *testing.T) {
	nSessions := 1 // number of concurrent sessions
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
	go BinaryReader(ownerConn, stdout, stderr, listenWg)
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
		go BinaryReader(client, stdout, stderr, listenWg)

		conns = append(conns, client)
	}
	for i, conn := range conns {
		stdout <- []byte(fmt.Sprint("Testing client: ", i+1))
		BinaryWriter(conn, stdout, stderr)
		time.Sleep(time.Second)
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

func TextReader(c TestClient, stdout chan []byte, stderr chan []byte, wg *sync.WaitGroup) {
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
		if msgType == client.TextMessage {
			stdout <- data
		} else {
			stderr <- []byte(fmt.Sprint("Unsupported message type: ", msgType))
		}
	}
}

func BinaryReader(c TestClient, stdout chan []byte, stderr chan []byte, wg *sync.WaitGroup) {
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
		if msgType == client.BinaryMessage {
			// attempt to recreate the sent file
			fp, err := os.Create(fmt.Sprint(c.id, ".", strings.Split(testFileName, ".")[1]))
			if err != nil {
				stderr <- []byte(err.Error())
				return
			}
			defer fp.Close()
			fp.Write(data)

		} else {
			stderr <- []byte(fmt.Sprint("Unsupported message type: ", msgType))
		}
	}
}

func BinaryWriter(c TestClient, stdout chan []byte, stderr chan []byte) {
	defer c.conn.Close()
	if testFileData == nil {
		stderr <- []byte("Write Error: File has not been initialized")
		return
	}
	c.conn.WriteMessage(client.BinaryMessage, testFileData)
}

func TextWriter(c TestClient, stdout chan []byte, stderr chan []byte) {
	defer c.conn.Close()
	msgTicker := time.NewTicker(time.Millisecond * 200)
	endTimer := time.NewTimer(time.Second * 1)
	msg := "This is my message"
	p := controllers.Message{
		Data: msg,
	}
	data, err := json.Marshal(&p)
	if err != nil {
		// could not marshall text message
		stderr <- []byte(fmt.Sprint("Error writing on client id: ", c.id, " Error: ", err.Error()))
	}
	for {
		select {
		case <-msgTicker.C:
			err := c.conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				stderr <- []byte(err.Error())
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
	res := controllers.HandshakeMessage{}
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
	res := controllers.HandshakeMessage{}
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
	url := url.URL{Scheme: "ws", Host: HOST, Path: endpoint}
	conn, _, err := websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		return TestClient{}, err
	}
	return TestClient{
		conn: conn,
		id:   uuid.NewString(),
	}, nil
}
