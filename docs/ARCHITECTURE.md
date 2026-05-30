# Architecture

Dune Awakening Shop is a **single Go service** that combines a Discord bot, a
shop/economy domain, and an in-game delivery engine against a self-hosted
Dune: Awakening server (run via CubeCoders AMP + Docker, or any topology where
the game DB and RabbitMQ broker are reachable).

```
                    ┌──────────────────────────────────────────┐
   Discord  ───────▶│  internal/discord   (discordgo bot)       │
   (slash cmds)     │   /shop /balance /buy /link + admin       │
                    └───────────────┬──────────────────────────┘
                                    │ calls
                    ┌───────────────▼──────────────────────────┐
   Web panel ──────▶│  internal/shop      (purchase flow)       │
   (later)          │   catalogue · pricing · stock · checkout  │
                    └───────┬───────────────────────┬──────────┘
                            │                        │
              ┌─────────────▼──────┐    ┌────────────▼─────────────┐
              │ internal/store     │    │ internal/delivery        │
              │  wallets,          │    │  grant items in-game via │
              │  transactions,     │    │  FLS / RabbitMQ          │
              │  catalogue,        │    │  (ported from dune-admin)│
              │  linked accounts   │    └────────────┬─────────────┘
              │  (Postgres)        │                 │
              └─────────┬──────────┘                 │
                        │                            │
        ┌───────────────▼──────────┐     ┌───────────▼───────────────┐
        │ internal/economy         │     │ Dune game server           │
        │  playtime accrual worker │◀────│  (DB online-players, RMQ)  │
        └──────────────────────────┘     └────────────────────────────┘
```

## Packages

| Package | Responsibility |
|---|---|
| `cmd/dune-shop` | Entry point: load config, wire components, run, graceful shutdown |
| `internal/config` | YAML config + env overrides for secrets |
| `internal/store` | Postgres data layer: wallets, transactions, catalogue, linked accounts |
| `internal/economy` | Currency sources: playtime accrual, vote-reward webhooks, real-money top-ups (Stripe/PayPal) |
| `internal/delivery` | Deliver purchased items into the game (FLS/RabbitMQ) |
| `internal/shop` | Shop domain: catalogue, pricing, stock, checkout/purchase flow |
| `internal/discord` | discordgo bot, slash commands, interaction handlers |
| `internal/web` | (later) Admin/shop HTTP API + panel |

## Data model (initial)

- **LinkedAccount** — maps a Discord user ID ↔ in-game account/character.
- **Wallet** — per linked account: currency balance.
- **CatalogItem** — purchasable entry: in-game item id, display name, price,
  optional stock/limits, category.
- **Transaction** — append-only ledger: earn (playtime), spend (purchase),
  admin adjustment, with delivery status.

## Design principles

- One binary, one config, one database — easy to self-host.
- Delivery is idempotent and auditable (every grant has a transaction row).
- Discord and web are thin front-ends over the same `shop` + `store` core.
- Secrets (bot token, FLS token, DB password) come from env or a 600-mode file,
  never from committed config.
