-- +goose Up
-- +goose StatementBegin
-- Which tiles a player has ever seen, as a bitset of MapSize*MapSize bits in
-- row-major order (bit y*width + x). At the current map size that is 5625 bits
-- — 704 bytes per player — so it lives on the user row rather than in a join
-- table of explored coordinates.
--
-- Terrain is remembered once seen: fog re-hides cities, buildings and armies,
-- but the shape of the land stays charted, which is what makes scouting
-- meaningful rather than merely repeated.
ALTER TABLE users ADD COLUMN explored BYTEA NOT NULL DEFAULT ''::bytea;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN explored;
-- +goose StatementEnd
