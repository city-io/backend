-- name: GetAllCities :many
SELECT
    city_id,
    type,
    owner,
    name,
    population,
    population_cap,
    militia_population,
    militia_percent,
    tax_rate_percent,
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
    militia_population,
    militia_percent,
    tax_rate_percent,
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
    sqlc.arg(militia_population),
    sqlc.arg(militia_percent),
    sqlc.arg(tax_rate_percent),
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
    militia_population = sqlc.arg(militia_population),
    militia_percent = sqlc.arg(militia_percent),
    tax_rate_percent = sqlc.arg(tax_rate_percent),
    start_coords    = ROW(sqlc.arg(start_x)::int4, sqlc.arg(start_y)::int4)::coordinates,
    size            = sqlc.arg(size),
    updated_at      = NOW()
WHERE city_id = sqlc.arg(city_id);

-- name: BatchCreateCities :exec
INSERT INTO cities (
    city_id,
    type,
    owner,
    name,
    population,
    population_cap,
    militia_population,
    militia_percent,
    tax_rate_percent,
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
    v.militia_population,
    v.militia_percent,
    v.tax_rate_percent,
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
        UNNEST(sqlc.arg(militia_populations)::float8[]) AS militia_population,
        UNNEST(sqlc.arg(militia_percents)::int[])   AS militia_percent,
        UNNEST(sqlc.arg(tax_rate_percents)::int[])  AS tax_rate_percent,
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
    militia_population,
    militia_percent,
    tax_rate_percent,
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
    militia_population = v.militia_population,
    militia_percent = v.militia_percent,
    tax_rate_percent = v.tax_rate_percent,
    start_coords    = ROW(v.start_x, v.start_y)::coordinates,
    size            = v.size
FROM (
    SELECT
        UNNEST(sqlc.arg(city_ids)::text[])          AS city_id,
        UNNEST(sqlc.arg(types)::text[])             AS type,
        UNNEST(sqlc.arg(owners)::text[])            AS owner,
        UNNEST(sqlc.arg(names)::text[])             AS name,
        UNNEST(sqlc.arg(populations)::float8[])     AS population,
        UNNEST(sqlc.arg(population_caps)::float8[]) AS population_cap,
        UNNEST(sqlc.arg(militia_populations)::float8[]) AS militia_population,
        UNNEST(sqlc.arg(militia_percents)::int[])  AS militia_percent,
        UNNEST(sqlc.arg(tax_rate_percents)::int[])  AS tax_rate_percent,
        UNNEST(sqlc.arg(start_xs)::int[])           AS start_x,
        UNNEST(sqlc.arg(start_ys)::int[])           AS start_y,
        UNNEST(sqlc.arg(sizes)::int[])              AS size
) AS v
WHERE c.city_id = v.city_id;
