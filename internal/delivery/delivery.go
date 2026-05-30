// Package delivery grants purchased/awarded items to a player in-game, mirroring
// the two paths the game itself uses:
//
//  1. FLS / PlayFab grant — authoritative, account-level. Uses the
//     ServiceAuthToken (self-host JWT) to call Funcom Live Services → PlayFab
//     GrantItemsToUser. Items land in the account inventory and appear on next
//     login; works while the player is offline.
//
//  2. RMQ server command — live, world-level. Publishes a SpawnItem server
//     command to the running game server via the broker. Immediate, but the
//     player must be online.
//
// Ported from the Icehunter/dune-admin delivery path.
package delivery

import (
	"context"
	"fmt"
	"net/http"
)

// Request is a single item delivery to a player. Each engine uses the fields it
// needs (RMQ: PlayerName + AssetItemID; FLS: PlayFabID + PlayFabItemID).
type Request struct {
	PlayerName    string // in-game character name (RMQ SpawnItem)
	PlayFabID     string // PlayFab account id (FLS grant)
	AssetItemID   string // dev/asset item id (RMQ SpawnItem)
	PlayFabItemID string // PlayFab catalog item id (FLS grant)
	Count         int
}

// Engine delivers an item to a player in-game.
type Engine interface {
	Deliver(ctx context.Context, r Request) error
	Name() string
}

// Options configures which engine(s) New builds.
type Options struct {
	Mode string // "fls" | "rmq" | "both" (default "fls")

	// RMQ
	Container string // docker container, e.g. AMP_BuGIsland01
	MQRoot    string // extracted mq dir (default /AMP/duneawakening/extracted/mq)
	MQHome    string // broker HOME with .erlang.cookie (default .../runtime/mq-game-home)
	Node      string // broker node (default rabbit-game@localhost)

	// FLS
	FLSToken       string
	PlayFabTitleID string

	HTTPClient *http.Client // optional, for FLS
}

// New builds the delivery engine for the configured mode. "both" tries FLS first
// (account-level, reliable, works offline) and falls back to RMQ (live) so a
// single purchase is never granted twice.
func New(o Options) (Engine, error) {
	fls := &FLSEngine{Token: o.FLSToken, TitleID: o.PlayFabTitleID, Client: o.HTTPClient}
	rmq := &RMQEngine{Container: o.Container, MQRoot: o.MQRoot, Home: o.MQHome, Node: o.Node}

	switch o.Mode {
	case "", "fls":
		return fls, nil
	case "rmq":
		return rmq, nil
	case "both":
		return &Multi{Engines: []Engine{fls, rmq}}, nil
	default:
		return nil, fmt.Errorf("unknown delivery mode %q (want fls|rmq|both)", o.Mode)
	}
}

// Multi tries its engines in order and succeeds on the first that works.
type Multi struct {
	Engines []Engine
}

// Name implements Engine.
func (m *Multi) Name() string { return "multi" }

// Deliver tries each engine in order, returning nil on the first success.
func (m *Multi) Deliver(ctx context.Context, r Request) error {
	if len(m.Engines) == 0 {
		return fmt.Errorf("no delivery engines configured")
	}
	var lastErr error
	for _, e := range m.Engines {
		if err := e.Deliver(ctx, r); err != nil {
			lastErr = fmt.Errorf("%s: %w", e.Name(), err)
			continue
		}
		return nil
	}
	return fmt.Errorf("all delivery engines failed: %w", lastErr)
}
