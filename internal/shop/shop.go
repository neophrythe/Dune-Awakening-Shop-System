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
	ErrNoSpace        = errors.New("not enough inventory space")
)

// Purchaser is the subset of *store.Store the shop needs. Defined as an
// interface so the flow can be tested with fakes.
type Purchaser interface {
	LinkByDiscord(ctx context.Context, discordUserID string) (*store.LinkedAccount, error)
	Purchase(ctx context.Context, accountID, itemID int64) (*store.Transaction, *store.CatalogItem, error)
	PurchaseKit(ctx context.Context, accountID, kitID int64) (*store.Transaction, *store.Kit, error)
	SetDeliveryStatus(ctx context.Context, txnID int64, status store.DeliveryStatus) error
	Refund(ctx context.Context, accountID, amount int64, note string) (int64, error)
	Balance(ctx context.Context, accountID int64) (int64, error)
	BackpackSpaceByGameAccount(ctx context.Context, gameAccountHex string) (*store.BackpackSpace, error)
}

// ensureSpace checks the player's backpack has room for needSlots new stacks.
// If the inventory can't be located (player never spawned) it allows the
// purchase — the delivery itself is then the backstop. A read error also does
// not block a sale.
func (svc *Service) ensureSpace(ctx context.Context, link *store.LinkedAccount, needSlots int) error {
	sp, err := svc.store.BackpackSpaceByGameAccount(ctx, link.GameAccountID)
	if err != nil {
		return nil
	}
	if !sp.HasRoom(needSlots) {
		return fmt.Errorf("%w: %d free slot(s), need %d", ErrNoSpace, sp.FreeSlots, needSlots)
	}
	return nil
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

// BuyKitResult describes a completed kit purchase.
type BuyKitResult struct {
	Kit        *store.Kit
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

	// Reject early if the backpack is full, so the player never pays for an
	// item that can't be delivered. One item = one stack slot.
	if err := svc.ensureSpace(ctx, link, 1); err != nil {
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

// BuyKit runs the purchase→deliver→settle flow for a kit: it debits once, then
// delivers every item in the bundle. If any item fails to deliver the whole
// purchase is refunded and the transaction marked failed (all-or-nothing).
func (svc *Service) BuyKit(ctx context.Context, discordUserID string, kitID int64) (*BuyKitResult, error) {
	link, err := svc.store.LinkByDiscord(ctx, discordUserID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrNotLinked
	}
	if err != nil {
		return nil, err
	}

	txn, kit, err := svc.store.PurchaseKit(ctx, link.ID, kitID)
	if err != nil {
		return nil, err
	}

	// One free slot per distinct item line. If too full, refund and abort
	// before delivering anything.
	if serr := svc.ensureSpace(ctx, link, len(kit.Items)); serr != nil {
		_ = svc.store.SetDeliveryStatus(ctx, txn.ID, store.DeliveryFailed)
		_, _ = svc.store.Refund(ctx, link.ID, kit.Price, "refund: not enough inventory space")
		return nil, serr
	}

	for _, it := range kit.Items {
		req := delivery.Request{
			PlayerName:    link.CharacterName,
			PlayFabID:     link.GameAccountID,
			AssetItemID:   it.GameItemID,
			PlayFabItemID: it.GameItemID,
			Count:         it.Quantity,
		}
		if derr := svc.deliver.Deliver(ctx, req); derr != nil {
			_ = svc.store.SetDeliveryStatus(ctx, txn.ID, store.DeliveryFailed)
			_, _ = svc.store.Refund(ctx, link.ID, kit.Price, "refund: kit delivery failed")
			return nil, fmt.Errorf("%w: item %s: %v", ErrDeliveryFailed, it.GameItemID, derr)
		}
	}
	if err := svc.store.SetDeliveryStatus(ctx, txn.ID, store.DeliveryDone); err != nil {
		return nil, fmt.Errorf("mark delivered: %w", err)
	}

	bal, _ := svc.store.Balance(ctx, link.ID)
	return &BuyKitResult{Kit: kit, NewBalance: bal}, nil
}
