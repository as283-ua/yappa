-- name: GetUserByUsername :one
SELECT id, username, certificate
FROM users
WHERE username = $1;

-- name: CreateUser :exec
INSERT INTO users (username, certificate) 
VALUES ($1, $2);

-- name: NewUserInbox :exec
INSERT INTO user_inboxes (username, enc_inbox_code)
VALUES ($1, $2);

-- name: GetNewUserInboxes :many
SELECT enc_inbox_code
FROM user_inboxes
WHERE username = $1;

-- name: CreateInbox :exec
INSERT INTO chat_inboxes (code, current_token, enc_token) 
VALUES ($1, $2, $3);

-- name: SetToken :exec
UPDATE chat_inboxes
SET current_token = $2, enc_token = $3
WHERE code = $1;

-- name: GetInboxToken :one
SELECT current_token
FROM chat_inboxes
WHERE code = $1;

-- name: AddMessage :exec
INSERT INTO chat_inbox_messages (inbox_code, enc_msg) 
VALUES ($1, $2);

-- name: GetMessages :many
SELECT enc_msg
FROM chat_inbox_messages
WHERE inbox_code = $1;

-- name: FlushInbox :exec
DELETE FROM chat_inbox_messages
WHERE inbox_code = $1;
