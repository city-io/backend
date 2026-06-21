-- +goose Up
-- +goose StatementBegin
ALTER TABLE buildings DROP COLUMN target_level;
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
ALTER TABLE buildings ADD COLUMN target_level INTEGER NOT NULL DEFAULT 1 CHECK (target_level >= 1);
-- +goose StatementEnd
