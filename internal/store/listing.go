package store

import (
	"context"
	"fmt"
)

// LinkedAccountRow is a linked account enriched with its wallet balance, for
// admin listings.
type LinkedAccountRow struct {
	LinkedAccount
	Balance int64 `json:"balance"`
}

// ListLinkedAccounts returns linked accounts with balances, most recent first.
func (s *Store) ListLinkedAccounts(ctx context.Context, limit int) ([]LinkedAccountRow, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx,
		`SELECT la.id, la.discord_user_id, la.game_account_id, la.character_name, la.linked_at,
		        COALESCE(w.balance, 0)
		 FROM dune_shop.linked_accounts la
		 LEFT JOIN dune_shop.wallets w ON w.linked_account_id = la.id
		 ORDER BY la.linked_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list linked accounts: %w", err)
	}
	defer rows.Close()
	var out []LinkedAccountRow
	for rows.Next() {
		var r LinkedAccountRow
		if err := rows.Scan(&r.ID, &r.DiscordUserID, &r.GameAccountID,
			&r.CharacterName, &r.LinkedAt, &r.Balance); err != nil {
			return nil, fmt.Errorf("scan linked account: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Stats is a snapshot of headline numbers for the dashboard.
type Stats struct {
	LinkedAccounts        int64 `json:"linked_accounts"`
	CatalogItems          int64 `json:"catalog_items"`
	Kits                  int64 `json:"kits"`
	CurrencyInCirculation int64 `json:"currency_in_circulation"`
	Purchases             int64 `json:"purchases"`
}

// Stats returns aggregate counts for the dashboard overview.
func (s *Store) Stats(ctx context.Context) (*Stats, error) {
	var st Stats
	err := s.pool.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM dune_shop.linked_accounts),
		  (SELECT count(*) FROM dune_shop.catalog_items),
		  (SELECT count(*) FROM dune_shop.kits),
		  (SELECT COALESCE(sum(balance),0) FROM dune_shop.wallets),
		  (SELECT count(*) FROM dune_shop.transactions WHERE kind='spend')`).
		Scan(&st.LinkedAccounts, &st.CatalogItems, &st.Kits, &st.CurrencyInCirculation, &st.Purchases)
	if err != nil {
		return nil, fmt.Errorf("stats: %w", err)
	}
	return &st, nil
}

// RecentTransactionRow is a ledger entry enriched with the character name.
type RecentTransactionRow struct {
	Transaction
	CharacterName string `json:"character_name"`
}

// RecentTransactions returns the latest ledger entries across all accounts.
func (s *Store) RecentTransactions(ctx context.Context, limit int) ([]RecentTransactionRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx,
		`SELECT t.id, t.linked_account_id, t.kind, t.amount, t.catalog_item_id, t.kit_id,
		        t.delivery, t.note, t.created_at, COALESCE(la.character_name,'')
		 FROM dune_shop.transactions t
		 LEFT JOIN dune_shop.linked_accounts la ON la.id = t.linked_account_id
		 ORDER BY t.created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("recent transactions: %w", err)
	}
	defer rows.Close()
	var out []RecentTransactionRow
	for rows.Next() {
		var r RecentTransactionRow
		if err := rows.Scan(&r.ID, &r.LinkedAccountID, &r.Kind, &r.Amount,
			&r.CatalogItemID, &r.KitID, &r.Delivery, &r.Note, &r.CreatedAt, &r.CharacterName); err != nil {
			return nil, fmt.Errorf("scan recent txn: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
