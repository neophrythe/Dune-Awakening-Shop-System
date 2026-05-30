// Package store holds the persistent data model and (later) the Postgres
// data-access layer for the shop: linked accounts, wallets, catalogue and the
// transaction ledger.
package store

import "time"

// LinkedAccount maps a Discord user to an in-game Dune account/character.
type LinkedAccount struct {
	ID            int64
	DiscordUserID string
	GameAccountID string
	CharacterName string
	LinkedAt      time.Time
}

// Wallet holds a linked account's currency balance.
type Wallet struct {
	LinkedAccountID int64
	Balance         int64
	UpdatedAt       time.Time
}

// CatalogItem is a purchasable shop entry. GameItemID is the identifier handed
// to the delivery engine to grant the item in-game.
type CatalogItem struct {
	ID          int64
	GameItemID  string
	Name        string
	Description string
	Category    string
	Price       int64
	Quantity    int  // amount delivered per purchase
	Stock       *int // nil = unlimited
	Enabled     bool
}

// TxnKind enumerates ledger entry types.
type TxnKind string

const (
	TxnEarn   TxnKind = "earn"   // playtime reward
	TxnSpend  TxnKind = "spend"  // purchase
	TxnAdjust TxnKind = "adjust" // admin adjustment
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
	ID              int64
	LinkedAccountID int64
	Kind            TxnKind
	Amount          int64
	CatalogItemID   *int64 // set for purchases
	Delivery        DeliveryStatus
	Note            string
	CreatedAt       time.Time
}
