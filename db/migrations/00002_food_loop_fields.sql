-- +goose Up
-- +goose StatementBegin
ALTER TABLE cities
    ADD COLUMN food_production_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN food_upkeep           DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN net_food_flow         DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN starving              BOOLEAN          NOT NULL DEFAULT FALSE;

ALTER TABLE users
    ADD COLUMN food_income_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN food_upkeep_rate DOUBLE PRECISION NOT NULL DEFAULT 0;
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
ALTER TABLE cities
    DROP COLUMN food_production_rate,
    DROP COLUMN food_upkeep,
    DROP COLUMN net_food_flow,
    DROP COLUMN starving;

ALTER TABLE users
    DROP COLUMN food_income_rate,
    DROP COLUMN food_upkeep_rate;
-- +goose StatementEnd
