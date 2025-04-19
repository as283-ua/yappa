package chat

import (
	"context"

	"github.com/as283-ua/yappa/internal/server/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepo interface {
	ShareChatInbox(username string, encSender, encInboxCode, ecdhPub []byte) error
	CreateChatInbox(inboxCode []byte) error
	GetNewChats(username string) ([]db.GetNewUserInboxesRow, error)
	DeleteNewChats(username string) error
	SetInboxToken(inboxCode, token, encToken []byte) error
	GetToken(inboxCode []byte) (db.GetInboxTokenRow, error)
	AddMessage(inboxCode, encMsg []byte) error
	GetMessages(inboxCode []byte) ([][]byte, error)
	FlushInbox(inboxCode []byte) error
}

type PgxChatRepo struct {
	Pool *pgxpool.Pool
	Ctx  context.Context
}

var Repo ChatRepo

func (r PgxChatRepo) ShareChatInbox(username string, encSender, encInboxCode, ecdhPub []byte) error {
	queries := db.New(r.Pool)
	return queries.NewUserInbox(r.Ctx, db.NewUserInboxParams{
		Username:     username,
		EncSender:    encSender,
		EncInboxCode: encInboxCode,
		EcdhPub:      ecdhPub,
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

func (r PgxChatRepo) SetInboxToken(inboxCode, token, encToken []byte) error {
	queries := db.New(r.Pool)
	return queries.SetToken(r.Ctx, db.SetTokenParams{
		Code:         inboxCode,
		CurrentToken: token,
		EncToken:     encToken,
	})
}

func (r PgxChatRepo) GetToken(inboxCode []byte) (db.GetInboxTokenRow, error) {
	queries := db.New(r.Pool)
	return queries.GetInboxToken(r.Ctx, inboxCode)
}

func (r PgxChatRepo) AddMessage(inboxCode, encMsg []byte) error {
	queries := db.New(r.Pool)
	return queries.AddMessage(r.Ctx, db.AddMessageParams{
		InboxCode: inboxCode,
		EncMsg:    encMsg,
	})
}

func (r PgxChatRepo) GetMessages(inboxCode []byte) ([][]byte, error) {
	queries := db.New(r.Pool)
	return queries.GetMessages(r.Ctx, inboxCode)
}

func (r PgxChatRepo) FlushInbox(inboxCode []byte) error {
	queries := db.New(r.Pool)
	return queries.FlushInbox(r.Ctx, inboxCode)
}
