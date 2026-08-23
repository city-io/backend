-- name: GetExploredTiles :many
SELECT tile_x, tile_y
FROM explored_tiles
WHERE user_id = $1
ORDER BY tile_y, tile_x;

-- name: AddExploredTiles :exec
INSERT INTO explored_tiles (user_id, tile_x, tile_y)
SELECT sqlc.arg(user_id), discovered.x, discovered.y
FROM (
    SELECT
        UNNEST(sqlc.arg(tile_xs)::int[]) AS x,
        UNNEST(sqlc.arg(tile_ys)::int[]) AS y
) AS discovered
ON CONFLICT (user_id, tile_x, tile_y) DO NOTHING;
