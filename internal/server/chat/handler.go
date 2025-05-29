package chat

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"net/http"

	"github.com/as283-ua/yappa/api/gen/server"
	"github.com/as283-ua/yappa/internal/server/logging"
	"github.com/as283-ua/yappa/pkg/common"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/protobuf/proto"
)

// Creates a new chat inbox with no data other that its id. Users can't be associated to a given inbox.
func CreateChatInbox(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger()
	inboxId := make([]byte, 32)
	for {
		rand.Read(inboxId)
		err := Repo.CreateChatInbox(inboxId)
		if err == nil {
			break
		}
		var e *pgconn.PgError
		if errors.As(err, &e) && e.Code != pgerrcode.UniqueViolation {
			logger.Println("DB error:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	chatInit := &server.ChatInit{
		InboxId: inboxId,
	}
	raw, err := proto.Marshal(chatInit)
	if err != nil {
		logger.Println("Marshal error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(raw)
}

// Puts the encrypted inbox id and sender username into the ChatInboxes table where users will check for new chats
func NotifyChatInbox(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Println("Body read error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	notify := &server.ChatInitNotify{}
	err = proto.Unmarshal(body, notify)
	if err != nil {
		http.Error(w, "Incorrect body format", http.StatusBadRequest)
		return
	}

	err = Repo.ShareChatInbox(notify.Receiver, notify.EncSender, notify.EncInboxId, notify.EncSignature, notify.EncSerial, notify.KeyExchangeData)
	if err != nil {
		logger.Println("DB error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Returns new chats for client and deletes the entries from db if successful
func GetNewChats(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger()
	username := r.TLS.PeerCertificates[0].Subject.CommonName

	newChats, err := Repo.GetNewChats(username)
	if err != nil {
		logger.Println("DB error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	chats := &server.ListNewChats{
		Chats: make([]*server.NewChat, 0),
	}
	for _, v := range newChats {
		chats.Chats = append(chats.Chats, &server.NewChat{
			EncSender:       v.EncSender,
			EncInboxCode:    v.EncInboxCode,
			KeyExchangeData: v.KeyExchangeData,
			EncSerial:       v.EncSerial,
			EncSign:         v.EncSignature,
		})
	}

	data, err := proto.Marshal(chats)
	if err != nil {
		logger.Println("Marshal error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	_, err = w.Write(data)
	if err != nil {
		logger.Println("Send error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

	err = Repo.DeleteNewChats(username)
	if err != nil {
		logger.Println("DB delete error:", err)
	}
}

// get encrypted inbox token for specified inbox specified in body (straight bytes, no formatting)
func GetChatToken(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger()
	inboxId, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Println("Body read error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	token, err := Repo.GetToken(inboxId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Inbox not found", http.StatusNotFound)
			return
		}
		logger.Println("DB error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	tokenObj := &server.InboxToken{
		EncToken:        token.EncToken,
		KeyExchangeData: token.KeyExchangeData,
	}

	tokenProto, err := proto.Marshal(tokenObj)
	if err != nil {
		logger.Println("Marshal error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Write(tokenProto)
	w.WriteHeader(http.StatusOK)
}

// Returns new messages from inbox if the provided token is correct
func GetNewMessages(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger()

	bodyTxt, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Println("Body read error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	getMsgs := &server.GetNewMessages{}
	err = proto.Unmarshal(bodyTxt, getMsgs)
	if err != nil {
		http.Error(w, "Incorrect body format", http.StatusBadRequest)
		return
	}

	token, err := Repo.GetToken(getMsgs.InboxId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Inbox not found", http.StatusNotFound)
			return
		}
		logger.Println("DB error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	tokenHash := common.Hash(getMsgs.Token)
	if !bytes.Equal(token.CurrentTokenHash, tokenHash) {
		http.Error(w, "Bad token", http.StatusUnauthorized)
		return
	}

	msgs, err := Repo.GetMessages(getMsgs.InboxId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Inbox not found", http.StatusNotFound)
			return
		}
		logger.Println("DB error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	msgsProto := &server.ListNewMessages{}
	for _, msg := range msgs {
		msgsProto.Msgs = append(msgsProto.Msgs, &server.Message{
			EncMsg: msg.EncMsg,
			Serial: uint64(msg.SerialN),
		})
	}
	result, err := proto.Marshal(msgsProto)
	if err != nil {
		logger.Println("Marshal error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Write(result)
	w.WriteHeader(http.StatusOK)

	// this should be a transaction. look into it
	err = Repo.FlushInbox(getMsgs.InboxId)
	if err != nil {
		logger.Println("DB flush inbox error:", err)
		return
	}

	err = Repo.SetInboxToken(getMsgs.InboxId, nil, nil, nil)
	if err != nil {
		logging.GetLogger().Println("DB error:", err)
		return
	}
}
