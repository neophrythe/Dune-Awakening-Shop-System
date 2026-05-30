# Changelog

All notable changes to this project are documented here.

## [0.7.0] - 2026-05-31
### Added
- **Setup wizard** (`dune-shop setup`): interactive prompts that ask only for
  server-unique secrets/IDs and write a complete, valid `config.yaml` (mode 600)
  with proven defaults for a standard CubeCoders AMP layout.
- **Seed command** (`dune-shop seed`): idempotent import of a catalog + kits
  JSON; entries whose name already exists are skipped, safe to re-run.
- **`seed/default-catalog.json`**: curated, deliverable starter inventory
  (74 items across Special Weapons, Weapons, Armor, vehicle parts and tools,
  plus 3 vehicle starter kits) so a fresh install ships with a working shop.
- **`install.sh`**: one-command installer — prerequisites, build (with embedded
  dashboard), wizard, seed, and a systemd service.
- **Docker**: `Dockerfile` (multi-stage, static binary + docker CLI for RMQ
  delivery) and `docker-compose.yml`.
- **Release workflow**: tagged `v*` pushes build prebuilt Linux amd64/arm64
  binaries (dashboard embedded) and attach them to a GitHub Release.
- README **Quick Start** covering all four install paths.
