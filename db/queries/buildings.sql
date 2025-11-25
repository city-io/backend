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