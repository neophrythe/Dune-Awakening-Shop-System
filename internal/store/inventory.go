package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ErrNoInventory means the player's backpack inventory could not be located
// (e.g. they have never spawned in-world yet).
var ErrNoInventory = errors.New("player inventory not found")

// BackpackSpace describes free capacity in a player's main backpack
// (dune.inventories.inventory_type = 0).
type BackpackSpace struct {
	InventoryID int64
	MaxSlots    int // -1 = unlimited
	UsedSlots   int
	FreeSlots   int // large number if unlimited
}

// HasRoom reports whether at least needSlots free slots are available.
func (b BackpackSpace) HasRoom(needSlots int) bool {
	if b.MaxSlots < 0 {
		return true
	}
	return b.FreeSlots >= needSlots
}

// BackpackSpaceByGameAccount resolves a player's backpack (via the funcom hex id
// stored in dune.accounts."user") and reports its slot usage. The game DB lives
// in the `dune` schema alongside our `dune_shop` schema, so we can read it
// directly. Returns ErrNoInventory if the player has no backpack row yet.
func (s *Store) BackpackSpaceByGameAccount(ctx context.Context, gameAccountHex string) (*BackpackSpace, error) {
	var inv BackpackSpace
	err := s.pool.QueryRow(ctx, `
		SELECT i.id, COALESCE(i.max_item_count, -1)
		FROM dune.accounts ac
		JOIN dune.actors a    ON a.owner_account_id = ac.id
		JOIN dune.inventories i ON i.actor_id = a.id AND i.inventory_type = 0
		WHERE ac."user" = $1
		LIMIT 1`, gameAccountHex).Scan(&inv.InventoryID, &inv.MaxSlots)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoInventory
	}
	if err != nil {
		return nil, fmt.Errorf("find backpack: %w", err)
	}

	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM dune.items WHERE inventory_id = $1`, inv.InventoryID).
		Scan(&inv.UsedSlots); err != nil {
		return nil, fmt.Errorf("count backpack items: %w", err)
	}
	if inv.MaxSlots < 0 {
		inv.FreeSlots = 1 << 30
	} else {
		inv.FreeSlots = inv.MaxSlots - inv.UsedSlots
		if inv.FreeSlots < 0 {
			inv.FreeSlots = 0
		}
	}
	return &inv, nil
}
