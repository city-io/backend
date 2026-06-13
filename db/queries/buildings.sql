-- name: GetAllBuildings :many
SELECT
    building_id,
    city_id,
    type,
    level,
    target_level,
    (coords).x::int4 AS x,
    (coords).y::int4 AS y,
    construction_start,
    construction_end
FROM buildings;

-- name: GetBuildingsByCity :many
SELECT
    building_id,
    city_id,
    type,
    level,
    target_level,
    (coords).x::int4 AS x,
    (coords).y::int4 AS y,
    construction_start,
    construction_end
FROM buildings
WHERE city_id = $1;

-- name: CreateBuilding :exec
INSERT INTO buildings (
    building_id,
    city_id,
    type,
    level,
    target_level,
    coords,
    construction_start,
    construction_end
)
VALUES (
    sqlc.arg(building_id),
    sqlc.arg(city_id),
    sqlc.arg(type),
    sqlc.arg(level),
    sqlc.arg(target_level),
    ROW(sqlc.arg(x)::int4, sqlc.arg(y)::int4)::coordinates,
    sqlc.arg(construction_start),
    sqlc.arg(construction_end)
);

-- name: DeleteBuilding :exec
DELETE FROM buildings
WHERE building_id = $1;

-- name: BatchUpdateBuildings :exec
UPDATE buildings AS b
SET
    city_id            = v.city_id,
    type               = v.type,
    level              = v.level,
    target_level       = v.target_level,
    coords             = ROW(v.x, v.y)::coordinates,
    construction_start = v.construction_start,
    construction_end   = v.construction_end,
    updated_at         = NOW()
FROM (
    SELECT
        UNNEST(sqlc.arg(building_ids)::text[])             AS building_id,
        UNNEST(sqlc.arg(city_ids)::text[])                 AS city_id,
        UNNEST(sqlc.arg(types)::text[])                    AS type,
        UNNEST(sqlc.arg(levels)::int[])                    AS level,
        UNNEST(sqlc.arg(target_levels)::int[])             AS target_level,
        UNNEST(sqlc.arg(xs)::int[])                        AS x,
        UNNEST(sqlc.arg(ys)::int[])                         AS y,
        UNNEST(sqlc.arg(construction_starts)::timestamp[]) AS construction_start,
        UNNEST(sqlc.arg(construction_ends)::timestamp[])   AS construction_end
) AS v
WHERE b.building_id = v.building_id;

-- name: BatchCreateBuildings :exec
INSERT INTO buildings (
    building_id,
    city_id,
    type,
    level,
    target_level,
    coords,
    construction_start,
    construction_end
)
SELECT
    v.building_id,
    v.city_id,
    v.type,
    v.level,
    v.target_level,
    ROW(v.x, v.y)::coordinates,
    v.construction_start,
    v.construction_end
FROM (
    SELECT
        UNNEST(sqlc.arg(building_ids)::text[])              AS building_id,
        UNNEST(sqlc.arg(city_ids)::text[])                  AS city_id,
        UNNEST(sqlc.arg(types)::text[])                     AS type,
        UNNEST(sqlc.arg(levels)::int[])                     AS level,
        UNNEST(sqlc.arg(target_levels)::int[])              AS target_level,
        UNNEST(sqlc.arg(xs)::int[])                         AS x,
        UNNEST(sqlc.arg(ys)::int[])                         AS y,
        UNNEST(sqlc.arg(construction_starts)::timestamp[]) AS construction_start,
        UNNEST(sqlc.arg(construction_ends)::timestamp[])   AS construction_end
) AS v;
