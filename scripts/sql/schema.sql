CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    certificate TEXT NOT NULL UNIQUE
);
