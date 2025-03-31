-- name: GetChirp :one
SELECT *
FROM chirps
WHERE user_id = $1;
