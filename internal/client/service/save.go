package service

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/as283-ua/yappa/api/gen/client"
	"google.golang.org/protobuf/proto"
)

const WAL_PATH = "data.wal"

func savePath() string {
	return fmt.Sprintf("chats_%v.data", username)
}

func LoadChats() (*client.SaveState, error) {
	saveState := &client.SaveState{}
	saveStateRaw, err := os.ReadFile(savePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return saveState, nil
		}
		return nil, err
	}

	// decrypt (not yet implemented)

	err = proto.Unmarshal(saveStateRaw, saveState)
	if err != nil {
		return nil, err
	}

	for _, v := range saveState.Chats {
		if v.Peer == nil {
			return nil, fmt.Errorf("nil peer in a chat %v", v)
		}
	}

	return saveState, nil
}

func SaveChats(save *client.SaveState) error {
	if username == "" {
		return nil
	}

	saveRaw, err := proto.Marshal(save)
	if err != nil {
		return err
	}

	// encrypt file

	return os.WriteFile(savePath(), saveRaw, 0600)
}

func NewDirectChat(save *client.SaveState, chat *client.Chat) {
	c := DirectChat(save, chat.Peer.InboxId)
	if c == nil {
		log.Printf("Added new chat with %v\n", chat.Peer.Username)
		save.Chats = append(save.Chats, chat)
		return
	}
}

func NewEvent(chat *client.Chat, event *client.ClientEvent) {
	chat.Events = append(chat.Events, event)

	for i := chat.CurrentSerial + 1; i < event.Serial; i++ {
		// add missing event: map serial -> key
	}

	chat.CurrentSerial = event.Serial
	chat.Key = Ratchet(chat.Key)
	// todo: handle key change here
}

func DirectChat(save *client.SaveState, inboxId []byte) *client.Chat {
	for _, v := range save.Chats {
		if bytes.Equal(v.Peer.InboxId, inboxId) {
			return v
		}
	}
	return nil
}

func DirectChatByUser(save *client.SaveState, username string) *client.Chat {
	for _, v := range save.Chats {
		if v.Peer.Username == username {
			return v
		}
	}
	return nil
}
