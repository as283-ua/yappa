package save

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/as283-ua/yappa/api/gen/client"
	"github.com/as283-ua/yappa/pkg/common"
	"github.com/zalando/go-keyring"
	"google.golang.org/protobuf/proto"
)

const WAL_PATH = "data.wal"
const SERVICE_NAME = "YAPPA_PRIV_CHAT"

var username string
var mx = sync.Mutex{}

func SetSavepathUsername(name string) {
	username = name
}
func savePath() string {
	return fmt.Sprintf("chats_%v.data", username)
}

func LoadChats() (*client.SaveState, error) {
	mx.Lock()
	defer mx.Unlock()
	saveState := &client.SaveState{}
	encSaveGzipd, err := os.ReadFile(savePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return saveState, nil
		}
		return nil, fmt.Errorf("read file error: %v", err)
	}

	key64, err := keyring.Get(SERVICE_NAME, username)
	if err != nil {
		return nil, fmt.Errorf("keyring access error: %v", err)
	}
	key := make([]byte, 32)
	_, err = base64.StdEncoding.Decode(key, []byte(key64))
	if err != nil {
		return nil, fmt.Errorf("base 64 decode error: %v", err)
	}

	saveGzipd, err := common.Decrypt(encSaveGzipd, key)
	if err != nil {
		return nil, fmt.Errorf("decrypt error: %v", err)
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(saveGzipd))
	if err != nil {
		return nil, fmt.Errorf("gzip error: %v", err)
	}
	saveRaw, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, fmt.Errorf("read zip error: %v", err)
	}

	err = proto.Unmarshal(saveRaw, saveState)
	if err != nil {
		return nil, fmt.Errorf("format error: %v", err)
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

	key := make([]byte, 32)
	rand.Read(key)
	key64 := base64.StdEncoding.EncodeToString(key)
	err = keyring.Set(SERVICE_NAME, username, key64)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	_, err = gzipWriter.Write(saveRaw)
	if err != nil {
		return err
	}
	err = gzipWriter.Close()
	if err != nil {
		return err
	}

	saveGzipd := buf.Bytes()

	encSaveGzipd, err := common.Encrypt(saveGzipd, key)
	if err != nil {
		return err
	}

	return os.WriteFile(savePath(), encSaveGzipd, 0600)
}

func NewDirectChat(save *client.SaveState, chat *client.Chat) {
	_, ok := DirectChat(save, chat.Peer.InboxId)
	if !ok {
		save.Chats = append(save.Chats, chat)
		log.Printf("Added new chat with %v\n", chat.Peer.Username)
		return
	}
}

func NewEvent(chat *client.Chat, nextSerial uint64, nextKey []byte, event *client.ClientEvent) {
	mx.Lock()
	defer mx.Unlock()
	chat.Events = append(chat.Events, event)
	chat.CurrentSerial = nextSerial
	chat.Key = nextKey
}

func DirectChat(save *client.SaveState, inboxId []byte) (*client.Chat, bool) {
	for _, v := range save.Chats {
		if bytes.Equal(v.Peer.InboxId, inboxId) {
			return v, true
		}
	}
	return nil, false
}

func DirectChatByUser(save *client.SaveState, username string) *client.Chat {
	for _, v := range save.Chats {
		if v.Peer.Username == username {
			return v
		}
	}
	return nil
}
