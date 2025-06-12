package service

import (
	"bytes"
	"crypto/mlkem"
	"fmt"
	"log"

	"github.com/as283-ua/yappa/api/gen/client"
	"github.com/as283-ua/yappa/api/gen/server"
	"github.com/as283-ua/yappa/internal/client/save"
	"github.com/as283-ua/yappa/pkg/common"
)

var chatCache = make(map[[32]byte]*client.Chat)
var encapCache = make(map[string]*mlkem.EncapsulationKey1024)

func getChat(saveState *client.SaveState, inboxId []byte) (*client.Chat, error) {
	chat, ok := chatCache[[32]byte(inboxId)]
	if !ok {
		chat, ok = save.DirectChat(saveState, inboxId)
		if ok {
			chatCache[[32]byte(inboxId)] = chat
		}
	}
	if chat == nil {
		newChats, err := GetChatClient().GetNewChats()
		if err != nil {
			return nil, fmt.Errorf("errors while retrieving new chats: %v", err)
		}
		for _, nc := range newChats {
			save.NewDirectChat(saveState, nc)
			if bytes.Equal(nc.Peer.InboxId, inboxId) {
				chat = nc
			}
		}
	}
	if chat == nil {
		return nil, fmt.Errorf("received message from unknown inbox: %v", inboxId)
	}
	return chat, nil
}

func getEncap(chat *client.Chat) (*mlkem.EncapsulationKey1024, error) {
	var err error
	encapKey, ok := encapCache[chat.Peer.Username]
	if !ok {
		encapKey, err = mlkem.NewEncapsulationKey1024(chat.Peer.KeyExchange)
		if err != nil {
			return nil, err
		}
		encapCache[chat.Peer.Username] = encapKey
	}

	return encapKey, nil
}

func KeyExchNeeded(chat *client.Chat) bool {
	eventIdx := chat.CurrentSerial - chat.SerialStart
	var keyRotUserOffset int = 0
	if chat.Peer.Username == chat.Initiator {
		keyRotUserOffset = MLKEM_RATCHET_INTERVAL / 2
	}

	isMyTurn := (int(eventIdx)+keyRotUserOffset)%MLKEM_RATCHET_INTERVAL == 0

	lastSeen := -1
	for i := 0; !isMyTurn && i < len(chat.Events) && i < MLKEM_RATCHET_INTERVAL/2; i++ {
		if _, ok := chat.Events[len(chat.Events)-i-1].Payload.(*client.ClientEvent_KeyRotation); ok {
			lastSeen = i
			break
		}
	}

	if lastSeen == -1 && len(chat.Events) > MLKEM_RATCHET_INTERVAL/2 {
		lastSeen = MLKEM_RATCHET_INTERVAL/2 + 1
	}

	if !isMyTurn && (lastSeen > MLKEM_RATCHET_INTERVAL/2) {
		log.Printf("Long time no see: Current serial = %v, first message = %v. Frequency = %v", chat.CurrentSerial, chat.SerialStart, MLKEM_RATCHET_INTERVAL)
	}
	return isMyTurn || (lastSeen > MLKEM_RATCHET_INTERVAL/2)
}

func StartListening(saveState *client.SaveState) {
	chatCli := GetChatClient()
	<-ConnectedC
	for chatCli.GetConnected() {
		msg := <-chatCli.MainSub
		switch payload := msg.Payload.(type) {
		case *server.ServerMessage_Send:
			chat, err := getChat(saveState, msg.GetSend().InboxId)
			if err != nil {
				log.Printf("Error reading new incoming message: %v", err)
				break
			}

			event, usedSerial, err := DecryptPeerMessage(chat, payload)
			if err != nil {
				log.Println("Error decrypting peer msg:", err, usedSerial, payload.Send.Serial, common.Hash(payload.Send.EncData))
				break
			}
			var newSerial uint64 = chat.CurrentSerial
			var newKey []byte = chat.Key

			errored := false
			switch msg := event.Payload.(type) {
			case *client.ClientEvent_KeyRotation:
				decapKey := GetMlkemDecap()
				if decapKey == nil {
					errored = true
					log.Println("Received key rotation message but no MLKEM key is loaded")
					break
				}
				newKey, err = decapKey.Decapsulate(msg.KeyRotation.KeyExchangeData)
				if err != nil {
					errored = true
					log.Println("Error decapsulating key:", err)
					break
				}
				newSerial = usedSerial + 1
			default:
				if chat.CurrentSerial == usedSerial { // ratchet if order was kept, keep previous key and current serial otherwise
					newSerial++
					newKey = Ratchet(chat.Key)
				}
			}
			if errored {
				// todo send NACK to redo key exchange
				break
			}
			save.NewEvent(chat, newSerial, newKey, event)
			chatCli.Emit(chat.Peer.InboxId, event)

			if KeyExchNeeded(chat) {
				log.Printf("Sending key exchange on receive. Current serial = %v, first message = %v. Frequency = %v", chat.CurrentSerial, chat.SerialStart, MLKEM_RATCHET_INTERVAL)
				encapKey, err := getEncap(chat)
				if err != nil {
					log.Println("Error getting peer's public key:", err)
					continue
				}
				encMsg, kevent, key, err := KeyExchangeEvent(chat, encapKey)
				if err != nil {
					log.Println("Error in key exchange message creation:", err)
					continue
				}
				err = GetChatClient().Send(&server.ClientMessage{
					Payload: &server.ClientMessage_Send{
						Send: encMsg,
					},
				})
				if err != nil {
					log.Println("Error sending key exchange:", err)
					continue
				}
				save.NewEvent(chat, chat.CurrentSerial+1, key, kevent)
				chatCli.Emit(chat.Peer.InboxId, kevent)
			}
		}
	}
}
