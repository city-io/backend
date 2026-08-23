-- +goose Up
-- +goose StatementBegin
CREATE TABLE training_orders (
    training_order_id  VARCHAR(36) PRIMARY KEY,
    army_id            VARCHAR(36) NOT NULL UNIQUE,
    barracks_id        VARCHAR(36) NOT NULL,
    troop_type         VARCHAR(100) NOT NULL,
    count              BIGINT NOT NULL CHECK (count > 0),
    population_cost    BIGINT NOT NULL CHECK (population_cost > 0),
    gold_cost          BIGINT NOT NULL CHECK (gold_cost >= 0),
    started_at         TIMESTAMP NULL,
    completes_at       TIMESTAMP NULL,
    created_at         TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT training_orders_barracks_fk
        FOREIGN KEY (barracks_id) REFERENCES buildings (building_id)
        ON DELETE RESTRICT
);
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE training_orders;
-- +goose StatementEnd
