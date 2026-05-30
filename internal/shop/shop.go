// Package shop orchestrates the purchase flow: it ties the store's atomic
// purchase to in-game delivery and settles the result (mark delivered, or refund
// on delivery failure). It is deliberately decoupled from Discord so it can be
// unit-tested and reused by the web panel.
package shop

import (
	"context"
	"errors"
	"fmt"

	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/delivery"
	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/store"
)

// Errors surfaced to callers (the Discord bot maps these to user messages).
var (
	ErrNotLinked      = errors.New("account not linked")
	ErrDeliveryFailed = errors.New("delivery failed")
)

// Purchaser is the subset of *store.Store the shop needs. Defined as an
// interface so the flow can be tested with fakes.
type Purchaser interface {
	LinkByDiscord(ctx context.Context, discordUserID string) (*store.LinkedAccount, error)
	Purchase(ctx context.Context, accountID, itemID int64) (*store.Transaction, *store.CatalogItem, error)
	SetDeliveryStatus(ctx context.Context, txnID int64, status store.DeliveryStatus) error
	Refund(ctx context.Context, accountID, amount int64, note string) (int64, error)
	Balance(ctx context.Context, accountID int64) (int64, error)
}

// Service runs purchases against a store and a delivery engine.
type Service struct {
	store   Purchaser
	deliver delivery.Engine
}

// New builds a shop service.
func New(s Purchaser, d delivery.Engine) *Service {
	return &Service{store: s, deliver: d}
}

// BuyResult describes a completed purchase.
type BuyResult struct {
	Item       *store.CatalogItem
	NewBalance int64
}

// Buy runs the full purchase→deliver→settle flow for a Discord user. On a
// delivery failure the spend is refunded and the transaction marked failed.
func (svc *Service) Buy(ctx context.Context, discordUserID string, itemID int64) (*BuyResult, error) {
	link, err := svc.store.LinkByDiscord(ctx, discordUserID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrNotLinked
	}
	if err != nil {
		return nil, err
	}

	txn, item, err := svc.store.Purchase(ctx, link.ID, itemID)
	if err != nil {
		return nil, err // ErrInsufficientFunds, ErrOutOfStock, ErrItemUnavailable, ErrNotFound
	}

	req := delivery.Request{
		PlayerName:    link.CharacterName,
		PlayFabID:     link.GameAccountID,
		AssetItemID:   item.GameItemID,
		PlayFabItemID: item.GameItemID,
		Count:         item.Quantity,
	}
	if derr := svc.deliver.Deliver(ctx, req); derr != nil {
		_ = svc.store.SetDeliveryStatus(ctx, txn.ID, store.DeliveryFailed)
		_, _ = svc.store.Refund(ctx, link.ID, item.Price, "refund: delivery failed")
		return nil, fmt.Errorf("%w: %v", ErrDeliveryFailed, derr)
	}
	if err := svc.store.SetDeliveryStatus(ctx, txn.ID, store.DeliveryDone); err != nil {
		return nil, fmt.Errorf("mark delivered: %w", err)
	}

	bal, _ := svc.store.Balance(ctx, link.ID)
	return &BuyResult{Item: item, NewBalance: bal}, nil
}
