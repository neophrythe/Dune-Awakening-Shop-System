package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const itemCols = `id, game_item_id, name, description, category, price, quantity, stock, enabled`

// UpsertItem inserts a new catalog item (when it.ID == 0) or updates the
// existing one. It returns the item id.
func (s *Store) UpsertItem(ctx context.Context, it *CatalogItem) (int64, error) {
	if it.ID == 0 {
		err := s.pool.QueryRow(ctx,
			`INSERT INTO dune_shop.catalog_items
			   (game_item_id, name, description, category, price, quantity, stock, enabled)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
			it.GameItemID, it.Name, it.Description, it.Category,
			it.Price, it.Quantity, it.Stock, it.Enabled).Scan(&it.ID)
		if err != nil {
			return 0, fmt.Errorf("insert item: %w", err)
		}
		return it.ID, nil
	}
	ct, err := s.pool.Exec(ctx,
		`UPDATE dune_shop.catalog_items
		    SET game_item_id=$2, name=$3, description=$4, category=$5,
		        price=$6, quantity=$7, stock=$8, enabled=$9
		  WHERE id=$1`,
		it.ID, it.GameItemID, it.Name, it.Description, it.Category,
		it.Price, it.Quantity, it.Stock, it.Enabled)
	if err != nil {
		return 0, fmt.Errorf("update item: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return 0, ErrNotFound
	}
	return it.ID, nil
}

// GetItem returns a catalog item by id.
func (s *Store) GetItem(ctx context.Context, id int64) (*CatalogItem, error) {
	var it CatalogItem
	var stock *int
	err := s.pool.QueryRow(ctx,
		`SELECT `+itemCols+` FROM dune_shop.catalog_items WHERE id=$1`, id).
		Scan(&it.ID, &it.GameItemID, &it.Name, &it.Description, &it.Category,
			&it.Price, &it.Quantity, &stock, &it.Enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get item: %w", err)
	}
	it.Stock = stock
	return &it, nil
}

// ListItems returns catalog items (optionally only enabled), ordered by
// category then name.
func (s *Store) ListItems(ctx context.Context, onlyEnabled bool) ([]CatalogItem, error) {
	q := `SELECT ` + itemCols + ` FROM dune_shop.catalog_items`
	if onlyEnabled {
		q += ` WHERE enabled=TRUE`
	}
	q += ` ORDER BY category, name`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close()
	var out []CatalogItem
	for rows.Next() {
		var it CatalogItem
		var stock *int
		if err := rows.Scan(&it.ID, &it.GameItemID, &it.Name, &it.Description,
			&it.Category, &it.Price, &it.Quantity, &stock, &it.Enabled); err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		it.Stock = stock
		out = append(out, it)
	}
	return out, rows.Err()
}

// SetItemEnabled toggles an item's availability.
func (s *Store) SetItemEnabled(ctx context.Context, id int64, enabled bool) error {
	ct, err := s.pool.Exec(ctx,
		`UPDATE dune_shop.catalog_items SET enabled=$2 WHERE id=$1`, id, enabled)
	if err != nil {
		return fmt.Errorf("set enabled: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
