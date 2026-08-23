-- name: GetAllArmies :many
SELECT
    army_id,
    owner,
    (coords).x::int4 AS x,
    (coords).y::int4 AS y,
    troops,
    dest_x,
    dest_y,
    upkeep_city_id
FROM armies;

-- name: CreateArmy :exec
INSERT INTO armies (
    army_id,
    owner,
    coords,
    troops,
    dest_x,
    dest_y,
    upkeep_city_id
)
VALUES (
    sqlc.arg(army_id),
    sqlc.arg(owner),
    ROW(sqlc.arg(x)::int4, sqlc.arg(y)::int4)::coordinates,
    sqlc.arg(troops),
    sqlc.arg(dest_x),
    sqlc.arg(dest_y),
    sqlc.arg(upkeep_city_id)
)
ON CONFLICT (army_id) DO NOTHING;

-- name: DeleteArmy :exec
DELETE FROM armies
WHERE army_id = $1;

-- name: BatchUpdateArmies :exec
UPDATE armies AS a
SET
    owner          = v.owner,
    coords         = ROW(v.x, v.y)::coordinates,
    troops         = v.troops::jsonb,
    dest_x         = NULLIF(v.dest_x, -1),
    dest_y         = NULLIF(v.dest_y, -1),
    upkeep_city_id = NULLIF(v.upkeep_city_id, '')
FROM (
    SELECT
        UNNEST(sqlc.arg(army_ids)::text[])        AS army_id,
        UNNEST(sqlc.arg(owners)::text[])          AS owner,
        UNNEST(sqlc.arg(xs)::int[])               AS x,
        UNNEST(sqlc.arg(ys)::int[])               AS y,
        UNNEST(sqlc.arg(troops_list)::text[])     AS troops,
        UNNEST(sqlc.arg(dest_xs)::int[])          AS dest_x,
        UNNEST(sqlc.arg(dest_ys)::int[])          AS dest_y,
        UNNEST(sqlc.arg(upkeep_city_ids)::text[]) AS upkeep_city_id
) AS v
WHERE a.army_id = v.army_id;

-- name: BatchCreateArmies :exec
INSERT INTO armies (
    army_id,
    owner,
    coords,
    troops,
    dest_x,
    dest_y,
    upkeep_city_id
)
SELECT
    v.army_id,
    v.owner,
    ROW(v.x, v.y)::coordinates,
    v.troops::jsonb,
    NULLIF(v.dest_x, -1),
    NULLIF(v.dest_y, -1),
    NULLIF(v.upkeep_city_id, '')
FROM (
    SELECT
        UNNEST(sqlc.arg(army_ids)::text[])        AS army_id,
        UNNEST(sqlc.arg(owners)::text[])          AS owner,
        UNNEST(sqlc.arg(xs)::int[])               AS x,
        UNNEST(sqlc.arg(ys)::int[])               AS y,
        UNNEST(sqlc.arg(troops_list)::text[])     AS troops,
        UNNEST(sqlc.arg(dest_xs)::int[])          AS dest_x,
        UNNEST(sqlc.arg(dest_ys)::int[])          AS dest_y,
        UNNEST(sqlc.arg(upkeep_city_ids)::text[]) AS upkeep_city_id
) AS v;
