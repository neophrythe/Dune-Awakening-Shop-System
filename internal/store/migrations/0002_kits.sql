-- Kits (a.k.a. packs/bundles): a single priced purchase that delivers several
-- in-game items at once — like the admin tools' "give packs".

CREATE TABLE IF NOT EXISTS dune_shop.kits (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    category    TEXT        NOT NULL DEFAULT '',
    price       BIGINT      NOT NULL CHECK (price >= 0),
    stock       INT,        -- NULL = unlimited
    enabled     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The items contained in a kit.
CREATE TABLE IF NOT EXISTS dune_shop.kit_items (
    id           BIGSERIAL PRIMARY KEY,
    kit_id       BIGINT NOT NULL REFERENCES dune_shop.kits(id) ON DELETE CASCADE,
    game_item_id TEXT   NOT NULL,
    name         TEXT   NOT NULL DEFAULT '',
    quantity     INT    NOT NULL DEFAULT 1 CHECK (quantity > 0)
);

CREATE INDEX IF NOT EXISTS idx_kit_items_kit ON dune_shop.kit_items (kit_id);

-- Link a purchase transaction to the kit it bought (parallels catalog_item_id).
ALTER TABLE dune_shop.transactions
    ADD COLUMN IF NOT EXISTS kit_id BIGINT REFERENCES dune_shop.kits(id);
