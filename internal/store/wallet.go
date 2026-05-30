package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Balance returns the current wallet balance for a linked account.
func (s *Store) Balance(ctx context.Context, accountID int64) (int64, error) {
	var bal int64
	err := s.pool.QueryRow(ctx,
		`SELECT balance FROM dune_shop.wallets WHERE linked_account_id=$1`, accountID).Scan(&bal)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("balance: %w", err)
	}
	return bal, nil
}

// Credit adds amount (>= 0) to the wallet and records a ledger entry. It returns
// the new balance. Use it for earn (playtime/votes), real-money top-ups and
// admin adjustments.
func (s *Store) Credit(ctx context.Context, accountID, amount int64, kind TxnKind, note string) (int64, error) {
	if amount < 0 {
		return 0, fmt.Errorf("credit amount must be >= 0")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var bal int64
	err = tx.QueryRow(ctx,
		`UPDATE dune_shop.wallets SET balance=balance+$2, updated_at=now()
		 WHERE linked_account_id=$1 RETURNING balance`, accountID, amount).Scan(&bal)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("credit: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO dune_shop.transactions (linked_account_id, kind, amount, note)
		 VALUES ($1,$2,$3,$4)`, accountID, kind, amount, note); err != nil {
		return 0, fmt.Errorf("record credit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return bal, nil
}
