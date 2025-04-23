---- AUTH
-- name: GetUserByUsername :one
SELECT id, username, certificate, pub_key_exchange
FROM users
WHERE username = $1;

-- name: CreateUser :exec
INSERT INTO users (username, certificate, pub_key_exchange) 
VALUES ($1, $2, $3);


---- USER PERSONAL INBOXES
-- name: NewUserInbox :exec
INSERT INTO user_inboxes (username, enc_sender, enc_inbox_code, key_exchange_data)
VALUES ($1, $2, $3, $4);

-- name: GetNewUserInboxes :many
SELECT enc_sender, enc_inbox_code, key_exchange_data
FROM user_inboxes
WHERE username = $1;

-- name: DeleteNewUserInboxes :exec
DELETE FROM user_inboxes
WHERE username = $1;


---- CHAT INBOXES
-- name: CreateInbox :exec
INSERT INTO chat_inboxes (code, current_token_hash, enc_token, key_exchange_data) 
VALUES ($1, NULL, NULL, NULL);

-- name: SetToken :exec
UPDATE chat_inboxes
SET current_token_hash = $2, enc_token = $3, key_exchange_data = $4
WHERE code = $1;

-- name: GetInboxToken :one
SELECT current_token_hash, enc_token, key_exchange_data
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
