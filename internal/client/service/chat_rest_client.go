package service

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/mlkem"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"

	cli_proto "github.com/as283-ua/yappa/api/gen/client"
	"github.com/as283-ua/yappa/api/gen/server"
	"github.com/as283-ua/yappa/internal/client/save"
	"github.com/as283-ua/yappa/internal/client/settings"
	"github.com/as283-ua/yappa/pkg/common"
	"github.com/quic-go/quic-go/http3"
	"google.golang.org/protobuf/proto"
)

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

func (c *ChatClient) fetchNewChats() (*server.ListNewChats, error) {
	url := fmt.Sprintf("https://%v/chat/new", settings.CliSettings.ServerHost)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to retrieve new chats: " + resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	chats := &server.ListNewChats{}
	err = proto.Unmarshal(data, chats)
	if err != nil {
		return nil, err
	}

	return chats, nil
}

func (c *ChatClient) fetchChatToken(inboxId []byte) (*server.InboxToken, error) {
	url := fmt.Sprintf("https://%v/chat/token", settings.CliSettings.ServerHost)

	req, err := http.NewRequest("POST", url, bytes.NewReader(inboxId))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("inbox not found")
	} else if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to retrieve chat token: " + resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	token := &server.InboxToken{}
	err = proto.Unmarshal(data, token)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (c *ChatClient) fetchNewMessages(inboxId, token []byte) (*server.ListNewMessages, error) {
	url := fmt.Sprintf("https://%v/chat/token", settings.CliSettings.ServerHost)

	getMsgs := &server.GetNewMessages{
		InboxId: inboxId,
		Token:   token,
	}
	payload, err := proto.Marshal(getMsgs)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, errors.New("inbox not found")
	case http.StatusUnauthorized:
		return nil, errors.New("bad token")
	case http.StatusBadRequest:
		return nil, errors.New("incorrect body format")
	default:
		return nil, errors.New("failed to retrieve new messages: " + resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	msgs := &server.ListNewMessages{}
	err = proto.Unmarshal(data, msgs)
	if err != nil {
		return nil, err
	}

	return msgs, nil
}

func (c *ChatClient) GetNewChats(saveState *cli_proto.SaveState) error {
	chats, err := c.fetchNewChats()
	if err != nil {
		return err
	}
	errs := common.MultiError{Errors: make([]error, 0)}
	for _, chat := range chats.Chats {
		key, err := GetMlkemDecap().Decapsulate(chat.KeyExchangeData)
		if err != nil {
			errs.Errors = append(errs.Errors, err)
			continue
		}
		inboxId, err := common.Decrypt(chat.EncInboxCode, key)
		if err != nil {
			errs.Errors = append(errs.Errors, err)
			continue
		}
		sender, err := common.Decrypt(chat.EncSender, key)
		if err != nil {
			errs.Errors = append(errs.Errors, err)
			continue
		}
		serialB, err := common.Decrypt(chat.EncSerial, key)
		if err != nil {
			errs.Errors = append(errs.Errors, err)
			continue
		}

		serial := binary.LittleEndian.Uint64(serialB[:])
		userData, err := UsersClient{Client: c.client}.GetUserData(string(sender))
		if err != nil {
			errs.Errors = append(errs.Errors, err)
			continue
		}

		save.NewDirectChat(saveState, &cli_proto.Chat{
			Events:        make([]*cli_proto.ClientEvent, 0),
			SerialStart:   serial,
			CurrentSerial: serial,
			Key:           key,
			Peer: &cli_proto.PeerData{
				Username:    userData.Username,
				KeyExchange: userData.PubKeyExchange,
				Cert:        []byte(userData.Certificate),
				InboxId:     inboxId,
			},
		})
	}

	return errs.NilOrError()
}

func (c *ChatClient) GetNewMessages(saveState *cli_proto.SaveState) error {
	errs := common.MultiError{Errors: make([]error, 0)}
	for _, chat := range saveState.Chats {
		tokenObj, err := c.fetchChatToken(chat.Peer.InboxId)
		if err != nil {
			errs.Errors = append(errs.Errors, err)
			continue
		}
		tokenKey, err := GetMlkemDecap().Decapsulate(tokenObj.KeyExchangeData)
		if err != nil {
			errs.Errors = append(errs.Errors, err)
			continue
		}
		token, err := common.Decrypt(tokenObj.EncToken, tokenKey)
		if err != nil {
			errs.Errors = append(errs.Errors, err)
			continue
		}
		messages, err := c.fetchNewMessages(chat.Peer.InboxId, token)
		if err != nil {
			errs.Errors = append(errs.Errors, err)
			continue
		}
		for _, encMsg := range messages.Msgs {
			msg, err := common.Decrypt(encMsg, chat.Key)
			if err != nil {
				errs.Errors = append(errs.Errors, err)
				continue
			}
			event := &cli_proto.ClientEvent{}
			err = proto.Unmarshal(msg, event)
			if err != nil {
				errs.Errors = append(errs.Errors, err)
				continue
			}
			save.NewEvent(saveState, chat, event)
		}
	}
	return errs.NilOrError()
}
