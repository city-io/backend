-- +goose Up
-- +goose StatementBegin
CREATE TABLE world_state (
    id   SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    seed BIGINT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE world_state;
-- +goose StatementEnd
