-- name: CheckChirpOwnerShip :one
SELECT EXISTS(
SELECT id
FROM chirps
WHERE id = $1
AND user_id = $2);
