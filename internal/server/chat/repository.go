package chat

import (
	"context"

	"github.com/as283-ua/yappa/internal/server/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepo interface {
	ShareChatInbox(username string, encSender, encInboxCode, encSignature, encSerial, keyExchangeData []byte) error
	CreateChatInbox(inboxCode []byte) error
	GetNewChats(username string) ([]db.GetNewUserInboxesRow, error)
	DeleteNewChats(username string) error
	SetInboxToken(inboxCode, tokenHash, encToken, keyExchangeData []byte) error
	GetToken(inboxCode []byte) (db.GetInboxTokenRow, error)
	AddMessage(inboxCode []byte, serial uint64, encMsg []byte) error
	GetMessages(inboxCode []byte) ([]db.GetMessagesRow, error)
	FlushInbox(inboxCode []byte) error
}

type PgxChatRepo struct {
	Pool *pgxpool.Pool
	Ctx  context.Context
}

var Repo ChatRepo

func (r PgxChatRepo) ShareChatInbox(username string, encSender, encInboxCode, encSignature, encSerial, keyExchangeData []byte) error {
	queries := db.New(r.Pool)
	return queries.NewUserInbox(r.Ctx, db.NewUserInboxParams{
		Username:        username,
		EncSender:       encSender,
		EncInboxCode:    encInboxCode,
		KeyExchangeData: keyExchangeData,
		EncSignature:    encSignature,
		EncSerial:       encSerial,
	})
}

func (r PgxChatRepo) CreateChatInbox(inboxCode []byte) error {
	queries := db.New(r.Pool)
	return queries.CreateInbox(r.Ctx, inboxCode)
}

func (r PgxChatRepo) GetNewChats(username string) ([]db.GetNewUserInboxesRow, error) {
	queries := db.New(r.Pool)
	return queries.GetNewUserInboxes(r.Ctx, username)
}

func (r PgxChatRepo) DeleteNewChats(username string) error {
	queries := db.New(r.Pool)
	return queries.DeleteNewUserInboxes(r.Ctx, username)
}

func (r PgxChatRepo) SetInboxToken(inboxCode, tokenHash, encToken, keyExchangeData []byte) error {
	queries := db.New(r.Pool)
	return queries.SetToken(r.Ctx, db.SetTokenParams{
		Code:             inboxCode,
		CurrentTokenHash: tokenHash,
		EncToken:         encToken,
		KeyExchangeData:  keyExchangeData,
	})
}

func (r PgxChatRepo) GetToken(inboxCode []byte) (db.GetInboxTokenRow, error) {
	queries := db.New(r.Pool)
	return queries.GetInboxToken(r.Ctx, inboxCode)
}

func (r PgxChatRepo) AddMessage(inboxCode []byte, serial uint64, encMsg []byte) error {
	queries := db.New(r.Pool)
	return queries.AddMessage(r.Ctx, db.AddMessageParams{
		InboxCode: inboxCode,
		SerialN:   int64(serial),
		EncMsg:    encMsg,
	})
}

func (r PgxChatRepo) GetMessages(inboxCode []byte) ([]db.GetMessagesRow, error) {
	queries := db.New(r.Pool)
	return queries.GetMessages(r.Ctx, inboxCode)
}

func (r PgxChatRepo) FlushInbox(inboxCode []byte) error {
	queries := db.New(r.Pool)
	return queries.FlushInbox(r.Ctx, inboxCode)
}
