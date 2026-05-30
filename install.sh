#!/usr/bin/env bash
# Dune Awakening Shop — one-command installer.
#
#   curl -fsSL https://raw.githubusercontent.com/neophrythe/Dune-Awakening-Shop-System/main/install.sh | bash
#
# or, from a checkout:   ./install.sh
#
# Installs prerequisites, builds the binary (with the embedded dashboard),
# runs the interactive setup wizard (asks only for your keys/secrets), loads
# the starter catalog, and installs + starts a systemd service.
#
# Env overrides:
#   INSTALL_DIR   install location           (default /opt/dune-shop)
#   SERVICE_NAME  systemd unit name          (default dune-shop)
#   REPO_URL      git remote                 (default the GitHub repo)
#   BRANCH        branch to use              (default main)
#   NO_SYSTEMD=1  skip the systemd step
#   NO_SEED=1     skip loading the starter catalog
#   NO_UI=1       build without the embedded dashboard (no Node needed)
set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-/opt/dune-shop}"
SERVICE_NAME="${SERVICE_NAME:-dune-shop}"
REPO_URL="${REPO_URL:-https://github.com/neophrythe/Dune-Awakening-Shop-System.git}"
BRANCH="${BRANCH:-main}"
GO_VERSION="1.22.5"

log()  { printf '\033[36m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[33m[!]\033[0m %s\n' "$*"; }
die()  { printf '\033[31m[x]\033[0m %s\n' "$*" >&2; exit 1; }

SUDO=""
if [ "$(id -u)" -ne 0 ]; then
  command -v sudo >/dev/null 2>&1 && SUDO="sudo" || die "run as root or install sudo"
fi

install_pkg() {
  if command -v apt-get >/dev/null 2>&1; then
    $SUDO apt-get update -qq && $SUDO apt-get install -y -qq "$@"
  elif command -v dnf >/dev/null 2>&1; then
    $SUDO dnf install -y -q "$@"
  else
    warn "unknown package manager — please install manually: $*"
  fi
}

# --- 1. prerequisites ------------------------------------------------------
log "Checking prerequisites"
command -v git  >/dev/null 2>&1 || install_pkg git
command -v curl >/dev/null 2>&1 || install_pkg curl
command -v make >/dev/null 2>&1 || install_pkg make

need_go() {
  command -v go >/dev/null 2>&1 || return 0
  local v; v="$(go version | grep -oE 'go[0-9]+\.[0-9]+' | head -1 | tr -d 'go')"
  [ "$(printf '%s\n1.22\n' "$v" | sort -V | head -1)" = "1.22" ] && return 1 || return 0
}
if need_go; then
  log "Installing Go ${GO_VERSION}"
  arch="$(uname -m)"; case "$arch" in x86_64) arch=amd64;; aarch64|arm64) arch=arm64;; esac
  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${arch}.tar.gz" -o /tmp/go.tgz
  $SUDO rm -rf /usr/local/go && $SUDO tar -C /usr/local -xzf /tmp/go.tgz
fi
export PATH="/usr/local/go/bin:$PATH"
log "Go: $(go version)"

if [ -z "${NO_UI:-}" ] && ! command -v npm >/dev/null 2>&1; then
  log "Installing Node/npm (for the dashboard UI)"
  install_pkg nodejs npm || { warn "npm install failed — building without UI"; NO_UI=1; }
fi

# --- 2. fetch the source ---------------------------------------------------
if [ -f "go.mod" ] && grep -q "Dune-Awakening-Shop-System" go.mod 2>/dev/null; then
  SRC_DIR="$(pwd)"; log "Using current checkout: $SRC_DIR"
else
  SRC_DIR="$INSTALL_DIR/src"
  if [ -d "$SRC_DIR/.git" ]; then
    log "Updating existing checkout in $SRC_DIR"
    git -C "$SRC_DIR" fetch --depth 1 origin "$BRANCH" && git -C "$SRC_DIR" reset --hard "origin/$BRANCH"
  else
    log "Cloning $REPO_URL"
    $SUDO mkdir -p "$INSTALL_DIR" && $SUDO chown "$(id -u):$(id -g)" "$INSTALL_DIR"
    git clone --depth 1 -b "$BRANCH" "$REPO_URL" "$SRC_DIR"
  fi
fi
cd "$SRC_DIR"

# --- 3. build --------------------------------------------------------------
if [ -n "${NO_UI:-}" ]; then
  log "Building binary (no embedded UI)"; make build-server
else
  log "Building binary with embedded dashboard"
  make build || { warn "UI build failed — falling back to no-UI build"; make build-server; }
fi
BIN="$SRC_DIR/dune-shop"
[ -x "$BIN" ] || die "build produced no binary at $BIN"

# --- 4. configure (wizard) -------------------------------------------------
$SUDO mkdir -p "$INSTALL_DIR" && $SUDO chown "$(id -u):$(id -g)" "$INSTALL_DIR" 2>/dev/null || true
CONFIG="$INSTALL_DIR/config.yaml"
if [ -f "$CONFIG" ]; then
  log "Config exists at $CONFIG (skipping wizard; edit it or run '$BIN setup -o $CONFIG -force')"
else
  log "Running setup wizard"
  "$BIN" setup -o "$CONFIG"
fi
[ -f "$CONFIG" ] || die "no config written — aborting"

# --- 5. seed starter catalog ----------------------------------------------
if [ -z "${NO_SEED:-}" ]; then
  log "Loading starter catalog"
  "$BIN" seed -config "$CONFIG" -file "$SRC_DIR/seed/default-catalog.json" \
    || warn "seed failed (is the DB reachable?) — re-run later: $BIN seed -config $CONFIG"
fi

# --- 6. systemd service ----------------------------------------------------
if [ -z "${NO_SYSTEMD:-}" ] && command -v systemctl >/dev/null 2>&1; then
  UNIT="/etc/systemd/system/${SERVICE_NAME}.service"
  log "Installing systemd service: $UNIT"
  $SUDO tee "$UNIT" >/dev/null <<EOF
[Unit]
Description=Dune Awakening Shop
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=$SRC_DIR
ExecStart=$BIN -config $CONFIG
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
  $SUDO systemctl daemon-reload
  $SUDO systemctl enable --now "$SERVICE_NAME"
  sleep 2
  $SUDO systemctl --no-pager --lines=10 status "$SERVICE_NAME" || true
else
  log "Skipping systemd. Run manually:  $BIN -config $CONFIG"
fi

PORT="$(grep -A4 '^web:' "$CONFIG" | grep listen_addr | grep -oE '[0-9]+$' || echo 8091)"
echo
log "Done. Dashboard: http://<server-ip>:${PORT}"
log "In Discord: players use /link then /shop; admins use /grant, /additem, /addkit."
