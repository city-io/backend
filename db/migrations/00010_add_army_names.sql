-- +goose Up
-- +goose StatementBegin
ALTER TABLE armies
    ADD COLUMN name VARCHAR(32);

UPDATE armies
SET name = 'Army ' || LEFT(army_id, 8);

ALTER TABLE armies
    ALTER COLUMN name SET NOT NULL;

CREATE UNIQUE INDEX armies_owner_name_unique_idx
    ON armies (owner, LOWER(name));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX armies_owner_name_unique_idx;

ALTER TABLE armies
    DROP COLUMN name;
-- +goose StatementEnd
