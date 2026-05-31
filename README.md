<div align="center">

# 🏜️ Dune Awakening Shop System

### A Discord-driven in-game shop & economy for self-hosted *Dune: Awakening* servers

*Players earn currency by playing — browse, buy and receive items straight from Discord.*

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![CI](https://github.com/neophrythe/Dune-Awakening-Shop-System/actions/workflows/ci.yml/badge.svg)](https://github.com/neophrythe/Dune-Awakening-Shop-System/actions)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![Made with discordgo](https://img.shields.io/badge/Discord-discordgo-5865F2?logo=discord&logoColor=white)](https://github.com/bwmarrin/discordgo)

</div>

---

## ✨ Overview

**Dune Awakening Shop** turns your community's Discord into a full storefront for your
self-hosted *Dune: Awakening* world. It runs as a **single Go binary** — no microservice
sprawl, one config, one database.

Members link their game account once, earn an in-game currency just by playing on your
server, then spend it on items that are **delivered directly into the game** — either
account-level through Funcom Live Services / PlayFab, or live to an online player through
the game's message broker.

It brings together two proven ideas:

- the **server-side delivery engine** of [`dune-admin`](https://github.com/Icehunter/dune-admin) — grant items, talk to the game DB and broker, and
- the **Discord shop & economy** loop popularised by community shops for games like *Conan Exiles*

…rebuilt from scratch in Go as one cohesive tool.

---

## 🚀 Features

| | Feature | What it does |
|---|---|---|
| 🪙 | **Three-source economy** | Earn via **playtime**, **vote rewards**, and **real-money top-ups** |
| 🖥️ | **Admin dashboard** | React web panel (login-protected): overview stats, item & kit management, players, live ledger |
| 🛒 | **Storefront** | Categorised item catalogue with prices, quantities and optional stock limits |
| 📦 | **Kits & packs** | Bundle several items into one priced purchase — delivered all-or-nothing |
| 💬 | **Discord-native** | Clean slash commands — `/shop`, `/balance`, `/buy`, `/link` |
| 🔗 | **Account linking** | Bind a Discord user to their in-game character |
| 📦 | **Dual delivery** | **FLS / PlayFab** grant (works offline) **+ live RMQ** spawn — or both with automatic fallback |
| 🛡️ | **Safe purchases** | Atomic debit, stock control, and **automatic refund if delivery fails** |
| 🧾 | **Audit ledger** | Every earn, spend and adjustment is an append-only transaction row |
| 🧰 | **Admin tools** | Role-gated `/grant` and `/additem` straight from Discord |
| 🔌 | **Webhooks** | Secured endpoints for vote sites and payment providers |

---

## 🎮 Discord Commands

| Command | Who | Description |
|---|---|---|
| `/link <character>` | Everyone | Link your Discord to your in-game character |
| `/howtolink` | Everyone | Step-by-step linking help for new players |
| `/balance` | Everyone | Show your current currency balance |
| `/shop` | Everyone | Browse the item catalogue |
| `/buy <item_id>` | Everyone | Purchase an item — delivered in-game instantly |
| `/kits` | Everyone | Browse item **packs/kits** (bundles of items) |
| `/buykit <kit_id>` | Everyone | Buy a kit — all its items delivered at once |
| `/grant <user> <amount>` | 🛡️ Admin | Grant currency to a member |
| `/additem <game_item_id> <name> <price> …` | 🛡️ Admin | Add or update a shop item |
| `/addkit <name> <price> …` | 🛡️ Admin | Create a new kit/pack |
| `/addkititem <kit_id> <game_item_id> …` | 🛡️ Admin | Add an item to a kit |

### 🔗 Linking for non-technical players

Players shouldn't need to hunt for a cryptic ID. When you set a
`game.character_lookup_query` in the config, linking is as simple as:

```
/link name:Muad'Dib
```

The bot resolves the in-game account from the **character name** automatically —
no account id required. The built-in **`/howtolink`** command walks members
through it (and reminds them names are case-sensitive). If you leave the lookup
query empty, `/link` falls back to asking for an account id, and `/howtolink`
explains that flow instead.

---

## 🏗️ Architecture

One binary, four cooperating packages, one Postgres database (its own `dune_shop`
schema — it never touches the game's tables except to read who's online).

```
        Discord  ──slash──▶  internal/discord ──▶ internal/shop ──┬──▶ internal/store    (Postgres: wallets,
   vote / payment ─webhook─▶  internal/economy ───────────────────┤      catalogue, ledger, linked accounts)
                                     │ playtime accrual            │
                                     ▼                             └──▶ internal/delivery (FLS/PlayFab + RMQ)
                              Dune game server  ◀────── grants ─────────────────┘
```

| Package | Responsibility |
|---|---|
| `cmd/dune-shop` | Entry point — load config, wire components, graceful shutdown |
| `internal/store` | Postgres data layer: migrations, wallets, catalogue, atomic purchases, ledger |
| `internal/delivery` | In-game delivery: FLS/PlayFab grant + RMQ `SpawnItem` (mode `fls` / `rmq` / `both`) |
| `internal/shop` | Purchase orchestration: buy → deliver → settle (with refund-on-failure) |
| `internal/discord` | discordgo bot, slash commands, interaction routing |
| `internal/economy` | Playtime accrual worker + vote & payment webhooks |

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the full design.

---

## ⚡ Quick Start

> **Requirements:** a PostgreSQL database (your *Dune: Awakening* server's — the
> same DB whether you run the server via **CubeCoders AMP** or **docker-compose**),
> a Discord bot token, and Go 1.22+ only if you build from source.

### Which delivery mode do I use?

The shop is **not** AMP-specific — it delivers items to any self-hosted Dune server:

| Your Dune server runs as… | Use mode | Why |
|---|---|---|
| CubeCoders **AMP** (Docker) | `rmq` (or `both`) | live instant delivery; the defaults match AMP |
| Plain **docker-compose** container | `rmq` (or `both`) | same path — just set your container name + `mq_root` |
| **Bare metal**, no container | `fls` | account-level grant over HTTPS; no Docker needed |

`rmq` runs `docker exec <container> …` against the server's bundled RabbitMQ, so it
works for **any** containerised Dune server. Bare-metal hosts use `fls` instead.
The setup wizard asks for whichever the chosen mode needs.

### Install — pick one path

**One command (recommended)** — on the host running your Dune server:

```bash
curl -fsSL https://raw.githubusercontent.com/neophrythe/Dune-Awakening-Shop-System/main/install.sh | bash
```

Checks prerequisites, builds the binary (embedded dashboard), runs the **setup
wizard** (asks only for your secrets/IDs), loads the **starter catalog**, and
installs a systemd service.

**Prebuilt binary** — from [Releases](https://github.com/neophrythe/Dune-Awakening-Shop-System/releases):

```bash
tar xzf dune-shop_*_linux_amd64.tar.gz && cd dune-shop_*
./dune-shop setup           # interactive — writes config.yaml
./dune-shop seed            # load the starter catalog
./dune-shop -config config.yaml
```

**Docker / docker-compose:**

```bash
docker compose run --rm shop setup -o /config/config.yaml
docker compose run --rm shop seed -config /config/config.yaml -file /app/seed/default-catalog.json
docker compose up -d
```

**From source:**

```bash
make build                  # or: make build-server  (no embedded dashboard)
./dune-shop setup
./dune-shop seed
./dune-shop -config config.yaml
```

The service migrates its own schema on first run, connects the delivery engine,
starts the Discord bot, and (if enabled) runs the playtime worker and webhook
server.

### 🔐 Secrets via environment

Keep credentials out of `config.yaml` — these env vars override the file:

| Variable | Overrides |
|---|---|
| `DUNE_SHOP_DISCORD_TOKEN` | Discord bot token |
| `DUNE_SHOP_DB_PASS` | Database password |
| `DUNE_SHOP_FLS_TOKEN` | Funcom self-host service token |
| `DUNE_SHOP_PAYMENT_SECRET` | Payment provider secret key |

---

## ⚙️ Configuration

A fully-commented [`config.example.yaml`](config.example.yaml) ships with the repo.
Highlights:

```yaml
economy:
  currency_name: "Spice"
  playtime:   { enabled: true,  per_minute: 10, accrual_interval: "60s" }
  votes:      { enabled: false, reward: 500 }
  realmoney:  { enabled: true,  provider: "manual" }   # admin-confirmed donations

delivery:
  mode: "both"             # fls | rmq | both  (both = FLS first, RMQ fallback)
  # rmq — any containerised Dune server (AMP or docker-compose):
  amp_container: "AMP_BuGIsland01"   # your container name (`docker ps`)
  mq_root: "/AMP/duneawakening/extracted/mq"   # server binaries dir in container
  # fls — bare metal / no Docker:
  fls_token: ""            # ServiceAuthToken (or DUNE_SHOP_FLS_TOKEN)
  playfab_title_id: ""
```

> Don't hand-edit unless you want to — `dune-shop setup` writes a complete,
> valid config from a few prompts (see Quick Start above).

### Webhooks

| Endpoint | Purpose | Auth |
|---|---|---|
| `POST /webhook/vote` | Credit a player for a confirmed vote | `X-Webhook-Secret` header |
| `POST /webhook/payment` | Credit a player after a payment | `X-Webhook-Secret` header |
| `GET /healthz` | Liveness probe | — |

## 🖥️ Admin Dashboard

A login-protected web panel for managing the shop: overview stats, item CRUD,
the kit/pack builder, linked players with balances, and the live transaction
ledger. Built as a React SPA and **embedded into the binary** by `make build`.

Enable it in `config.yaml`:

```yaml
web:
  enabled: true
  listen_addr: "0.0.0.0:8091"
  admin_user: "admin"
  admin_password: ""        # prefer the DUNE_SHOP_WEB_PASSWORD env var
  session_secret: ""        # random string; prefer DUNE_SHOP_WEB_SECRET
```

Then browse to `http://<host>:8091` and sign in. Sessions are stateless,
HMAC-signed cookies (12 h). Run the service behind a TLS-terminating reverse
proxy if it is exposed to the internet.

---

## 🗺️ Roadmap

- [x] Postgres store, wallets & audit ledger
- [x] Dual in-game delivery (FLS/PlayFab + RMQ)
- [x] Discord bot with full shop & admin commands
- [x] Playtime, vote & real-money economy
- [x] Kits / packs (multi-item bundles)
- [x] Web admin dashboard (React SPA, login-protected)
- [ ] First-class Stripe & PayPal signature verification

---

## 🤝 Contributing

Contributions are welcome! Please read [`CONTRIBUTING.md`](CONTRIBUTING.md) — we use
feature branches, Conventional Commits, and CI must stay green (`gofmt`, `go vet`,
`go build`, `go test`).

```bash
gofmt -s -w . && go vet ./... && go build ./... && go test ./...
```

---

## 📜 License

This project is licensed under the **GNU Affero General Public License v3.0** —
see [LICENSE](LICENSE). The AGPL keeps the shop and any hosted/modified version
open: if you run a changed copy as a service, you must share your changes.

> Built on the shoulders of the open-source Dune community tools that came
> before it. Community software, not affiliated with Funcom.
