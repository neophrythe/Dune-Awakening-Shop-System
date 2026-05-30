package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// GameAccountByCharacter resolves a character name to its game account id using
// an admin-configured query. The query is operator-supplied configuration and
// must take the character name as $1 and return a single text column. The
// character name itself is passed as a bound parameter (never interpolated), so
// player input cannot inject SQL. Returns ErrNotFound when no character matches.
func (s *Store) GameAccountByCharacter(ctx context.Context, query, character string) (string, error) {
	if query == "" {
		return "", ErrNotFound
	}
	var accountID string
	err := s.pool.QueryRow(ctx, query, character).Scan(&accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("character lookup: %w", err)
	}
	return accountID, nil
}
