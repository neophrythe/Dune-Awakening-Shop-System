package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Purchase atomically: locks the catalog item, verifies it is enabled and in
// stock, debits the price from the wallet (failing with ErrInsufficientFunds if
// the balance is too low), decrements limited stock, and records a pending
// 'spend' transaction. The caller then performs in-game delivery and calls
// SetDeliveryStatus.
//
// It returns the created transaction and the purchased item.
func (s *Store) Purchase(ctx context.Context, accountID, itemID int64) (*Transaction, *CatalogItem, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	var it CatalogItem
	var stock *int
	err = tx.QueryRow(ctx,
		`SELECT id, game_item_id, name, description, category, price, quantity, stock, enabled
		 FROM dune_shop.catalog_items WHERE id=$1 FOR UPDATE`, itemID).
		Scan(&it.ID, &it.GameItemID, &it.Name, &it.Description, &it.Category,
			&it.Price, &it.Quantity, &stock, &it.Enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("load item: %w", err)
	}
	it.Stock = stock
	if !it.Enabled {
		return nil, nil, ErrItemUnavailable
	}
	if stock != nil && *stock <= 0 {
		return nil, nil, ErrOutOfStock
	}

	var newBal int64
	err = tx.QueryRow(ctx,
		`UPDATE dune_shop.wallets SET balance=balance-$2, updated_at=now()
		 WHERE linked_account_id=$1 AND balance>=$2 RETURNING balance`,
		accountID, it.Price).Scan(&newBal)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if e := tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM dune_shop.wallets WHERE linked_account_id=$1)`,
			accountID).Scan(&exists); e != nil {
			return nil, nil, fmt.Errorf("debit check: %w", e)
		}
		if !exists {
			return nil, nil, ErrNotFound
		}
		return nil, nil, ErrInsufficientFunds
	}
	if err != nil {
		return nil, nil, fmt.Errorf("debit: %w", err)
	}

	if stock != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE dune_shop.catalog_items SET stock=stock-1 WHERE id=$1`, itemID); err != nil {
			return nil, nil, fmt.Errorf("decrement stock: %w", err)
		}
	}

	t := Transaction{
		LinkedAccountID: accountID,
		Kind:            TxnSpend,
		Amount:          -it.Price,
		CatalogItemID:   &it.ID,
		Delivery:        DeliveryPending,
		Note:            "purchase: " + it.Name,
	}
	err = tx.QueryRow(ctx,
		`INSERT INTO dune_shop.transactions
		     (linked_account_id, kind, amount, catalog_item_id, delivery, note)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, created_at`,
		t.LinkedAccountID, t.Kind, t.Amount, t.CatalogItemID, t.Delivery, t.Note).
		Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("record purchase: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return &t, &it, nil
}

// SetDeliveryStatus updates a transaction's delivery status (e.g. done/failed).
func (s *Store) SetDeliveryStatus(ctx context.Context, txnID int64, status DeliveryStatus) error {
	ct, err := s.pool.Exec(ctx,
		`UPDATE dune_shop.transactions SET delivery=$2 WHERE id=$1`, txnID, status)
	if err != nil {
		return fmt.Errorf("set delivery: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Refund credits a previously-spent amount back to the wallet, recording an
// 'adjust' entry. Use it when delivery fails after a successful debit.
func (s *Store) Refund(ctx context.Context, accountID, amount int64, note string) (int64, error) {
	return s.Credit(ctx, accountID, amount, TxnAdjust, note)
}
