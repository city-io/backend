-- +goose Up
-- +goose StatementBegin
CREATE TABLE armies (
    army_id         VARCHAR(36) PRIMARY KEY,
    owner           VARCHAR(36) NOT NULL,
    coords          COORDINATES NOT NULL,
    troops          JSONB NOT NULL,
    dest_x          INTEGER NULL,
    dest_y          INTEGER NULL,
    upkeep_city_id  VARCHAR(36) NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Armies stack and coexist with buildings and each other, so coords has no
    -- unique constraint.
    CONSTRAINT armies_owner_fk
        FOREIGN KEY (owner) REFERENCES users (user_id)
        ON DELETE CASCADE
);
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE armies;
-- +goose StatementEnd
