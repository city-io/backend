-- +goose Up
-- +goose StatementBegin
ALTER TABLE cities ADD COLUMN population_cap DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (population_cap >= 0);
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
ALTER TABLE cities DROP COLUMN population_cap;
-- +goose StatementEnd
