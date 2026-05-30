package store

import (
	"context"
	"fmt"
)

// OnlineGameAccounts runs an admin-configured query (returning one text column
// of game account ids) against the game database. Used by the playtime-accrual
// worker. The query is operator-supplied configuration, not user input.
func (s *Store) OnlineGameAccounts(ctx context.Context, query string) ([]string, error) {
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("online accounts query: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan online account: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
