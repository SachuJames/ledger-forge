-- LedgerForge schema, v1.
-- Balances are materialized: every journal entry updates them in the same
-- transaction that writes the entry, so reads never recompute history.

CREATE TABLE IF NOT EXISTS accounts (
    id         TEXT PRIMARY KEY,
    code       TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS entries (
    id              TEXT PRIMARY KEY,
    idempotency_key TEXT UNIQUE,
    narration       TEXT NOT NULL DEFAULT '',
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS entry_lines (
    id         BIGSERIAL PRIMARY KEY,
    entry_id   TEXT NOT NULL REFERENCES entries(id),
    account_id TEXT NOT NULL REFERENCES accounts(id),
    currency   CHAR(3) NOT NULL,
    debit      BIGINT NOT NULL DEFAULT 0 CHECK (debit >= 0),
    credit     BIGINT NOT NULL DEFAULT 0 CHECK (credit >= 0),
    memo       TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_entry_lines_account ON entry_lines(account_id);
CREATE INDEX IF NOT EXISTS idx_entry_lines_entry ON entry_lines(entry_id);

CREATE TABLE IF NOT EXISTS balances (
    account_id TEXT NOT NULL REFERENCES accounts(id),
    currency   CHAR(3) NOT NULL,
    balance    BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (account_id, currency)
);
