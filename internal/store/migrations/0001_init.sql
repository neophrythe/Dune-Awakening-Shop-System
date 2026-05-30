-- Dune Awakening Shop — initial schema.
-- Lives in its own schema (dune_shop) so it never collides with the game's
-- own `dune` schema in the same database.

CREATE SCHEMA IF NOT EXISTS dune_shop;

-- Maps a Discord user to their in-game Dune account/character.
CREATE TABLE IF NOT EXISTS dune_shop.linked_accounts (
    id              BIGSERIAL PRIMARY KEY,
    discord_user_id TEXT        NOT NULL UNIQUE,
    game_account_id TEXT        NOT NULL UNIQUE,
    character_name  TEXT        NOT NULL DEFAULT '',
    linked_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One currency wallet per linked account. Balance can never go negative.
CREATE TABLE IF NOT EXISTS dune_shop.wallets (
    linked_account_id BIGINT      PRIMARY KEY
                       REFERENCES dune_shop.linked_accounts(id) ON DELETE CASCADE,
    balance           BIGINT      NOT NULL DEFAULT 0 CHECK (balance >= 0),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Purchasable shop entries.
CREATE TABLE IF NOT EXISTS dune_shop.catalog_items (
    id           BIGSERIAL PRIMARY KEY,
    game_item_id TEXT        NOT NULL,
    name         TEXT        NOT NULL,
    description  TEXT        NOT NULL DEFAULT '',
    category     TEXT        NOT NULL DEFAULT '',
    price        BIGINT      NOT NULL CHECK (price >= 0),
    quantity     INT         NOT NULL DEFAULT 1 CHECK (quantity > 0),
    stock        INT,        -- NULL = unlimited
    enabled      BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Append-only ledger. amount > 0 credits, amount < 0 debits.
CREATE TABLE IF NOT EXISTS dune_shop.transactions (
    id                BIGSERIAL PRIMARY KEY,
    linked_account_id BIGINT      NOT NULL
                       REFERENCES dune_shop.linked_accounts(id) ON DELETE CASCADE,
    kind              TEXT        NOT NULL,
    amount            BIGINT      NOT NULL,
    catalog_item_id   BIGINT      REFERENCES dune_shop.catalog_items(id),
    delivery          TEXT        NOT NULL DEFAULT '',
    note              TEXT        NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_transactions_account_time
    ON dune_shop.transactions (linked_account_id, created_at DESC);
