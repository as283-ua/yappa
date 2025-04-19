// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: queries.sql

package db

import (
	"context"
)

const addMessage = `-- name: AddMessage :exec
INSERT INTO chat_inbox_messages (inbox_code, enc_msg) 
VALUES ($1, $2)
`

type AddMessageParams struct {
	InboxCode []byte
	EncMsg    []byte
}

// -- CHAT MESSAGES
func (q *Queries) AddMessage(ctx context.Context, arg AddMessageParams) error {
	_, err := q.db.Exec(ctx, addMessage, arg.InboxCode, arg.EncMsg)
	return err
}

const changeEcdhTemp = `-- name: ChangeEcdhTemp :exec
UPDATE users
SET ecdh_temp = $2
WHERE username = $1
`

type ChangeEcdhTempParams struct {
	Username string
	EcdhTemp []byte
}

func (q *Queries) ChangeEcdhTemp(ctx context.Context, arg ChangeEcdhTempParams) error {
	_, err := q.db.Exec(ctx, changeEcdhTemp, arg.Username, arg.EcdhTemp)
	return err
}

const createInbox = `-- name: CreateInbox :exec
INSERT INTO chat_inboxes (code, current_token, enc_token) 
VALUES ($1, NULL, NULl)
`

// -- CHAT INBOXES
func (q *Queries) CreateInbox(ctx context.Context, code []byte) error {
	_, err := q.db.Exec(ctx, createInbox, code)
	return err
}

const createUser = `-- name: CreateUser :exec
INSERT INTO users (username, certificate) 
VALUES ($1, $2)
`

type CreateUserParams struct {
	Username    string
	Certificate string
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) error {
	_, err := q.db.Exec(ctx, createUser, arg.Username, arg.Certificate)
	return err
}

const deleteNewUserInboxes = `-- name: DeleteNewUserInboxes :exec
DELETE FROM user_inboxes
WHERE username = $1
`

func (q *Queries) DeleteNewUserInboxes(ctx context.Context, username string) error {
	_, err := q.db.Exec(ctx, deleteNewUserInboxes, username)
	return err
}

const flushInbox = `-- name: FlushInbox :exec
DELETE FROM chat_inbox_messages
WHERE inbox_code = $1
`

func (q *Queries) FlushInbox(ctx context.Context, inboxCode []byte) error {
	_, err := q.db.Exec(ctx, flushInbox, inboxCode)
	return err
}

const getInboxToken = `-- name: GetInboxToken :one
SELECT current_token
FROM chat_inboxes
WHERE code = $1
`

func (q *Queries) GetInboxToken(ctx context.Context, code []byte) ([]byte, error) {
	row := q.db.QueryRow(ctx, getInboxToken, code)
	var current_token []byte
	err := row.Scan(&current_token)
	return current_token, err
}

const getMessages = `-- name: GetMessages :many
SELECT enc_msg
FROM chat_inbox_messages
WHERE inbox_code = $1
`

func (q *Queries) GetMessages(ctx context.Context, inboxCode []byte) ([][]byte, error) {
	rows, err := q.db.Query(ctx, getMessages, inboxCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items [][]byte
	for rows.Next() {
		var enc_msg []byte
		if err := rows.Scan(&enc_msg); err != nil {
			return nil, err
		}
		items = append(items, enc_msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getNewUserInboxes = `-- name: GetNewUserInboxes :many
SELECT enc_inbox_code, ecdh_pub
FROM user_inboxes
WHERE username = $1
`

type GetNewUserInboxesRow struct {
	EncInboxCode []byte
	EcdhPub      []byte
}

func (q *Queries) GetNewUserInboxes(ctx context.Context, username string) ([]GetNewUserInboxesRow, error) {
	rows, err := q.db.Query(ctx, getNewUserInboxes, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetNewUserInboxesRow
	for rows.Next() {
		var i GetNewUserInboxesRow
		if err := rows.Scan(&i.EncInboxCode, &i.EcdhPub); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getUserByUsername = `-- name: GetUserByUsername :one
SELECT id, username, certificate, ecdh_temp
FROM users
WHERE username = $1
`

// -- AUTH
func (q *Queries) GetUserByUsername(ctx context.Context, username string) (User, error) {
	row := q.db.QueryRow(ctx, getUserByUsername, username)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Certificate,
		&i.EcdhTemp,
	)
	return i, err
}

const newUserInbox = `-- name: NewUserInbox :exec
INSERT INTO user_inboxes (username, enc_sender, enc_inbox_code, ecdh_pub)
VALUES ($1, $2, $3, $4)
`

type NewUserInboxParams struct {
	Username     string
	EncSender    []byte
	EncInboxCode []byte
	EcdhPub      []byte
}

// -- USER PERSONAL INBOXES
func (q *Queries) NewUserInbox(ctx context.Context, arg NewUserInboxParams) error {
	_, err := q.db.Exec(ctx, newUserInbox,
		arg.Username,
		arg.EncSender,
		arg.EncInboxCode,
		arg.EcdhPub,
	)
	return err
}

const setToken = `-- name: SetToken :exec
UPDATE chat_inboxes
SET current_token = $2, enc_token = $3
WHERE code = $1
`

type SetTokenParams struct {
	Code         []byte
	CurrentToken []byte
	EncToken     []byte
}

func (q *Queries) SetToken(ctx context.Context, arg SetTokenParams) error {
	_, err := q.db.Exec(ctx, setToken, arg.Code, arg.CurrentToken, arg.EncToken)
	return err
}
