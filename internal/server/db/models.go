// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0

package db

type ChatInbox struct {
	Code         []byte
	CurrentToken []byte
	EncToken     []byte
}

type ChatInboxMessage struct {
	ID        int32
	InboxCode []byte
	EncMsg    []byte
}

type User struct {
	ID          int32
	Username    string
	Certificate string
	EcdhTemp    []byte
}

type UserInbox struct {
	Username     string
	EncSender    []byte
	EncInboxCode []byte
	EcdhPub      []byte
}
