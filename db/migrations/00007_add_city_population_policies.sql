-- +goose Up
-- +goose StatementBegin
ALTER TABLE cities
    ADD COLUMN population_basis DOUBLE PRECISION NOT NULL,
    ADD COLUMN militia_population DOUBLE PRECISION NOT NULL,
    ADD COLUMN militia_target DOUBLE PRECISION NOT NULL,
    ADD COLUMN tax_rate_percent INTEGER NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE cities
    DROP COLUMN IF EXISTS tax_rate_percent,
    DROP COLUMN IF EXISTS militia_target,
    DROP COLUMN IF EXISTS militia_percent,
    DROP COLUMN IF EXISTS militia_population,
    DROP COLUMN IF EXISTS population_basis,
    -- Older revisions of this unreleased migration used garrison names. The
    -- development reset must be able to tear down either shape.
    DROP COLUMN IF EXISTS garrison_percent,
    DROP COLUMN IF EXISTS garrison_population;
-- +goose StatementEnd
