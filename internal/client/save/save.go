package save

import (
	"bytes"
	"errors"
	"os"

	"github.com/as283-ua/yappa/api/gen/client"
	"google.golang.org/protobuf/proto"
)

const SAVE_PATH = "chats.data"
const WAL_PATH = "data.wal"

func LoadChats() (*client.SaveState, error) {
	saveState := &client.SaveState{}
	saveStateRaw, err := os.ReadFile(SAVE_PATH)
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

	return saveState, nil
}

func SaveChats(save *client.SaveState) error {
	saveRaw, err := proto.Marshal(save)
	if err != nil {
		return err
	}

	// encrypt file

	return os.WriteFile(SAVE_PATH, saveRaw, 0600)
}

func NewDirectChat(save *client.SaveState, chat *client.Chat) {
	save.Chats = append(save.Chats, chat)
}

func NewEvent(save *client.SaveState, peer *client.PeerData, event *client.ClientEvent) {
	idx := -1
	var chat *client.Chat
	for i, v := range save.Chats {
		if bytes.Equal(v.Peer.InboxId, peer.InboxId) && v.Peer.Username == peer.Username {
			idx = i
			chat = v
		}
	}

	if idx == -1 {
		return
	}

	chat.Events = append(chat.Events, event)
}

func DirectChat(save *client.SaveState, username string) *client.Chat {
	for _, v := range save.Chats {
		if v.Peer.Username == username {
			return v
		}
	}
	return nil
}
