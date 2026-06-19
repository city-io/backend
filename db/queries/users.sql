-- name: GetAllUsers :many
SELECT * FROM users;

-- name: GetUserByIdentifier :one
SELECT * FROM users
WHERE email = $1 OR username = $1;

-- name: CreateUser :exec
INSERT INTO users (
    user_id, email, username, password, gold, food
)
VALUES (
    $1, $2, $3, $4, $5, $6
);

-- name: DeleteUser :exec
DELETE FROM users
WHERE user_id = $1;

-- name: UpdateUserStats :exec
UPDATE users
SET
    gold       = $2,
    food       = $3,
    updated_at = NOW()
WHERE user_id = $1;

-- name: UpdateUser :exec
UPDATE users
SET
    username   = $2,
    gold       = $3,
    food       = $4,
    updated_at = NOW()
WHERE user_id = $1;

-- name: BatchUpdateUsers :exec
UPDATE users AS u
SET
    gold             = v.gold,
    food             = v.food,
    food_income_rate = v.food_income_rate,
    food_upkeep_rate = v.food_upkeep_rate,
    updated_at       = NOW()
FROM (
    SELECT
        UNNEST(sqlc.arg(user_ids)::text[])             AS user_id,
        UNNEST(sqlc.arg(golds)::int8[])                AS gold,
        UNNEST(sqlc.arg(foods)::int8[])                AS food,
        UNNEST(sqlc.arg(food_income_rates)::float8[])  AS food_income_rate,
        UNNEST(sqlc.arg(food_upkeep_rates)::float8[])  AS food_upkeep_rate
) AS v
WHERE u.user_id = v.user_id;