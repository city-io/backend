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

-- name: UpdateUser :exec
UPDATE users
SET
    email      = $2,
    username   = $3,
    password   = $4,
    gold       = $5,
    food       = $6,
    updated_at = NOW()
WHERE user_id = $1;

-- name: BatchUpdateUsers :exec
UPDATE users AS u
SET
    email      = v.email,
    username   = v.username,
    password   = v.password,
    gold       = v.gold,
    food       = v.food,
    updated_at = NOW()
FROM (
    SELECT
        unnest($1::text[])     AS user_id,
        unnest($2::text[])     AS email,
        unnest($3::text[])     AS username,
        unnest($4::text[])     AS password,
        unnest($5::bigint[])   AS gold,
        unnest($6::bigint[])   AS food
) AS v
WHERE u.user_id = v.user_id;
