-- name: CreateCity :exec
INSERT INTO cities (
    city_id, type, owner, name, population, population_cap,
    start_coord, size
)
VALUES (
    $1, $2, $3, $4, $5, $6, POINT($7, $8), $9
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
    start_coord     = POINT($7, $8),
    size            = $9,
    updated_at      = NOW()
WHERE city_id = $1;

-- name: BatchUpdateCities :exec
UPDATE cities AS c
SET
    type            = v.type,
    owner           = v.owner,
    name            = v.name,
    population      = v.population,
    population_cap  = v.population_cap,
    start_coord     = POINT(v.start_x, v.start_y),
    size            = v.size,
    updated_at      = NOW()
FROM (
    SELECT
        unnest($1::text[])          AS city_id,
        unnest($2::text[])          AS type,
        unnest($3::text[])          AS owner,
        unnest($4::text[])          AS name,
        unnest($5::float8[])        AS population,
        unnest($6::float8[])        AS population_cap,
        unnest($7::int[])           AS start_x,
        unnest($8::int[])           AS start_y,
        unnest($9::int[])           AS size
) AS v
WHERE c.city_id = v.city_id;
