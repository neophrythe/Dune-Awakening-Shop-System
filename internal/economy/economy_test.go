package economy

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/store"
)

type fakeCrediter struct {
	links    map[string]*store.LinkedAccount
	credited map[int64]int64
}

func newFakeCrediter() *fakeCrediter {
	return &fakeCrediter{
		links:    map[string]*store.LinkedAccount{},
		credited: map[int64]int64{},
	}
}

func (f *fakeCrediter) LinkByGameAccount(_ context.Context, id string) (*store.LinkedAccount, error) {
	if la, ok := f.links[id]; ok {
		return la, nil
	}
	return nil, store.ErrNotFound
}

func (f *fakeCrediter) Credit(_ context.Context, accID, amount int64, _ store.TxnKind, _ string) (int64, error) {
	f.credited[accID] += amount
	return f.credited[accID], nil
}

type fakeSource struct{ ids []string }

func (f fakeSource) OnlinePlayers(context.Context) ([]string, error) { return f.ids, nil }

func TestAccrualTick(t *testing.T) {
	fc := newFakeCrediter()
	fc.links["g1"] = &store.LinkedAccount{ID: 1}
	fc.links["g2"] = &store.LinkedAccount{ID: 2}

	w := &AccrualWorker{
		Store:  fc,
		Source: fakeSource{ids: []string{"g1", "g2", "unlinked"}},
		Amount: 10,
	}
	n := w.tick(context.Background())
	if n != 2 {
		t.Fatalf("credited = %d, want 2", n)
	}
	if fc.credited[1] != 10 || fc.credited[2] != 10 {
		t.Fatalf("balances = %v", fc.credited)
	}
	if _, ok := fc.credited[0]; ok {
		t.Fatal("unlinked player should not be credited")
	}
}

func postJSON(t *testing.T, h http.Handler, path, secret, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	if secret != "" {
		req.Header.Set("X-Webhook-Secret", secret)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestVoteWebhook(t *testing.T) {
	fc := newFakeCrediter()
	fc.links["g1"] = &store.LinkedAccount{ID: 1}
	srv := &WebhookServer{Store: fc, VotesEnabled: true, VoteSecret: "s3cr3t", VoteReward: 100}
	h := srv.Handler()

	if rec := postJSON(t, h, "/webhook/vote", "s3cr3t", `{"game_account_id":"g1"}`); rec.Code != http.StatusOK {
		t.Fatalf("vote ok: code %d", rec.Code)
	}
	if fc.credited[1] != 100 {
		t.Fatalf("vote credit = %d, want 100", fc.credited[1])
	}
	if rec := postJSON(t, h, "/webhook/vote", "wrong", `{"game_account_id":"g1"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad secret: code %d", rec.Code)
	}
	if rec := postJSON(t, h, "/webhook/vote", "s3cr3t", `{"game_account_id":"nope"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("unlinked: code %d", rec.Code)
	}
}

func TestPaymentWebhook(t *testing.T) {
	fc := newFakeCrediter()
	fc.links["g1"] = &store.LinkedAccount{ID: 1}
	srv := &WebhookServer{Store: fc, PayEnabled: true, PaySecret: "pay"}
	h := srv.Handler()

	if rec := postJSON(t, h, "/webhook/payment", "pay", `{"game_account_id":"g1","amount":500}`); rec.Code != http.StatusOK {
		t.Fatalf("payment ok: code %d", rec.Code)
	}
	if fc.credited[1] != 500 {
		t.Fatalf("payment credit = %d, want 500", fc.credited[1])
	}
	if rec := postJSON(t, h, "/webhook/payment", "pay", `{"game_account_id":"g1","amount":0}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad amount: code %d", rec.Code)
	}
}
