package economy

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/store"
)

// WebhookServer exposes vote-reward and real-money top-up webhooks. Each route
// is protected by a shared secret sent in the X-Webhook-Secret header.
type WebhookServer struct {
	Store    Crediter
	Currency string

	VotesEnabled bool
	VoteSecret   string
	VoteReward   int64

	PayEnabled bool
	PaySecret  string
}

// Handler builds the HTTP routes for the webhooks plus a health check.
func (s *WebhookServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /webhook/vote", s.handleVote)
	mux.HandleFunc("POST /webhook/payment", s.handlePayment)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}

type voteReq struct {
	GameAccountID string `json:"game_account_id"`
}

func (s *WebhookServer) handleVote(w http.ResponseWriter, r *http.Request) {
	if !s.VotesEnabled {
		http.Error(w, "votes disabled", http.StatusNotFound)
		return
	}
	if !secretOK(r, s.VoteSecret) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req voteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GameAccountID == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if s.credit(r.Context(), req.GameAccountID, s.VoteReward, "vote reward", w) {
		writeJSON(w, map[string]any{"ok": true, "credited": s.VoteReward})
	}
}

type payReq struct {
	GameAccountID string `json:"game_account_id"`
	Amount        int64  `json:"amount"`
}

func (s *WebhookServer) handlePayment(w http.ResponseWriter, r *http.Request) {
	if !s.PayEnabled {
		http.Error(w, "payments disabled", http.StatusNotFound)
		return
	}
	if !secretOK(r, s.PaySecret) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req payReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GameAccountID == "" || req.Amount <= 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if s.credit(r.Context(), req.GameAccountID, req.Amount, "real-money top-up", w) {
		writeJSON(w, map[string]any{"ok": true, "credited": req.Amount})
	}
}

// credit resolves the linked account and credits it, writing an HTTP error and
// returning false on failure.
func (s *WebhookServer) credit(ctx context.Context, gameID string, amount int64, note string, w http.ResponseWriter) bool {
	la, err := s.Store.LinkByGameAccount(ctx, gameID)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "account not linked", http.StatusNotFound)
		return false
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return false
	}
	if _, err := s.Store.Credit(ctx, la.ID, amount, store.TxnEarn, note); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return false
	}
	return true
}

func secretOK(r *http.Request, secret string) bool {
	if secret == "" {
		return false
	}
	got := r.Header.Get("X-Webhook-Secret")
	return subtle.ConstantTimeCompare([]byte(got), []byte(secret)) == 1
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
