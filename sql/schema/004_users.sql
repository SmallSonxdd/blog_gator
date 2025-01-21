-- +goose Up
AlTER TABLE feeds
ADD COLUMN last_fetched_at TIMESTAMP;

-- +goose Down
ALTER TABLE feeds 
DROP COLUMN last_fetched_at;