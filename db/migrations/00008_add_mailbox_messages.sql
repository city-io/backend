-- +goose Up
-- +goose StatementBegin
CREATE TABLE mailbox_messages (
    mailbox_message_id VARCHAR(36) PRIMARY KEY,
    recipient_id       VARCHAR(36) NOT NULL,
    kind               VARCHAR(40) NOT NULL,
    payload            JSONB NOT NULL,
    created_at         TIMESTAMP NOT NULL,
    read_at            TIMESTAMP NULL,

    CONSTRAINT mailbox_messages_recipient_fk
        FOREIGN KEY (recipient_id) REFERENCES users (user_id)
        ON DELETE CASCADE
);

CREATE INDEX mailbox_messages_recipient_created_idx
    ON mailbox_messages (recipient_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS mailbox_messages;
-- +goose StatementEnd
