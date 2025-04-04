-- name: UpgradeUserToChirpyRed :exec
UPDATE users
SET is_chirpy_red = true
WHERE is_chirpy_red = false
AND id = $1;
