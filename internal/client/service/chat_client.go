package service

import (
	"context"
	"encoding/binary"
	"log"
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
	subs   []chan<- *server.ServerMessage

	MainSub chan<- *server.ServerMessage
}

var client *ChatClient

func InitChatClient(h3c *http.Client) *ChatClient {
	client = &ChatClient{
		client:  h3c,
		str:     nil,
		subsMu:  sync.RWMutex{},
		subs:    make([]chan<- *server.ServerMessage, 0),
		MainSub: make(chan<- *server.ServerMessage, 50),
	}
	return client
}

func GetChatClient() *ChatClient {
	return client
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
	log.Println("Connected")
	go c.readloop()
	go c.heartbeatLoop()
	return nil
}

func (c *ChatClient) Close() error {
	return c.str.Close()
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

func (c *ChatClient) readloop() {
	var msg server.ServerMessage
	var msgRaw, lenBytes []byte = make([]byte, 0, 4096), make([]byte, 4)
	defer c.str.Close()
	for {
		_, err := c.str.Read(lenBytes)
		if err != nil {
			log.Println("Readloop error, byte length read:", err)
			break
		}
		msgLen := binary.BigEndian.Uint32(lenBytes[:])
		_, err = c.str.Read(msgRaw[:msgLen])
		if err != nil {
			log.Println("Readloop error, data read:", err)
			break
		}

		err = proto.Unmarshal(msgRaw, &msg)
		if err != nil {
			log.Println("Readloop error, unmarshal:", err)
			break
		}
		c.dispatch(&msg)
	}
}

func (c *ChatClient) dispatch(msg *server.ServerMessage) {
	c.MainSub <- msg

	c.subsMu.RLock()
	defer c.subsMu.RUnlock()

	for _, ch := range c.subs {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (c *ChatClient) heartbeatLoop() {
	ticker := time.NewTicker(20 * time.Second)
	for {
		<-ticker.C
		c.Send(&server.ClientMessage{Payload: &server.ClientMessage_Hb{}})
		log.Printf("Heartbeat %v\n", time.Now())
	}
}

func (c *ChatClient) Subscribe() chan<- *server.ServerMessage {
	c.subsMu.RLock()
	defer c.subsMu.RUnlock()

	ch := make(chan<- *server.ServerMessage, 50)
	c.subs = append(c.subs, ch)

	return ch
}
