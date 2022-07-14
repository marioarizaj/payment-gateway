CREATE TABLE IF NOT EXISTS merchants (
    id uuid PRIMARY KEY,
    name varchar NOT NULL,
    address varchar NOT NULL,
    created_at timestamp NOT NULL,
    updated_at timestamp NOT NULL
);