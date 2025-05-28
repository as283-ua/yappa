package connection

import (
	"context"
	"crypto/mlkem"
	"crypto/rand"
	"encoding/binary"
	"io"
	"net/http"

	"github.com/as283-ua/yappa/api/gen/server"
	"github.com/as283-ua/yappa/internal/server/auth"
	"github.com/as283-ua/yappa/internal/server/chat"
	"github.com/as283-ua/yappa/internal/server/logging"
	"github.com/as283-ua/yappa/pkg/common"
	"github.com/quic-go/quic-go/http3"
	"google.golang.org/protobuf/proto"
)

var sessions map[string]*http3.Stream = make(map[string]*http3.Stream)

func upgrade(w http.ResponseWriter) (http3.Stream, error) {
	w.WriteHeader(http.StatusOK)
	w.(http.Flusher).Flush()

	return w.(http3.HTTPStreamer).HTTPStream(), nil
}

func Connection(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger()
	username := r.TLS.PeerCertificates[0].Subject.CommonName
	logger.Println("Someone connected:", username)

	_, err := auth.Repo.GetUserData(context.Background(), username)
	if err != nil {
		http.Error(w, "Invalid user", http.StatusBadRequest)
		logger.Println("Invalid user:", err)
		return
	}

	str, err := upgrade(w)
	if err != nil {
		http.Error(w, "Bad http configuration", http.StatusBadRequest)
		logger.Println("Upgrade error:", err)
		return
	}
	defer func() {
		delete(sessions, username)
		str.Close()
	}()
	sessions[username] = &str

	for {
		var lenBuf [4]byte
		_, err := io.ReadFull(str, lenBuf[:])
		if err != nil {
			logger.Println("Connection error:", err)
			return
		}
		msgLen := binary.BigEndian.Uint32(lenBuf[:])
		var msg []byte = make([]byte, msgLen)

		str.Read(msg)

		protoMsg := &server.ClientMessage{}
		err = proto.Unmarshal(msg, protoMsg)

		if err != nil {
			logger.Println("Failed to unmarshall client data:", err)
			return
		}

		switch payload := protoMsg.Payload.(type) {
		case *server.ClientMessage_Send:
			chatSend := payload.Send
			handleMsg(chatSend)
		case *server.ClientMessage_Hb:
		default:
			// Unknown or unset
		}
	}
}

func handleMsg(msg *server.SendMsg) {
	conn, ok := sessions[msg.Receiver]
	if !ok {
		saveToInbox(msg)
		return
	}
	send := &server.ServerMessage{
		Payload: &server.ServerMessage_Send{
			Send: &server.ReceiveMsg{
				Serial:  msg.Serial,
				InboxId: msg.InboxId,
				EncData: msg.Message,
			},
		},
	}
	sendBytes, _ := proto.Marshal(send)

	messageLen := len(sendBytes)
	lenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBytes, uint32(messageLen))

	(*conn).Write((append(lenBytes, sendBytes...)))
}

func saveToInbox(msg *server.SendMsg) error {
	tokenObj, err := chat.Repo.GetToken(msg.InboxId)
	if err != nil {
		return err
	}
	if tokenObj.KeyExchangeData == nil {
		receiver, err := auth.Repo.GetUserData(context.Background(), msg.Receiver)
		if err != nil {
			logging.GetLogger().Println("Get user error:", err)
			return err
		}
		kybKey, err := mlkem.NewEncapsulationKey1024(receiver.PubKeyExchange)
		if err != nil {
			logging.GetLogger().Println("Kyber encapsulation parse error:", err)
			return err
		}
		key, cipherText := kybKey.Encapsulate()

		token := make([]byte, 32)
		rand.Read(token)

		tokenEnc, err := common.Encrypt(token, key)
		if err != nil {
			logging.GetLogger().Println("AES error enc:", err)
			return err
		}

		err = chat.Repo.SetInboxToken(msg.InboxId, common.Hash(token), tokenEnc, cipherText)
		if err != nil {
			logging.GetLogger().Println("DB error:", err)
			return err
		}
	}

	err = chat.Repo.AddMessage(msg.InboxId, msg.Message)
	if err != nil {
		logging.GetLogger().Println("DB error:", err)
		return err
	}
	return nil
}
