-- +goose Up
-- +goose StatementBegin
CREATE TABLE explored_tiles (
    user_id       VARCHAR(36) NOT NULL,
    tile_x        INTEGER NOT NULL,
    tile_y        INTEGER NOT NULL,
    discovered_at TIMESTAMP NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, tile_x, tile_y),
    CONSTRAINT explored_tiles_user_fk
        FOREIGN KEY (user_id) REFERENCES users (user_id)
        ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE explored_tiles;
-- +goose StatementEnd
