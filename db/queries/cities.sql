-- name: GetAllCities :many
SELECT
    city_id,
    type,
    owner,
    name,
    population,
    population_cap,
    (start_coords).x::int4 AS start_x,
    (start_coords).y::int4 AS start_y,
    size,
    created_at,
    updated_at
FROM cities;

-- name: CreateCity :exec
INSERT INTO cities (
    city_id,
    type,
    owner,
    name,
    population,
    population_cap,
    start_coords,
    size
)
VALUES (
    sqlc.arg(city_id),
    sqlc.arg(type),
    sqlc.arg(owner),
    sqlc.arg(name),
    sqlc.arg(population),
    sqlc.arg(population_cap),
    ROW(sqlc.arg(start_x)::int4, sqlc.arg(start_y)::int4)::coordinates,
    sqlc.arg(size)
);

-- name: DeleteCity :exec
DELETE FROM cities
WHERE city_id = $1;

-- name: UpdateCity :exec
UPDATE cities
SET
    type            = sqlc.arg(type),
    owner           = sqlc.arg(owner),
    name            = sqlc.arg(name),
    population      = sqlc.arg(population),
    population_cap  = sqlc.arg(population_cap),
    start_coords    = ROW(sqlc.arg(start_x)::int4, sqlc.arg(start_y)::int4)::coordinates,
    size            = sqlc.arg(size),
    updated_at      = NOW()
WHERE city_id = sqlc.arg(city_id);

-- name: FindEmptyCityBlock :one
SELECT
  x::int4 AS x,
  y::int4 AS y
FROM generate_series(0, sqlc.arg(map_width)::int4  - sqlc.arg(size)::int4)  AS x
CROSS JOIN generate_series(0, sqlc.arg(map_height)::int4 - sqlc.arg(size)::int4) AS y
WHERE NOT EXISTS (
  SELECT 1
  FROM cities c
  WHERE
    -- X overlap
    (c.start_coords).x + c.size - 1 >= x
    AND (c.start_coords).x <= x + sqlc.arg(size)::int4 - 1
    -- Y overlap
    AND (c.start_coords).y + c.size - 1 >= y
    AND (c.start_coords).y <= y + sqlc.arg(size)::int4 - 1
)
ORDER BY random()
LIMIT 1;

-- name: BatchCreateCities :exec
INSERT INTO cities (
    city_id,
    type,
    owner,
    name,
    population,
    population_cap,
    start_coords,
    size
)
SELECT
    v.city_id,
    v.type,
    NULLIF(v.owner, ''),
    v.name,
    v.population,
    v.population_cap,
    ROW(v.start_x, v.start_y)::coordinates,
    v.size
FROM (
    SELECT
        UNNEST(sqlc.arg(city_ids)::text[])           AS city_id,
        UNNEST(sqlc.arg(types)::text[])              AS type,
        UNNEST(sqlc.arg(owners)::text[])             AS owner,
        UNNEST(sqlc.arg(names)::text[])              AS name,
        UNNEST(sqlc.arg(populations)::float8[])      AS population,
        UNNEST(sqlc.arg(population_caps)::float8[])  AS population_cap,
        UNNEST(sqlc.arg(start_xs)::int[])            AS start_x,
        UNNEST(sqlc.arg(start_ys)::int[])            AS start_y,
        UNNEST(sqlc.arg(sizes)::int[])               AS size
) AS v;

-- name: GetCitiesByOwner :many
SELECT
    city_id,
    type,
    owner,
    name,
    population,
    population_cap,
    (start_coords).x::int4 AS start_x,
    (start_coords).y::int4 AS start_y,
    size,
    created_at,
    updated_at
FROM cities
WHERE owner = $1;

-- name: BatchUpdateCities :exec
UPDATE cities AS c
SET
    type            = v.type,
    owner           = NULLIF(v.owner, ''),
    name            = v.name,
    population      = v.population,
    population_cap  = v.population_cap,
    start_coords    = ROW(v.start_x, v.start_y)::coordinates,
    size            = v.size,
    updated_at      = NOW()
FROM (
    SELECT
        UNNEST(sqlc.arg(city_ids)::text[])         AS city_id,
        UNNEST(sqlc.arg(types)::text[])            AS type,
        UNNEST(sqlc.arg(owners)::text[])           AS owner,
        UNNEST(sqlc.arg(names)::text[])            AS name,
        UNNEST(sqlc.arg(populations)::float8[])    AS population,
        UNNEST(sqlc.arg(population_caps)::float8[]) AS population_cap,
        UNNEST(sqlc.arg(start_xs)::int[])          AS start_x,
        UNNEST(sqlc.arg(start_ys)::int[])          AS start_y,
        UNNEST(sqlc.arg(sizes)::int[])             AS size
) AS v
WHERE c.city_id = v.city_id;