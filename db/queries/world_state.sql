-- name: GetOrCreateWorldSeed :one
INSERT INTO world_state (id, seed)
VALUES (1, $1)
ON CONFLICT (id) DO UPDATE
SET seed = world_state.seed
RETURNING seed;
