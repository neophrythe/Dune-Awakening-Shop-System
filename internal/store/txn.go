package store

import (
	"context"
	"fmt"
)

// ListTransactions returns the most recent ledger entries for an account.
func (s *Store) ListTransactions(ctx context.Context, accountID int64, limit int) ([]Transaction, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, linked_account_id, kind, amount, catalog_item_id, delivery, note, created_at
		 FROM dune_shop.transactions WHERE linked_account_id=$1
		 ORDER BY created_at DESC LIMIT $2`, accountID, limit)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()
	var out []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.LinkedAccountID, &t.Kind, &t.Amount,
			&t.CatalogItemID, &t.Delivery, &t.Note, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan txn: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
