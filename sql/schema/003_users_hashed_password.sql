-- +goose Up
ALTER TABLE IF EXISTS users
ADD COLUMN IF NOT EXISTS hashed_password TEXT NOT NULL DEFAULT 'unset';


-- +goose Down
ALTER TABLE IF EXISTS users
DROP COLUMN IF EXISTS hashed_password;
