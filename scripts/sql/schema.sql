CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    certificate TEXT NOT NULL UNIQUE,
    ecdh_temp BYTEA
);

CREATE TABLE user_inboxes (
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL,
    enc_sender BYTEA NOT NULL,
    enc_inbox_code BYTEA NOT NULL,
    ecdh_pub BYTEA NOT NULL,
    FOREIGN KEY (username) REFERENCES users(username)
);

CREATE TABLE chat_inboxes (
    code BYTEA PRIMARY KEY,
    current_token BYTEA,
    enc_token BYTEA
);

CREATE TABLE chat_inbox_messages (
    id SERIAL PRIMARY KEY,
    inbox_code BYTEA NOT NULL,
    enc_msg BYTEA NOT NULL,
    FOREIGN KEY (inbox_code) REFERENCES chat_inboxes(code)
);
