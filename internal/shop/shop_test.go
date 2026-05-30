package shop

import (
	"context"
	"errors"
	"testing"

	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/delivery"
	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/store"
)

type fakeStore struct {
	link        *store.LinkedAccount
	linkErr     error
	txn         *store.Transaction
	item        *store.CatalogItem
	purchaseErr error
	statusSet   store.DeliveryStatus
	refunded    int64
	balance     int64
}

func (f *fakeStore) LinkByDiscord(context.Context, string) (*store.LinkedAccount, error) {
	return f.link, f.linkErr
}
func (f *fakeStore) Purchase(context.Context, int64, int64) (*store.Transaction, *store.CatalogItem, error) {
	return f.txn, f.item, f.purchaseErr
}
func (f *fakeStore) SetDeliveryStatus(_ context.Context, _ int64, s store.DeliveryStatus) error {
	f.statusSet = s
	return nil
}
func (f *fakeStore) Refund(_ context.Context, _ int64, amount int64, _ string) (int64, error) {
	f.refunded = amount
	return 0, nil
}
func (f *fakeStore) Balance(context.Context, int64) (int64, error) { return f.balance, nil }

type fakeDeliver struct {
	err    error
	called bool
}

func (f *fakeDeliver) Name() string { return "fake" }
func (f *fakeDeliver) Deliver(context.Context, delivery.Request) error {
	f.called = true
	return f.err
}

func baseStore() *fakeStore {
	return &fakeStore{
		link:    &store.LinkedAccount{ID: 1, CharacterName: "Paul", GameAccountID: "pf1"},
		txn:     &store.Transaction{ID: 9, Amount: -200},
		item:    &store.CatalogItem{ID: 5, GameItemID: "Item_Water", Price: 200, Quantity: 1},
		balance: 300,
	}
}

func TestBuySuccess(t *testing.T) {
	fs := baseStore()
	fd := &fakeDeliver{}
	res, err := New(fs, fd).Buy(context.Background(), "discord1", 5)
	if err != nil {
		t.Fatalf("buy: %v", err)
	}
	if !fd.called {
		t.Fatal("expected delivery to be attempted")
	}
	if fs.statusSet != store.DeliveryDone {
		t.Fatalf("status = %q, want done", fs.statusSet)
	}
	if fs.refunded != 0 {
		t.Fatalf("unexpected refund %d", fs.refunded)
	}
	if res.NewBalance != 300 || res.Item.ID != 5 {
		t.Fatalf("unexpected result %+v", res)
	}
}

func TestBuyNotLinked(t *testing.T) {
	fs := baseStore()
	fs.link, fs.linkErr = nil, store.ErrNotFound
	if _, err := New(fs, &fakeDeliver{}).Buy(context.Background(), "x", 5); !errors.Is(err, ErrNotLinked) {
		t.Fatalf("expected ErrNotLinked, got %v", err)
	}
}

func TestBuyInsufficientFunds(t *testing.T) {
	fs := baseStore()
	fs.txn, fs.item, fs.purchaseErr = nil, nil, store.ErrInsufficientFunds
	if _, err := New(fs, &fakeDeliver{}).Buy(context.Background(), "discord1", 5); !errors.Is(err, store.ErrInsufficientFunds) {
		t.Fatalf("expected ErrInsufficientFunds, got %v", err)
	}
}

func TestBuyDeliveryFailureRefunds(t *testing.T) {
	fs := baseStore()
	fd := &fakeDeliver{err: errors.New("broker down")}
	_, err := New(fs, fd).Buy(context.Background(), "discord1", 5)
	if !errors.Is(err, ErrDeliveryFailed) {
		t.Fatalf("expected ErrDeliveryFailed, got %v", err)
	}
	if fs.statusSet != store.DeliveryFailed {
		t.Fatalf("status = %q, want failed", fs.statusSet)
	}
	if fs.refunded != 200 {
		t.Fatalf("refund = %d, want 200", fs.refunded)
	}
}
