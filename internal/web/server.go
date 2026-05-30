// Package web serves the admin dashboard: a JSON API plus (in production
// builds) the embedded React SPA. It is the unified front-end for managing the
// shop — catalogue, kits, linked accounts, wallets and transactions.
package web

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/config"
	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/store"
)

// Server is the dashboard HTTP server.
type Server struct {
	store    *store.Store
	auth     *authenticator
	currency string
	secure   bool
}

// New builds the dashboard server from config.
func New(cfg config.WebConfig, st *store.Store, currency string) *Server {
	return &Server{
		store:    st,
		auth:     newAuthenticator(cfg.AdminUser, cfg.AdminPassword, cfg.SessionSecret),
		currency: currency,
	}
}

// Handler returns the fully-wired HTTP handler (API + SPA).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Public auth endpoints.
	mux.HandleFunc("POST /api/login", s.handleLogin)
	mux.HandleFunc("POST /api/logout", s.handleLogout)
	mux.HandleFunc("GET /api/session", s.handleSession)

	// Protected API.
	mux.HandleFunc("GET /api/stats", s.auth.requireAuth(s.handleStats))
	mux.HandleFunc("GET /api/items", s.auth.requireAuth(s.handleListItems))
	mux.HandleFunc("POST /api/items", s.auth.requireAuth(s.handleUpsertItem))
	mux.HandleFunc("POST /api/items/{id}/enabled", s.auth.requireAuth(s.handleSetItemEnabled))
	mux.HandleFunc("GET /api/kits", s.auth.requireAuth(s.handleListKits))
	mux.HandleFunc("POST /api/kits", s.auth.requireAuth(s.handleCreateKit))
	mux.HandleFunc("POST /api/kits/{id}/items", s.auth.requireAuth(s.handleAddKitItem))
	mux.HandleFunc("POST /api/kits/{id}/enabled", s.auth.requireAuth(s.handleSetKitEnabled))
	mux.HandleFunc("GET /api/accounts", s.auth.requireAuth(s.handleListAccounts))
	mux.HandleFunc("GET /api/transactions", s.auth.requireAuth(s.handleRecentTransactions))

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// SPA (embedded in production builds; dev falls back to ./web/dist).
	if h := spaHandler(); h != nil {
		mux.Handle("/", h)
	}
	return mux
}

// ListenAndServe runs the server until ctx is cancelled.
func (s *Server) ListenAndServe(ctx context.Context, addr string, secure bool) error {
	s.secure = secure
	srv := &http.Server{Addr: addr, Handler: s.Handler(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// ── helpers ─────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func decode(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
