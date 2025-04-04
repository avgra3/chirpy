-- +goose Up
CREATE TABLE refresh_tokens(
	token TEXT PRIMARY KEY,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	user_id UUID NOT NULL,
	expires_at TIMESTAMP NOT NULL DEFAULT (NOW()+INTERVAL '60 day'),
	revoked_at TIMESTAMP DEFAULT NULL,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
-- +goose Down
DROP TABLE IF EXISTS refresh_tokens;
