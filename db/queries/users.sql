-- name: GetAllUsers :many
SELECT * FROM users;

-- name: CreateUser :exec
INSERT INTO users (
    user_id, email, username, password, gold, food, created_at, updated_at
)
VALUES (
    $1, $2, $3, $4, $5, $6, NOW(), NOW()
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