-- name: UserLogin :one
SELECT *
FROM users
WHERE email = $1;
