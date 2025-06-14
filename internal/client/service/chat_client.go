package service

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/as283-ua/yappa/api/gen/client"
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
	subs   map[[32]byte][]chan *client.ClientEvent

	MainSub chan *server.ServerMessage

	connected bool
	// ConnectedC chan bool
}

var chatClient *ChatClient
var ConnectedC chan bool = make(chan bool, 50)

func InitChatClient(h3c *http.Client) *ChatClient {
	chatClient = &ChatClient{
		client:  h3c,
		str:     nil,
		subsMu:  sync.RWMutex{},
		subs:    make(map[[32]byte][]chan *client.ClientEvent),
		MainSub: make(chan *server.ServerMessage, 50),
	}
	return chatClient
}

func GetChatClient() *ChatClient {
	return chatClient
}

func (c *ChatClient) GetConnected() bool {
	return c.connected
}

func (c *ChatClient) setConnected(connected bool) {
	c.connected = connected
	ConnectedC <- c.connected
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
	if int(msgLen) >= cap(msgRaw) {
		return fmt.Errorf("invalid incoming message size: %v. Max: %v", msgLen, cap(msgRaw))
	}
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
	var msgRaw, lenBytes []byte = make([]byte, 0, 4096), make([]byte, 4)
	defer c.Close()
	for c.connected {
		msg := &server.ServerMessage{}
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
}

func (c *ChatClient) Emit(inboxId []byte, msg *client.ClientEvent) {
	c.subsMu.RLock()
	defer c.subsMu.RUnlock()

	inboxSubs, ok := c.subs[[32]byte(inboxId)]
	if !ok {
		return
	}

	for _, ch := range inboxSubs {
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

func (c *ChatClient) Subscribe(inboxId [32]byte) (int, chan *client.ClientEvent) {
	c.subsMu.RLock()
	defer c.subsMu.RUnlock()

	inboxSubs, ok := c.subs[inboxId]
	if !ok {
		inboxSubs = make([]chan *client.ClientEvent, 0)
	}
	ch := make(chan *client.ClientEvent, 50)
	inboxSubs = append(inboxSubs, ch)
	c.subs[inboxId] = inboxSubs
	id := len(c.subs[inboxId])

	return id, ch
}

func (c *ChatClient) Unsubscribe(inboxId [32]byte, id int) {
	c.subsMu.RLock()
	defer c.subsMu.RUnlock()

	inboxSubs, ok := c.subs[inboxId]
	if !ok {
		return
	}
	if id < 0 || id >= len(inboxSubs) {
		return
	}
	inboxSubs[id] = nil
}
