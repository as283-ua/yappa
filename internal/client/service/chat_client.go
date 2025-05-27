package service

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/mlkem"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	mathrand "math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	cli_proto "github.com/as283-ua/yappa/api/gen/client"
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

func (c *ChatClient) initChat() ([]byte, error) {
	url := fmt.Sprintf("https://%v/chat/init", settings.CliSettings.ServerHost)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, handleHttpErrors(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%v", string(body))
	}
	chatInit := &server.ChatInit{}
	err = proto.Unmarshal(body, chatInit)
	if err != nil {
		return nil, err
	}

	return chatInit.InboxId, nil
}

func (c *ChatClient) notifyNewChat(notify *server.ChatInitNotify) error {
	raw, err := proto.Marshal(notify)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://%v/chat/notify", settings.CliSettings.ServerHost)

	req, err := http.NewRequest("POST", url, bytes.NewReader(raw))
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return handleHttpErrors(err)
	}
	defer resp.Body.Close()

	return nil
}

func chatData(peer *server.UserData, inboxId []byte) (*cli_proto.Chat, []byte, error) {
	encapKey, err := mlkem.NewEncapsulationKey1024(peer.PubKeyExchange)
	if err != nil {
		return nil, nil, err
	}
	key, keyExchData := encapKey.Encapsulate()

	var serialBytes [8]byte
	_, err = rand.Read(serialBytes[:])
	if err != nil {
		return nil, nil, err
	}
	serial := binary.LittleEndian.Uint64(serialBytes[:])
	client := &cli_proto.Chat{
		Events:        make([]*cli_proto.ClientEvent, 0),
		SerialStart:   serial,
		CurrentSerial: serial,
		Key:           key,
		Peer: &cli_proto.PeerData{
			Username:    peer.Username,
			KeyExchange: peer.PubKeyExchange,
			Cert:        []byte(peer.Certificate),
			InboxId:     inboxId,
		},
	}
	return client, keyExchData, nil
}

func encryptChatData(peername string, sendername string, serial uint64, inboxId, key, keyExchData []byte, privSignKey *ecdsa.PrivateKey) (*server.ChatInitNotify, error) {
	serialB := make([]byte, 8)
	binary.LittleEndian.PutUint64(serialB[:], serial)
	encSerial, err := common.Encrypt(serialB, key)
	if err != nil {
		return nil, err
	}

	encSender, err := common.Encrypt([]byte(sendername), key)
	if err != nil {
		return nil, err
	}

	signature, err := privSignKey.Sign(rand.Reader, inboxId, crypto.BLAKE2b_256)
	if err != nil {
		return nil, err
	}
	encSign, err := common.Encrypt(signature, key)
	if err != nil {
		return nil, err
	}

	encInboxId, err := common.Encrypt(inboxId, key)
	if err != nil {
		return nil, err
	}

	notify := &server.ChatInitNotify{
		Receiver:        peername,
		EncSerial:       encSerial,
		EncSender:       encSender,
		EncSignature:    encSign,
		EncInboxId:      encInboxId,
		KeyExchangeData: keyExchData,
	}

	return notify, nil
}

func (c *ChatClient) NewChat(peer *server.UserData) (*cli_proto.Chat, error) {
	inboxId, err := c.initChat()
	if err != nil {
		return nil, err
	}
	chat, keyExchData, err := chatData(peer, inboxId)
	if err != nil {
		return nil, err
	}

	t, ok := c.client.Transport.(*http3.Transport)
	if !ok {
		return nil, errors.New("http transport retrieve error")
	}

	tlsConf := t.TLSClientConfig
	if tlsConf == nil || len(tlsConf.Certificates) == 0 {
		return nil, errors.New("no certificates loaded")
	}

	clientName := GetUsername()

	privK, ok := GetCertificate().PrivateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not of expected type ECDSA")
	}

	chatNotify, err := encryptChatData(chat.Peer.Username, clientName, chat.CurrentSerial, inboxId, chat.Key, keyExchData, privK)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data for chat notification: %w", err)
	}

	err = c.notifyNewChat(chatNotify)
	if err != nil {
		return nil, fmt.Errorf("failed to send notification: %w", err)
	}
	return chat, nil
}
