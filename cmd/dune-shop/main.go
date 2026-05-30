// Command dune-shop runs the Dune Awakening Shop service: a Discord-driven
// in-game shop and playtime economy for a self-hosted Dune: Awakening server.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/config"
	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/store"
)

// Version is the build version, overridable at release time via -ldflags.
var Version = "0.1.0-dev"

func main() {
	cfgPath := flag.String("config", "config.yaml", "path to config file")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		log.Printf("dune-shop %s", Version)
		return
	}

	log.Printf("Dune Awakening Shop %s starting", Version)

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	log.Printf("currency=%q playtime=%v(%d/%s) votes=%v realmoney=%v db=%s:%d/%s",
		cfg.Economy.CurrencyName,
		cfg.Economy.Playtime.Enabled, cfg.Economy.Playtime.PerMinute, cfg.Economy.Playtime.AccrualDuration(),
		cfg.Economy.Votes.Enabled, cfg.Economy.RealMoney.Enabled,
		cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.New(ctx, cfg.Database.DSN())
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Printf("store connected and migrated")

	// TODO(milestones): start economy worker, delivery engine, Discord bot, web panel.

	<-ctx.Done()
	log.Printf("shutting down")
}
