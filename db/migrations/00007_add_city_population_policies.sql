-- +goose Up
-- +goose StatementBegin
ALTER TABLE cities
    ADD COLUMN militia_population DOUBLE PRECISION NOT NULL,
    ADD COLUMN militia_percent INTEGER NOT NULL,
    ADD COLUMN tax_rate_percent INTEGER NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE cities
    DROP COLUMN tax_rate_percent,
    DROP COLUMN militia_percent,
    DROP COLUMN militia_population;
-- +goose StatementEnd
