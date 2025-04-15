CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    certificate TEXT NOT NULL UNIQUE
);

CREATE TABLE user_inboxes (
    username TEXT PRIMARY KEY,
    enc_inbox_code BYTEA NOT NULL,
    FOREIGN KEY (username) REFERENCES users(username)
);

CREATE TABLE chat_inboxes (
    code BYTEA PRIMARY KEY,
    current_token BYTEA NOT NULL,
    enc_token BYTEA NOT NULL
);

CREATE TABLE chat_inbox_messages (
    id SERIAL PRIMARY KEY,
    inbox_code INTEGER NOT NULL,
    enc_msg BYTEA NOT NULL,
    FOREIGN KEY (inbox_code) REFERENCES chat_inboxes(code)
);
