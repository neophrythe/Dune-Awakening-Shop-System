# Dune Awakening Shop System

An open-source **Discord-driven in-game shop** for self-hosted
[Dune: Awakening](https://duneawakening.com/) servers.

Players earn an in-game currency by playing on the server, browse a shop from
Discord (or a web panel), and have purchased items **delivered straight into
the game** — all handled by a single Go service.

> Status: **early development.** See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
> and the [issues](https://github.com/neophrythe/Dune-Awakening-Shop-System/issues)
> for the roadmap.

## Why

This project consolidates two ideas into one tool:

- the **server-side delivery engine** of [`dune-admin`](https://github.com/Icehunter/dune-admin)
  (grant items in-game via Funcom Live Services / RabbitMQ, read the game DB), and
- the **Discord shop & economy** flow popularised by community shops for games
  like Conan Exiles.

The result is a single self-hostable binary: **Dune Awakening Shop**.

## Features (planned)

- 🪙 **Playtime economy** — players earn currency per minute online on the server
- 🛒 **Shop** — item catalogue with prices, stock limits and categories
- 💬 **Discord bot** — `/shop`, `/balance`, `/buy`, `/link` slash commands
- 🔗 **Account linking** — connect a Discord user to their in-game character
- 📦 **In-game delivery** — purchases are granted directly to the player
- 🛠️ **Admin** — manage catalogue, balances and transactions (web panel + Discord)

## Quick start

```bash
git clone https://github.com/neophrythe/Dune-Awakening-Shop-System.git
cd Dune-Awakening-Shop-System
cp config.example.yaml config.yaml   # then edit
go build ./cmd/dune-shop
./dune-shop -config config.yaml
```

## License

[MIT](LICENSE) © 2026 neophrythe
