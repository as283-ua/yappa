-- name: GetUserByUsername :one
SELECT id, username, certificate
FROM users
WHERE username = $1;

-- name: CreateUser :exec
INSERT INTO users (username, certificate) 
VALUES ($1, $2);

-- name: UpdateUserCert :exec
UPDATE users 
SET certificate = $2
WHERE username = $1;
