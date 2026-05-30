package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/config"
	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/store"
)

// seedDoc mirrors seed/default-catalog.json.
type seedDoc struct {
	Items []store.CatalogItem `json:"items"`
	Kits  []store.Kit         `json:"kits"`
}

// runSeed imports a catalog/kit JSON into the shop database. It is idempotent:
// catalog items and kits whose name already exists are skipped, so it is safe
// to re-run after editing the file. Exits non-zero on fatal errors.
func runSeed(args []string) {
	fs := flag.NewFlagSet("seed", flag.ExitOnError)
	file := fs.String("file", "seed/default-catalog.json", "seed JSON file to import")
	cfgPath := fs.String("config", "config.yaml", "path to config file")
	_ = fs.Parse(args)

	raw, err := os.ReadFile(*file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed: read file:", err)
		os.Exit(1)
	}
	var doc seedDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		fmt.Fprintln(os.Stderr, "seed: parse json:", err)
		os.Exit(1)
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed: config:", err)
		os.Exit(1)
	}
	ctx := context.Background()
	st, err := store.New(ctx, cfg.Database.DSN())
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed: store:", err)
		os.Exit(1)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "seed: migrate:", err)
		os.Exit(1)
	}

	// existing names → skip sets (idempotent re-runs)
	haveItem := map[string]bool{}
	if cur, err := st.ListItems(ctx, false); err == nil {
		for _, c := range cur {
			haveItem[c.Name] = true
		}
	}
	haveKit := map[string]bool{}
	if cur, err := st.ListKits(ctx, false); err == nil {
		for _, k := range cur {
			haveKit[k.Name] = true
		}
	}

	var addItems, skipItems, addKits, skipKits int
	for i := range doc.Items {
		it := doc.Items[i]
		if haveItem[it.Name] {
			skipItems++
			continue
		}
		it.ID = 0
		it.Enabled = true
		if it.Quantity <= 0 {
			it.Quantity = 1
		}
		if _, err := st.UpsertItem(ctx, &it); err != nil {
			fmt.Fprintln(os.Stderr, "seed: add item", it.Name, ":", err)
			continue
		}
		addItems++
	}
	for i := range doc.Kits {
		k := doc.Kits[i]
		if haveKit[k.Name] {
			skipKits++
			continue
		}
		k.ID = 0
		k.Enabled = true
		if _, err := st.CreateKit(ctx, &k); err != nil {
			fmt.Fprintln(os.Stderr, "seed: add kit", k.Name, ":", err)
			continue
		}
		addKits++
	}

	fmt.Printf("seed done: items +%d (%d existing skipped), kits +%d (%d existing skipped)\n",
		addItems, skipItems, addKits, skipKits)
}
