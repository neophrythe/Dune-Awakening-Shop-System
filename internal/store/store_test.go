package store

import (
	"context"
	"errors"
	"os"
	"testing"
)

// testStore connects to the throwaway Postgres named by DUNE_SHOP_TEST_DB and
// resets the dune_shop schema. Tests are skipped when the variable is unset, so
// `go test ./...` stays green in CI without a database.
func testStore(t *testing.T) *Store {
	t.Helper()
	dsn := os.Getenv("DUNE_SHOP_TEST_DB")
	if dsn == "" {
		t.Skip("set DUNE_SHOP_TEST_DB (postgres DSN) to run store integration tests")
	}
	ctx := context.Background()
	s, err := New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if _, err := s.pool.Exec(ctx, `DROP SCHEMA IF EXISTS dune_shop CASCADE`); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(s.Close)
	return s
}

func TestLinkCreditAndPurchase(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)

	la, err := s.LinkAccount(ctx, "discord123", "game-abc", "Muad'Dib")
	if err != nil {
		t.Fatalf("link: %v", err)
	}
	if la.ID == 0 {
		t.Fatal("expected non-zero account id")
	}

	got, err := s.LinkByDiscord(ctx, "discord123")
	if err != nil || got.ID != la.ID {
		t.Fatalf("link by discord: %v %+v", err, got)
	}

	if bal, err := s.Balance(ctx, la.ID); err != nil || bal != 0 {
		t.Fatalf("initial balance: %v %d", err, bal)
	}

	if _, err := s.Credit(ctx, la.ID, 500, TxnEarn, "playtime"); err != nil {
		t.Fatalf("credit: %v", err)
	}

	stock := 1
	it := &CatalogItem{
		GameItemID: "FuelCanister_Large", Name: "Fuel Canister",
		Price: 200, Quantity: 1, Stock: &stock, Enabled: true,
	}
	if _, err := s.UpsertItem(ctx, it); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	txn, bought, err := s.Purchase(ctx, la.ID, it.ID)
	if err != nil {
		t.Fatalf("purchase: %v", err)
	}
	if bought.ID != it.ID || txn.Amount != -200 || txn.Delivery != DeliveryPending {
		t.Fatalf("unexpected purchase: %+v", txn)
	}
	if bal, _ := s.Balance(ctx, la.ID); bal != 300 {
		t.Fatalf("balance after purchase = %d, want 300", bal)
	}

	if _, _, err := s.Purchase(ctx, la.ID, it.ID); !errors.Is(err, ErrOutOfStock) {
		t.Fatalf("expected ErrOutOfStock, got %v", err)
	}

	if err := s.SetDeliveryStatus(ctx, txn.ID, DeliveryDone); err != nil {
		t.Fatalf("set delivery: %v", err)
	}

	txns, err := s.ListTransactions(ctx, la.ID, 10)
	if err != nil || len(txns) != 2 {
		t.Fatalf("list txns: %v len=%d", err, len(txns))
	}
}

func TestKitPurchase(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)

	la, err := s.LinkAccount(ctx, "dk", "gk", "Chani")
	if err != nil {
		t.Fatalf("link: %v", err)
	}
	if _, err := s.Credit(ctx, la.ID, 1000, TxnEarn, "seed"); err != nil {
		t.Fatalf("credit: %v", err)
	}

	stock := 1
	kit := &Kit{
		Name: "Starter Pack", Price: 300, Stock: &stock, Enabled: true,
		Items: []KitItem{
			{GameItemID: "Item_Water", Name: "Water", Quantity: 2},
			{GameItemID: "Item_Spice", Name: "Spice", Quantity: 1},
		},
	}
	kitID, err := s.CreateKit(ctx, kit)
	if err != nil {
		t.Fatalf("create kit: %v", err)
	}

	got, err := s.GetKit(ctx, kitID)
	if err != nil || len(got.Items) != 2 {
		t.Fatalf("get kit: %v items=%d", err, len(got.Items))
	}

	txn, bought, err := s.PurchaseKit(ctx, la.ID, kitID)
	if err != nil {
		t.Fatalf("purchase kit: %v", err)
	}
	if txn.KitID == nil || *txn.KitID != kitID || txn.Amount != -300 {
		t.Fatalf("unexpected txn %+v", txn)
	}
	if len(bought.Items) != 2 {
		t.Fatalf("bought items = %d, want 2", len(bought.Items))
	}
	if bal, _ := s.Balance(ctx, la.ID); bal != 700 {
		t.Fatalf("balance = %d, want 700", bal)
	}
	if _, _, err := s.PurchaseKit(ctx, la.ID, kitID); !errors.Is(err, ErrOutOfStock) {
		t.Fatalf("expected ErrOutOfStock, got %v", err)
	}
}

func TestInsufficientFunds(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	la, err := s.LinkAccount(ctx, "d2", "g2", "Stilgar")
	if err != nil {
		t.Fatalf("link: %v", err)
	}
	it := &CatalogItem{GameItemID: "x", Name: "Pricey", Price: 1000, Quantity: 1, Enabled: true}
	if _, err := s.UpsertItem(ctx, it); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if _, _, err := s.Purchase(ctx, la.ID, it.ID); !errors.Is(err, ErrInsufficientFunds) {
		t.Fatalf("expected ErrInsufficientFunds, got %v", err)
	}
}
