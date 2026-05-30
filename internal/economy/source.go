package economy

import "context"

// onlineQuerier runs the configured online-players query against the game DB.
// *store.Store satisfies it.
type onlineQuerier interface {
	OnlineGameAccounts(ctx context.Context, query string) ([]string, error)
}

// dbSource reads online players from the game database using an admin-configured
// query that returns one column of game account ids. Kept configurable because
// the exact game-DB table for presence is deployment-specific.
type dbSource struct {
	q     onlineQuerier
	query string
}

// NewDBSource builds a PlayerSource backed by a game-DB query. The query must
// return a single text column (game account id) of currently-online players.
// Returns nil when query is empty (accrual then stays disabled).
func NewDBSource(q onlineQuerier, query string) PlayerSource {
	if query == "" {
		return nil
	}
	return &dbSource{q: q, query: query}
}

func (s *dbSource) OnlinePlayers(ctx context.Context) ([]string, error) {
	return s.q.OnlineGameAccounts(ctx, s.query)
}
