package service

import (
	"context"
	"encoding/binary"
	"log"
	mathrand "math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/as283-ua/yappa/api/gen/server"
	"github.com/as283-ua/yappa/internal/client/settings"
	"github.com/as283-ua/yappa/pkg/common"
	"github.com/quic-go/quic-go/http3"
	"google.golang.org/protobuf/proto"
)

type ChatClient struct {
	client *http.Client
	str    *common.BiStream

	subsMu sync.RWMutex
	subs   map[int]chan *server.ServerMessage

	MainSub chan *server.ServerMessage

	connected  bool
	ConnectedC chan bool
}

var client *ChatClient

func InitChatClient(h3c *http.Client) *ChatClient {
	client = &ChatClient{
		client:     h3c,
		str:        nil,
		subsMu:     sync.RWMutex{},
		subs:       make(map[int]chan *server.ServerMessage),
		MainSub:    make(chan *server.ServerMessage, 50),
		ConnectedC: make(chan bool, 1),
	}
	return client
}

func GetChatClient() *ChatClient {
	return client
}

func (c *ChatClient) GetConnected() bool {
	return c.connected
}

func (c *ChatClient) setConnected(connected bool) {
	c.connected = connected
	c.ConnectedC <- c.connected
	if connected {
		log.Println("Connected")
	} else {
		log.Println("Disconnected")
	}
}

func (c *ChatClient) Connect() error {
	serverURL := "https://" + settings.CliSettings.ServerHost + "/connect"
	u, err := url.Parse(serverURL)
	if err != nil {
		return err
	}

	c.str, err = common.Http3Stream(context.Background(), u, c.client.Transport.(*http3.Transport), http.Header{})
	if err != nil {
		return err
	}
	c.setConnected(true)
	go c.readloop()
	go c.heartbeatLoop()
	return nil
}

func (c *ChatClient) Close() error {
	if c != nil && c.GetConnected() {
		log.Println("Closed connection")
		c.setConnected(false)
		return c.str.Close()
	}
	return nil
}

func (c *ChatClient) Send(msg *server.ClientMessage) error {
	m, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	// length of message at the start of the frame
	messageLen := len(m)
	lenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBytes, uint32(messageLen))

	c.str.Write(append(lenBytes, m...))
	return nil
}

func (c *ChatClient) readOnce(msg *server.ServerMessage, msgRaw, lenBytes []byte) error {
	_, err := c.str.Read(lenBytes)
	if err != nil {
		return err
	}
	msgLen := binary.BigEndian.Uint32(lenBytes[:])
	_, err = c.str.Read(msgRaw[:msgLen])
	if err != nil {
		return err
	}
	err = proto.Unmarshal(msgRaw[:msgLen], msg)
	if err != nil {
		return err
	}
	return nil
}

func (c *ChatClient) readloop() {
	var msg = &server.ServerMessage{}
	var msgRaw, lenBytes []byte = make([]byte, 0, 4096), make([]byte, 4)
	defer c.Close()
	for c.connected {
		err := c.readOnce(msg, msgRaw, lenBytes)
		if err != nil {
			log.Println("Readloop error, unmarshal:", err)
			break
		}

		c.dispatch(msg)
	}
}

func (c *ChatClient) dispatch(msg *server.ServerMessage) {
	c.MainSub <- msg

	c.subsMu.RLock()
	defer c.subsMu.RUnlock()

	for id, ch := range c.subs {
		log.Printf("Dispatching message %v to subscriber with id %v", msg.GetSend(), id)
		select {
		case ch <- msg:
		default:
		}
	}
}

func (c *ChatClient) heartbeatLoop() {
	ticker := time.NewTicker(20 * time.Second)
	for c.connected {
		<-ticker.C
		err := c.Send(&server.ClientMessage{Payload: &server.ClientMessage_Hb{}})
		if err != nil {
			log.Printf("HB error: %v", err)
		}
		log.Printf("Heartbeat %v\n", time.Now())
	}
}

func (c *ChatClient) Subscribe() (int, chan *server.ServerMessage) {
	c.subsMu.RLock()
	defer c.subsMu.RUnlock()

	ch := make(chan *server.ServerMessage, 50)
	id := mathrand.Int()
	c.subs[id] = ch

	return id, ch
}

func (c *ChatClient) Unsubscribe(id int) {
	c.subsMu.RLock()
	defer c.subsMu.RUnlock()

	delete(c.subs, id)
}
