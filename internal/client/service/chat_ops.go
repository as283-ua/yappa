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
}

func DecryptPeerMessage(chat *cli_proto.Chat, msg *server.ServerMessage_Send) (*cli_proto.ClientEvent, uint64, []byte, error) {
	var key []byte
	var currentSerial = chat.CurrentSerial + 1
	if chat.CurrentSerial+1 == msg.Send.Serial {
		key = chat.Key
		currentSerial = msg.Send.Serial
	} else {
		// ratchet should not extend more than MAX_RATCHET_CYCLE. should have set the new key with mlkem
		// if msg.Send.Serial == chat.CurrentSerial+save.MAX_RATCHET_CYCLE {
		// 	return nil, fmt.Errorf("serial number for message (%v) exceeded MAX RATCHET CYCLE (%v)", msg.Send.Serial, save.MAX_RATCHET_CYCLE)
		// }

		// ratchet until we get key for serial of msg
		for i := chat.CurrentSerial + 1; i < msg.Send.Serial; i++ {
			key = Ratchet(key)
		}
	}
	encRaw := msg.Send.EncData
	raw, err := common.Decrypt(encRaw, key)
	if err != nil {
		return nil, 0, nil, err
	}

	peerMsg := &cli_proto.ClientEvent{}
	err = proto.Unmarshal(raw, peerMsg)

	if err != nil {
		return nil, 0, nil, err
	}
	return peerMsg, currentSerial, key, nil
}

func EncryptMessageForPeer(chat *cli_proto.Chat, txt string) (*server.SendMsg, *cli_proto.ClientEvent, []byte, error) {
	serial := chat.CurrentSerial + 1
	event := &cli_proto.ClientEvent{
		Timestamp: uint64(time.Now().UTC().Unix()),
		Serial:    serial,
		Sender:    GetUsername(),
		Payload: &cli_proto.ClientEvent_Message{
			Message: &cli_proto.ChatMessage{
				Msg: txt,
			},
		},
	}
	raw, err := proto.Marshal(event)
	if err != nil {
		return nil, nil, nil, err
	}
	key := Ratchet(chat.Key)

	encRaw, err := common.Encrypt(raw, key)
	if err != nil {
		return nil, nil, nil, err
	}

	return &server.SendMsg{
		Serial:   serial,
		Receiver: chat.Peer.Username,
		InboxId:  chat.Peer.InboxId,
		Message:  encRaw,
	}, event, key, nil
}
