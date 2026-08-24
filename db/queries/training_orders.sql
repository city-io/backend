-- name: GetTrainingOrdersByCity :many
SELECT *
FROM training_orders
WHERE city_id = $1
ORDER BY created_at, training_order_id;

-- name: CreateTrainingOrder :exec
INSERT INTO training_orders (
    training_order_id,
    army_id,
    city_id,
    barracks_id,
    troop_type,
    count,
    population_cost,
    gold_cost,
    started_at,
    completes_at,
    created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: AssignTrainingOrder :exec
UPDATE training_orders
SET
    barracks_id = $2,
    started_at = $3,
    completes_at = $4
WHERE training_order_id = $1
  AND started_at IS NULL;

-- name: DeleteTrainingOrder :exec
DELETE FROM training_orders
WHERE training_order_id = $1;
