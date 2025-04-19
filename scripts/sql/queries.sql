---- AUTH
-- name: GetUserByUsername :one
SELECT id, username, certificate, ecdh_temp
FROM users
WHERE username = $1;

-- name: CreateUser :exec
INSERT INTO users (username, certificate) 
VALUES ($1, $2);

-- name: ChangeEcdhTemp :exec
UPDATE users
SET ecdh_temp = $2
WHERE username = $1;


---- USER PERSONAL INBOXES
-- name: NewUserInbox :exec
INSERT INTO user_inboxes (username, enc_sender, enc_inbox_code, ecdh_pub)
VALUES ($1, $2, $3, $4);

-- name: GetNewUserInboxes :many
SELECT enc_sender, enc_inbox_code, ecdh_pub
FROM user_inboxes
WHERE username = $1;

-- name: DeleteNewUserInboxes :exec
DELETE FROM user_inboxes
WHERE username = $1;


---- CHAT INBOXES
-- name: CreateInbox :exec
INSERT INTO chat_inboxes (code, current_token, enc_token) 
VALUES ($1, NULL, NULl);

-- name: SetToken :exec
UPDATE chat_inboxes
SET current_token = $2, enc_token = $3
WHERE code = $1;

-- name: GetInboxToken :one
SELECT current_token, enc_token
FROM chat_inboxes
WHERE code = $1;


---- CHAT MESSAGES
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
