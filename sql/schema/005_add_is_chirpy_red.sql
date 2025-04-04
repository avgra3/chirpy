-- +goose Up
ALTER TABLE IF EXISTS users
ADD COLUMN IF NOT EXISTS is_chirpy_red BOOLEAN DEFAULT false;


-- +goose Down
ALTER TABLE IF EXISTS users
DROP COLUMN IF EXISTS is_chirpy_red;
