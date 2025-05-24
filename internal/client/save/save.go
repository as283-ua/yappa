package save

import (
	"errors"
	"os"

	"github.com/as283-ua/yappa/api/gen/client"
	"google.golang.org/protobuf/proto"
)

const SAVE_PATH = "chats.data"

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
