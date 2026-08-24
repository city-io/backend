-- +goose Up
-- +goose StatementBegin
ALTER TABLE training_orders
    ADD COLUMN city_id VARCHAR(36);

UPDATE training_orders AS training
SET city_id = buildings.city_id
FROM buildings
WHERE buildings.building_id = training.barracks_id;

ALTER TABLE training_orders
    ALTER COLUMN city_id SET NOT NULL,
    ALTER COLUMN barracks_id DROP NOT NULL,
    ADD CONSTRAINT training_orders_city_fk
        FOREIGN KEY (city_id) REFERENCES cities (city_id)
        ON DELETE RESTRICT;

-- Old per-barracks queues carried the eventual barracks on waiting rows.
-- Waiting work is deliberately unassigned in the city pipeline.
UPDATE training_orders
SET barracks_id = NULL
WHERE started_at IS NULL;

CREATE INDEX training_orders_city_queue_idx
    ON training_orders (city_id, created_at, training_order_id);

CREATE UNIQUE INDEX training_orders_active_barracks_idx
    ON training_orders (barracks_id)
    WHERE started_at IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM training_orders
WHERE barracks_id IS NULL;

DROP INDEX training_orders_active_barracks_idx;
DROP INDEX training_orders_city_queue_idx;

ALTER TABLE training_orders
    ALTER COLUMN barracks_id SET NOT NULL,
    DROP CONSTRAINT training_orders_city_fk,
    DROP COLUMN city_id;
-- +goose StatementEnd
