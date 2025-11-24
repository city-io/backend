-- name: GetAllCities :many
SELECT * FROM cities;

-- name: CreateCity :exec
INSERT INTO cities (
    city_id, type, owner, name, population, population_cap,
    start_coord, size
)
VALUES (
    $1, $2, $3, $4, $5, $6, COORDINATES($7, $8), $9
);

-- name: DeleteCity :exec
DELETE FROM cities
WHERE city_id = $1;

-- name: UpdateCity :exec
UPDATE cities
SET
    type            = $2,
    owner           = $3,
    name            = $4,
    population      = $5,
    population_cap  = $6,
    start_coord     = COORDINATES($7, $8),
    size            = $9,
    updated_at      = NOW()
WHERE city_id = $1;

-- name: BatchUpdateCities :exec
UPDATE cities AS c
SET
    type            = v.type,
    owner           = NULLIF(v.owner, ''),
    name            = v.name,
    population      = v.population,
    population_cap  = v.population_cap,
    start_coord     = POINT(v.start_x, v.start_y),
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