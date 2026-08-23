-- +goose Up
-- +goose StatementBegin
ALTER TABLE cities
    ADD COLUMN garrison_population DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (garrison_population >= 0),
    ADD COLUMN garrison_percent INTEGER NOT NULL DEFAULT 10 CHECK (garrison_percent BETWEEN 5 AND 30),
    ADD COLUMN tax_rate_percent INTEGER NOT NULL DEFAULT 10 CHECK (tax_rate_percent BETWEEN 0 AND 100);

UPDATE cities
SET
    garrison_percent = CASE WHEN type = 'town' THEN 30 ELSE 10 END,
    tax_rate_percent = CASE WHEN type = 'town' THEN 0 ELSE 10 END,
    garrison_population = LEAST(
        population,
        population_cap * CASE WHEN type = 'town' THEN 0.30 ELSE 0.10 END
    );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE cities
    DROP COLUMN tax_rate_percent,
    DROP COLUMN garrison_percent,
    DROP COLUMN garrison_population;
-- +goose StatementEnd
