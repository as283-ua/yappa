package mock

import (
	"bytes"
	"errors"

	"github.com/as283-ua/yappa/internal/server/db"
)

type MockChatRepo struct {
	userInboxes       map[string][]db.UserInbox
	userInboxSerial   int
	chatInboxes       []db.ChatInbox
	chatInboxMessages []db.ChatInboxMessage
}

func EmptyMockChatRepo() *MockChatRepo {
	return &MockChatRepo{
		userInboxes:       map[string][]db.UserInbox{},
		userInboxSerial:   0,
		chatInboxes:       make([]db.ChatInbox, 0),
		chatInboxMessages: make([]db.ChatInboxMessage, 0),
	}
}

func (r MockChatRepo) GetChatInboxes() []db.ChatInbox {
	return r.chatInboxes
}

func (r MockChatRepo) GetUserInboxes() map[string][]db.UserInbox {
	return r.userInboxes
}

func (r *MockChatRepo) ShareChatInbox(username string, encSender, encInboxCode, ecdhPub []byte) error {
	r.userInboxes[username] = append(r.userInboxes[username], db.UserInbox{
		ID:           int32(r.userInboxSerial),
		Username:     username,
		EncSender:    encSender,
		EncInboxCode: encInboxCode,
		EcdhPub:      ecdhPub,
	})
	r.userInboxSerial++
	return nil
}

func (r *MockChatRepo) CreateChatInbox(inboxCode []byte) error {
	r.chatInboxes = append(r.chatInboxes, db.ChatInbox{
		Code: inboxCode,
	})
	return nil
}

func (r MockChatRepo) GetNewChats(username string) ([]db.GetNewUserInboxesRow, error) {
	result := []db.GetNewUserInboxesRow{}
	for _, v := range r.userInboxes[username] {
		result = append(result, db.GetNewUserInboxesRow{
			EncInboxCode: v.EncInboxCode,
			EncSender:    v.EncSender,
			EcdhPub:      v.EcdhPub,
		})
	}
	return result, nil
}

func (r *MockChatRepo) DeleteNewChats(username string) error {
	r.userInboxes[username] = []db.UserInbox{}
	return nil
}

func (r *MockChatRepo) SetInboxToken(inboxCode, token, encToken []byte) error {
	idx := -1
	for i, v := range r.chatInboxes {
		if bytes.Equal(v.Code, inboxCode) {
			idx = i
			break
		}
	}
	if idx != -1 {
		r.chatInboxes[idx].CurrentToken = token
		r.chatInboxes[idx].EncToken = encToken
	} else {
		return errors.New("inbox not found")
	}
	return nil
}

func (r MockChatRepo) GetToken(inboxCode []byte) (db.GetInboxTokenRow, error) {
	for _, v := range r.chatInboxes {
		if bytes.Equal(v.Code, inboxCode) {
			return db.GetInboxTokenRow{
				CurrentToken: v.CurrentToken,
				EncToken:     v.EncToken,
			}, nil
		}
	}

	return db.GetInboxTokenRow{}, errors.New("inbox not found")
}

func (r *MockChatRepo) AddMessage(inboxCode, encMsg []byte) error {
	_, err := r.GetToken(inboxCode)
	if err != nil {
		return err
	}
	r.chatInboxMessages = append(r.chatInboxMessages, db.ChatInboxMessage{
		InboxCode: inboxCode,
		EncMsg:    encMsg,
	})
	return nil
}

func (r MockChatRepo) GetMessages(inboxCode []byte) ([][]byte, error) {
	result := make([][]byte, 0)
	for _, v := range r.chatInboxMessages {
		if bytes.Equal(v.InboxCode, inboxCode) {
			result = append(result, v.EncMsg)
		}
	}
	return result, nil
}

func (r *MockChatRepo) FlushInbox(inboxCode []byte) error {
	newList := make([]db.ChatInboxMessage, 0)
	for _, v := range r.chatInboxMessages {
		if !bytes.Equal(v.InboxCode, inboxCode) {
			newList = append(newList, v)
		}
	}
	r.chatInboxMessages = newList
	return nil
}
