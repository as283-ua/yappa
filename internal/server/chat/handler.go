package chat

import (
	"io"
	"net/http"

	"github.com/as283-ua/yappa/api/gen"
	"github.com/as283-ua/yappa/internal/server/logging"
	"google.golang.org/protobuf/proto"
)

func CreateChatInbox(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Println("Body read error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	chatInit := &gen.ChatInit{}
	err = proto.Unmarshal(body, chatInit)
	if err != nil {
		http.Error(w, "Incorrect body format", http.StatusBadRequest)
		return
	}

	err = Repo.CreateChatInbox(chatInit.InboxId)
	if err != nil {
		logger.Println("DB error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
func NotifyChatInbox(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Println("Body read error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	notify := &gen.ChatInitNotify{}
	err = proto.Unmarshal(body, notify)
	if err != nil {
		http.Error(w, "Incorrect body format", http.StatusBadRequest)
		return
	}

	err = Repo.ShareChatInbox(notify.Receiver, notify.EncSender, notify.EncInboxId, notify.EcdhPub)
	if err != nil {
		logger.Println("DB error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
