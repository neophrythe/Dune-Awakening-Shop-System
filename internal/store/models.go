// Package store holds the persistent data model and the Postgres data-access
// layer for the shop: linked accounts, wallets, catalogue, kits and the
// transaction ledger.
package store

import "time"

// LinkedAccount maps a Discord user to an in-game Dune account/character.
type LinkedAccount struct {
	ID            int64     `json:"id"`
	DiscordUserID string    `json:"discord_user_id"`
	GameAccountID string    `json:"game_account_id"`
	CharacterName string    `json:"character_name"`
	LinkedAt      time.Time `json:"linked_at"`
}

// Wallet holds a linked account's currency balance.
type Wallet struct {
	LinkedAccountID int64     `json:"linked_account_id"`
	Balance         int64     `json:"balance"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CatalogItem is a purchasable shop entry. GameItemID is the identifier handed
// to the delivery engine to grant the item in-game.
type CatalogItem struct {
	ID          int64  `json:"id"`
	GameItemID  string `json:"game_item_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Price       int64  `json:"price"`
	Quantity    int    `json:"quantity"` // amount delivered per purchase
	Stock       *int   `json:"stock"`    // nil = unlimited
	Enabled     bool   `json:"enabled"`
}

// Kit is a priced bundle that delivers several in-game items in one purchase
// (like the admin tools' "give packs"). Its contents live in []KitItem.
type Kit struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Price       int64     `json:"price"`
	Stock       *int      `json:"stock"` // nil = unlimited
	Enabled     bool      `json:"enabled"`
	Items       []KitItem `json:"items"`
}

// KitItem is one in-game item contained in a Kit.
type KitItem struct {
	ID         int64  `json:"id"`
	KitID      int64  `json:"kit_id"`
	GameItemID string `json:"game_item_id"`
	Name       string `json:"name"`
	Quantity   int    `json:"quantity"`
}

// TxnKind enumerates ledger entry types.
type TxnKind string

const (
	TxnEarn   TxnKind = "earn"   // playtime/vote/payment reward
	TxnSpend  TxnKind = "spend"  // purchase
	TxnAdjust TxnKind = "adjust" // admin adjustment / refund
)

// DeliveryStatus tracks in-game delivery of a purchased item.
type DeliveryStatus string

const (
	DeliveryNone    DeliveryStatus = ""
	DeliveryPending DeliveryStatus = "pending"
	DeliveryDone    DeliveryStatus = "done"
	DeliveryFailed  DeliveryStatus = "failed"
)

// Transaction is an append-only ledger entry. Amount is positive for credits
// (earn/adjust up) and negative for debits (spend).
type Transaction struct {
	ID              int64          `json:"id"`
	LinkedAccountID int64          `json:"linked_account_id"`
	Kind            TxnKind        `json:"kind"`
	Amount          int64          `json:"amount"`
	CatalogItemID   *int64         `json:"catalog_item_id"` // set for single-item purchases
	KitID           *int64         `json:"kit_id"`          // set for kit purchases
	Delivery        DeliveryStatus `json:"delivery"`
	Note            string         `json:"note"`
	CreatedAt       time.Time      `json:"created_at"`
}
