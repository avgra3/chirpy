-- name: DeleteChirp :one
WITH deleted AS (
DELETE FROM chirps
WHERE id = $1
AND user_id = $2
RETURNING *)
SELECT COUNT(*) FROM deleted;

