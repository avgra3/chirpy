-- name: GetChirpByChirpID :one
SELECT *
FROM chirps
WHERE id = $1;
