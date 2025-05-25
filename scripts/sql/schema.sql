DROP TABLE IF EXISTS user_inboxes CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS chat_inboxes CASCADE;
DROP TABLE IF EXISTS chat_inbox_messages CASCADE;

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    certificate TEXT NOT NULL UNIQUE,
    pub_key_exchange BYTEA NOT NULL
);

CREATE TABLE user_inboxes (
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL,
    enc_sender BYTEA NOT NULL,
    enc_signature BYTEA NOT NULL,
    enc_serial BYTEA NOT NULL,
    enc_inbox_code BYTEA NOT NULL,
    key_exchange_data BYTEA NOT NULL,
    FOREIGN KEY (username) REFERENCES users(username)
);

CREATE TABLE chat_inboxes (
    code BYTEA PRIMARY KEY,
    current_token_hash BYTEA,
    enc_token BYTEA,
    key_exchange_data BYTEA
);

CREATE TABLE chat_inbox_messages (
    id SERIAL PRIMARY KEY,
    inbox_code BYTEA NOT NULL,
    enc_msg BYTEA NOT NULL,
    FOREIGN KEY (inbox_code) REFERENCES chat_inboxes(code)
);
