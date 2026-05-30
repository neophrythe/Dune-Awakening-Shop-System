// Package economy supplies the three currency sources: a playtime-accrual worker
// that credits online players on an interval, and HTTP webhooks for vote rewards
// and real-money top-ups.
package economy

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/store"
)

// Crediter is the subset of *store.Store the economy needs.
type Crediter interface {
	LinkByGameAccount(ctx context.Context, gameAccountID string) (*store.LinkedAccount, error)
	Credit(ctx context.Context, accountID, amount int64, kind store.TxnKind, note string) (int64, error)
}

// PlayerSource returns the game account ids of currently-online players.
type PlayerSource interface {
	OnlinePlayers(ctx context.Context) ([]string, error)
}

// AccrualWorker periodically credits playtime currency to online, linked players.
type AccrualWorker struct {
	Store    Crediter
	Source   PlayerSource
	Amount   int64
	Interval time.Duration
	Currency string
}

// Run blocks until ctx is cancelled, crediting online players each interval.
func (w *AccrualWorker) Run(ctx context.Context) {
	if w.Amount <= 0 || w.Interval <= 0 || w.Source == nil {
		log.Printf("economy: playtime accrual disabled")
		return
	}
	log.Printf("economy: playtime accrual every %s (+%d %s)", w.Interval, w.Amount, w.Currency)
	t := time.NewTicker(w.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if n := w.tick(ctx); n > 0 {
				log.Printf("economy: credited %d online player(s)", n)
			}
		}
	}
}

// tick credits every online player that has a linked account. It returns how
// many were credited. Unlinked players are skipped silently.
func (w *AccrualWorker) tick(ctx context.Context) int {
	players, err := w.Source.OnlinePlayers(ctx)
	if err != nil {
		log.Printf("economy: online players: %v", err)
		return 0
	}
	credited := 0
	for _, gameID := range players {
		la, err := w.Store.LinkByGameAccount(ctx, gameID)
		if errors.Is(err, store.ErrNotFound) {
			continue
		}
		if err != nil {
			log.Printf("economy: lookup %s: %v", gameID, err)
			continue
		}
		if _, err := w.Store.Credit(ctx, la.ID, w.Amount, store.TxnEarn, "playtime"); err != nil {
			log.Printf("economy: credit %d: %v", la.ID, err)
			continue
		}
		credited++
	}
	return credited
}
