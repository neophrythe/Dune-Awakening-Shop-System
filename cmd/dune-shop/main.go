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

	// TODO(milestones): connect store, start economy worker, Discord bot, web panel.
	log.Printf("scaffold ready — components not yet wired (see docs/ARCHITECTURE.md)")

	<-ctx.Done()
	log.Printf("shutting down")
}
