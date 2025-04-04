-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password, is_chirpy_red)
VALUES
(GEN_RANDOM_UUID(), NOW(), NOW(), $1, $2, false)
RETURNING *;
