-- name: CreateMailboxMessage :exec
INSERT INTO mailbox_messages (
    mailbox_message_id,
    recipient_id,
    kind,
    payload,
    created_at,
    read_at
) VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetMailboxMessagesByRecipient :many
SELECT *
FROM mailbox_messages
WHERE recipient_id = $1
ORDER BY created_at DESC
LIMIT 100;

-- name: MarkMailboxMessageRead :one
UPDATE mailbox_messages
SET read_at = COALESCE(read_at, NOW())
WHERE mailbox_message_id = $1
  AND recipient_id = $2
RETURNING *;
