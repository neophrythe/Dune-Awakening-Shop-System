package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// LinkAccount creates or updates the Discord↔game mapping and ensures the
// account has a wallet.
func (s *Store) LinkAccount(ctx context.Context, discordUserID, gameAccountID, characterName string) (*LinkedAccount, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var la LinkedAccount
	err = tx.QueryRow(ctx,
		`INSERT INTO dune_shop.linked_accounts (discord_user_id, game_account_id, character_name)
		 VALUES ($1,$2,$3)
		 ON CONFLICT (discord_user_id) DO UPDATE
		     SET game_account_id=EXCLUDED.game_account_id,
		         character_name=EXCLUDED.character_name
		 RETURNING id, discord_user_id, game_account_id, character_name, linked_at`,
		discordUserID, gameAccountID, characterName).
		Scan(&la.ID, &la.DiscordUserID, &la.GameAccountID, &la.CharacterName, &la.LinkedAt)
	if err != nil {
		return nil, fmt.Errorf("link account: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO dune_shop.wallets (linked_account_id) VALUES ($1)
		 ON CONFLICT (linked_account_id) DO NOTHING`, la.ID); err != nil {
		return nil, fmt.Errorf("ensure wallet: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &la, nil
}

// LinkByDiscord looks up a linked account by Discord user id.
func (s *Store) LinkByDiscord(ctx context.Context, discordUserID string) (*LinkedAccount, error) {
	return s.scanLink(ctx, "discord_user_id", discordUserID)
}

// LinkByGameAccount looks up a linked account by in-game account id.
func (s *Store) LinkByGameAccount(ctx context.Context, gameAccountID string) (*LinkedAccount, error) {
	return s.scanLink(ctx, "game_account_id", gameAccountID)
}

func (s *Store) scanLink(ctx context.Context, col, val string) (*LinkedAccount, error) {
	var la LinkedAccount
	err := s.pool.QueryRow(ctx,
		`SELECT id, discord_user_id, game_account_id, character_name, linked_at
		 FROM dune_shop.linked_accounts WHERE `+col+`=$1`, val).
		Scan(&la.ID, &la.DiscordUserID, &la.GameAccountID, &la.CharacterName, &la.LinkedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get link: %w", err)
	}
	return &la, nil
}
