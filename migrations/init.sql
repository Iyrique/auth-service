CREATE TABLE refresh_tokens
(
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL,
    token_hash TEXT NOT NULL,
    user_agent TEXT NOT NULL,
    ip_address TEXT NOT NULL,
    used       BOOLEAN   DEFAULT FALSE,
    issued_at  TIMESTAMP DEFAULT now()
);
