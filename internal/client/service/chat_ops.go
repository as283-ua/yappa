package service

import (
	"crypto/sha256"
	"time"

	cli_proto "github.com/as283-ua/yappa/api/gen/client"
	"github.com/as283-ua/yappa/api/gen/server"
	"github.com/as283-ua/yappa/pkg/common"
	"google.golang.org/protobuf/proto"
)

const MAX_RATCHET_CYCLE = 20

func Ratchet(v []byte) []byte {
	h := sha256.New()
	h.Write(v)
	return h.Sum(nil)
	// return v
}

func DecryptPeerMessage(chat *cli_proto.Chat, msg *server.ServerMessage_Send) (*cli_proto.ClientEvent, uint64, error) {
	var usedKey []byte = chat.Key
	usedSerial := chat.CurrentSerial
	if chat.CurrentSerial == msg.Send.Serial {
		usedKey = chat.Key
	} else {
		usedSerial = msg.Send.Serial
		// ratchet should not extend more than MAX_RATCHET_CYCLE. should have set the new key with mlkem
		// if msg.Send.Serial == chat.CurrentSerial+save.MAX_RATCHET_CYCLE {
		// 	return nil, fmt.Errorf("serial number for message (%v) exceeded MAX RATCHET CYCLE (%v)", msg.Send.Serial, save.MAX_RATCHET_CYCLE)
		// }

		// ratchet until we get key for serial of msg
		for i := chat.CurrentSerial; i < msg.Send.Serial; i++ {
			usedKey = Ratchet(usedKey)
		}
	}
	encRaw := msg.Send.EncData
	raw, err := common.Decrypt(encRaw, usedKey)
	if err != nil {
		return nil, 0, err
	}

	peerMsg := &cli_proto.ClientEvent{}
	err = proto.Unmarshal(raw, peerMsg)

	if err != nil {
		return nil, 0, err
	}
	return peerMsg, usedSerial, nil
}

func EncryptMessageForPeer(chat *cli_proto.Chat, txt string) (*server.SendMsg, *cli_proto.ClientEvent, error) {
	event := &cli_proto.ClientEvent{
		Timestamp: uint64(time.Now().UTC().Unix()),
		Serial:    chat.CurrentSerial,
		Sender:    GetUsername(),
		Payload: &cli_proto.ClientEvent_Message{
			Message: &cli_proto.ChatMessage{
				Msg: txt,
			},
		},
	}
	raw, err := proto.Marshal(event)
	if err != nil {
		return nil, nil, err
	}

	encRaw, err := common.Encrypt(raw, chat.Key)
	if err != nil {
		return nil, nil, err
	}

	return &server.SendMsg{
		Serial:   chat.CurrentSerial,
		Receiver: chat.Peer.Username,
		InboxId:  chat.Peer.InboxId,
		Message:  encRaw,
	}, event, nil
}
