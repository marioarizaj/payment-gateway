CREATE TABLE IF NOT EXISTS payments
(
    id                uuid PRIMARY KEY,
    merchant_id       uuid      NOT NULL references merchants (id),
    payment_status    varchar   NOT NULL DEFAULT 'processing',
    failed_reason     varchar,
    amount            bigint    NOT NULL,
    currency_code     varchar   NOT NULL,
    description       varchar   NOT NULL,
    card_name         varchar   NOT NULL,
    card_number       varchar   NOT NULL,
    card_expiry_month varchar   NOT NULL,
    card_expiry_year  varchar   NOT NULL,
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
