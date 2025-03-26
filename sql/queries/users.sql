-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email)
VALUES
(GET_RANDOM_UUID(), NOW(), NOW(), $1)
RETURNING *;
