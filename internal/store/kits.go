package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// CreateKit inserts a kit and its items in one transaction, returning the new id.
func (s *Store) CreateKit(ctx context.Context, k *Kit) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var id int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO dune_shop.kits (name, description, category, price, stock, enabled)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		k.Name, k.Description, k.Category, k.Price, k.Stock, k.Enabled).Scan(&id); err != nil {
		return 0, fmt.Errorf("insert kit: %w", err)
	}
	for _, it := range k.Items {
		if _, err := tx.Exec(ctx,
			`INSERT INTO dune_shop.kit_items (kit_id, game_item_id, name, quantity)
			 VALUES ($1,$2,$3,$4)`, id, it.GameItemID, it.Name, it.Quantity); err != nil {
			return 0, fmt.Errorf("insert kit item: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return id, nil
}

// AddKitItem appends an item to an existing kit.
func (s *Store) AddKitItem(ctx context.Context, kitID int64, it KitItem) error {
	ct, err := s.pool.Exec(ctx,
		`INSERT INTO dune_shop.kit_items (kit_id, game_item_id, name, quantity)
		 SELECT $1,$2,$3,$4 WHERE EXISTS (SELECT 1 FROM dune_shop.kits WHERE id=$1)`,
		kitID, it.GameItemID, it.Name, it.Quantity)
	if err != nil {
		return fmt.Errorf("add kit item: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetKit returns a kit with its items.
func (s *Store) GetKit(ctx context.Context, id int64) (*Kit, error) {
	var k Kit
	var stock *int
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, description, category, price, stock, enabled
		 FROM dune_shop.kits WHERE id=$1`, id).
		Scan(&k.ID, &k.Name, &k.Description, &k.Category, &k.Price, &stock, &k.Enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get kit: %w", err)
	}
	k.Stock = stock
	items, err := s.kitItems(ctx, id)
	if err != nil {
		return nil, err
	}
	k.Items = items
	return &k, nil
}

func (s *Store) kitItems(ctx context.Context, kitID int64) ([]KitItem, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, kit_id, game_item_id, name, quantity
		 FROM dune_shop.kit_items WHERE kit_id=$1 ORDER BY id`, kitID)
	if err != nil {
		return nil, fmt.Errorf("kit items: %w", err)
	}
	defer rows.Close()
	var out []KitItem
	for rows.Next() {
		var it KitItem
		if err := rows.Scan(&it.ID, &it.KitID, &it.GameItemID, &it.Name, &it.Quantity); err != nil {
			return nil, fmt.Errorf("scan kit item: %w", err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// ListKits returns kits (optionally only enabled), each with its items, ordered
// by category then name.
func (s *Store) ListKits(ctx context.Context, onlyEnabled bool) ([]Kit, error) {
	q := `SELECT id, name, description, category, price, stock, enabled FROM dune_shop.kits`
	if onlyEnabled {
		q += ` WHERE enabled=TRUE`
	}
	q += ` ORDER BY category, name`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list kits: %w", err)
	}
	defer rows.Close()
	var kits []Kit
	for rows.Next() {
		var k Kit
		var stock *int
		if err := rows.Scan(&k.ID, &k.Name, &k.Description, &k.Category, &k.Price, &stock, &k.Enabled); err != nil {
			return nil, fmt.Errorf("scan kit: %w", err)
		}
		k.Stock = stock
		kits = append(kits, k)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range kits {
		items, err := s.kitItems(ctx, kits[i].ID)
		if err != nil {
			return nil, err
		}
		kits[i].Items = items
	}
	return kits, nil
}

// SetKitEnabled toggles a kit's availability.
func (s *Store) SetKitEnabled(ctx context.Context, id int64, enabled bool) error {
	ct, err := s.pool.Exec(ctx, `UPDATE dune_shop.kits SET enabled=$2 WHERE id=$1`, id, enabled)
	if err != nil {
		return fmt.Errorf("set kit enabled: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// PurchaseKit atomically locks the kit, verifies it is enabled, non-empty and in
// stock, debits the price (ErrInsufficientFunds if too low), decrements limited
// stock and records a pending 'spend' transaction. The caller then delivers
// every item in the returned kit and calls SetDeliveryStatus.
func (s *Store) PurchaseKit(ctx context.Context, accountID, kitID int64) (*Transaction, *Kit, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	var k Kit
	var stock *int
	err = tx.QueryRow(ctx,
		`SELECT id, name, description, category, price, stock, enabled
		 FROM dune_shop.kits WHERE id=$1 FOR UPDATE`, kitID).
		Scan(&k.ID, &k.Name, &k.Description, &k.Category, &k.Price, &stock, &k.Enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("load kit: %w", err)
	}
	k.Stock = stock
	if !k.Enabled {
		return nil, nil, ErrItemUnavailable
	}
	if stock != nil && *stock <= 0 {
		return nil, nil, ErrOutOfStock
	}

	items, err := s.kitItems(ctx, kitID)
	if err != nil {
		return nil, nil, err
	}
	if len(items) == 0 {
		return nil, nil, ErrItemUnavailable
	}
	k.Items = items

	var newBal int64
	err = tx.QueryRow(ctx,
		`UPDATE dune_shop.wallets SET balance=balance-$2, updated_at=now()
		 WHERE linked_account_id=$1 AND balance>=$2 RETURNING balance`,
		accountID, k.Price).Scan(&newBal)
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
		if _, err := tx.Exec(ctx, `UPDATE dune_shop.kits SET stock=stock-1 WHERE id=$1`, kitID); err != nil {
			return nil, nil, fmt.Errorf("decrement kit stock: %w", err)
		}
	}

	t := Transaction{
		LinkedAccountID: accountID,
		Kind:            TxnSpend,
		Amount:          -k.Price,
		KitID:           &k.ID,
		Delivery:        DeliveryPending,
		Note:            "purchase kit: " + k.Name,
	}
	err = tx.QueryRow(ctx,
		`INSERT INTO dune_shop.transactions
		     (linked_account_id, kind, amount, kit_id, delivery, note)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, created_at`,
		t.LinkedAccountID, t.Kind, t.Amount, t.KitID, t.Delivery, t.Note).
		Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("record kit purchase: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return &t, &k, nil
}
