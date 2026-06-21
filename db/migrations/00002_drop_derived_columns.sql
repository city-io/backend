-- +goose Up
-- +goose StatementBegin
ALTER TABLE cities DROP COLUMN population_cap;
ALTER TABLE buildings DROP COLUMN target_level;
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
ALTER TABLE cities ADD COLUMN population_cap DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (population_cap >= 0);
ALTER TABLE buildings ADD COLUMN target_level INTEGER NOT NULL DEFAULT 1 CHECK (target_level >= 1);
-- +goose StatementEnd
