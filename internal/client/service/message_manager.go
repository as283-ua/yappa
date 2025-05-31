package service

import (
	"bytes"
	"log"

	"github.com/as283-ua/yappa/api/gen/client"
	"github.com/as283-ua/yappa/api/gen/server"
	"github.com/as283-ua/yappa/internal/client/save"
)

func StartListening(saveState *client.SaveState) {
	chatCli := GetChatClient()
	chatMap := make(map[[32]byte]*client.Chat)
	<-ConnectedC
	for chatCli.GetConnected() {
		msg := <-chatCli.MainSub
		switch payload := msg.Payload.(type) {
		case *server.ServerMessage_Send:
			chat, ok := chatMap[[32]byte(payload.Send.InboxId)]
			if !ok {
				chat, ok = save.DirectChat(saveState, payload.Send.InboxId)
				if ok {
					chatMap[[32]byte(payload.Send.InboxId)] = chat
				}
			}
			if chat == nil {
				newChats, err := GetChatClient().GetNewChats()
				if err != nil {
					log.Printf("Errors while retrieving new chats: %v", err)
				}
				for _, nc := range newChats {
					save.NewDirectChat(saveState, nc)
					if bytes.Equal(nc.Peer.InboxId, payload.Send.InboxId) {
						chat = nc
					}
				}
			}
			if chat == nil {
				log.Printf("Received message from unknown inbox: %v", payload.Send.InboxId)
				continue
			}
			event, usedSerial, err := DecryptPeerMessage(chat, payload)
			if err != nil {
				log.Println("Error decrypting peer msg:", err)
				break
			}
			var newSerial uint64 = chat.CurrentSerial
			var newKey []byte = chat.Key
			if chat.CurrentSerial == usedSerial { // ratchet if order was kept, keep previous key and current serial otherwise
				newSerial++
				errored := false
				switch msg := event.Payload.(type) {
				case *client.ClientEvent_KeyRotation:
					decapKey := GetMlkemDecap()
					if decapKey != nil {
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
				default:
					newKey = Ratchet(chat.Key)
				}
				if errored {
					// todo send NACK to redo key exchange
					break
				}
			}
			save.NewEvent(chat, newSerial, newKey, event)
			chatCli.Emit(chat.Peer.InboxId, event)
		}
	}
}
