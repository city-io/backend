-- +goose Up
-- +goose StatementBegin
ALTER TABLE cities
    ADD COLUMN garrison_population DOUBLE PRECISION NOT NULL,
    ADD COLUMN garrison_percent INTEGER NOT NULL,
    ADD COLUMN tax_rate_percent INTEGER NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE cities
    DROP COLUMN tax_rate_percent,
    DROP COLUMN garrison_percent,
    DROP COLUMN garrison_population;
-- +goose StatementEnd
