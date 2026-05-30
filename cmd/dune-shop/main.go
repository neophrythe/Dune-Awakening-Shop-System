// Command dune-shop runs the Dune Awakening Shop service: a Discord-driven
// in-game shop and playtime economy for a self-hosted Dune: Awakening server.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/config"
	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/delivery"
	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/discord"
	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/economy"
	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/shop"
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

	deliverer, err := delivery.New(delivery.Options{
		Mode:           cfg.Delivery.Mode,
		Container:      cfg.Delivery.AMPContainer,
		MQRoot:         cfg.Delivery.MQRoot,
		MQHome:         cfg.Delivery.MQHome,
		Node:           cfg.Delivery.MQNode,
		FLSToken:       cfg.Delivery.FLSToken,
		PlayFabTitleID: cfg.Delivery.PlayFabTitleID,
	})
	if err != nil {
		log.Fatalf("delivery: %v", err)
	}
	log.Printf("delivery engine: %s", deliverer.Name())

	shopSvc := shop.New(st, deliverer)

	if cfg.Discord.Token != "" {
		bot, err := discord.New(cfg.Discord, st, shopSvc, cfg.Economy.CurrencyName, cfg.Game.CharacterLookupQuery)
		if err != nil {
			log.Fatalf("discord: %v", err)
		}
		if err := bot.Start(); err != nil {
			log.Fatalf("discord start: %v", err)
		}
		defer bot.Stop()
	} else {
		log.Printf("discord token not set — bot disabled")
	}

	if cfg.Economy.Playtime.Enabled {
		worker := &economy.AccrualWorker{
			Store:    st,
			Source:   economy.NewDBSource(st, cfg.Economy.Playtime.OnlineQuery),
			Amount:   cfg.Economy.Playtime.PerMinute,
			Interval: cfg.Economy.Playtime.AccrualDuration(),
			Currency: cfg.Economy.CurrencyName,
		}
		go worker.Run(ctx)
	}

	if cfg.Economy.Votes.Enabled || cfg.Economy.RealMoney.Enabled {
		hooks := &economy.WebhookServer{
			Store:        st,
			Currency:     cfg.Economy.CurrencyName,
			VotesEnabled: cfg.Economy.Votes.Enabled,
			VoteSecret:   cfg.Economy.Votes.Secret,
			VoteReward:   cfg.Economy.Votes.Reward,
			PayEnabled:   cfg.Economy.RealMoney.Enabled,
			PaySecret:    cfg.Economy.RealMoney.WebhookSecret,
		}
		srv := &http.Server{Addr: cfg.ListenAddr, Handler: hooks.Handler(), ReadHeaderTimeout: 5 * time.Second}
		go func() {
			log.Printf("economy webhooks listening on %s", cfg.ListenAddr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("webhook server: %v", err)
			}
		}()
		defer func() { _ = srv.Close() }()
	}

	// TODO(later): admin web panel (internal/web).

	<-ctx.Done()
	log.Printf("shutting down")
}
