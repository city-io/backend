-- +goose Up
ALTER TABLE buildings
    ALTER COLUMN city_id DROP NOT NULL,
    ADD COLUMN owner VARCHAR(36) NULL,
    ADD CONSTRAINT buildings_owner_fk
        FOREIGN KEY (owner) REFERENCES users (user_id) ON DELETE CASCADE,
    ADD CONSTRAINT buildings_city_or_owner_check
        CHECK ((city_id IS NULL) <> (owner IS NULL));

-- +goose Down
DELETE FROM buildings WHERE city_id IS NULL;

ALTER TABLE buildings
    DROP CONSTRAINT buildings_city_or_owner_check,
    DROP CONSTRAINT buildings_owner_fk,
    DROP COLUMN owner,
    ALTER COLUMN city_id SET NOT NULL;
