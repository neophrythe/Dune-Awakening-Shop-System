package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/config"
	"gopkg.in/yaml.v3"
)

// Proven defaults for a standard Funcom Dune: Awakening + CubeCoders AMP setup,
// so an operator can accept most prompts and only fill in their own secrets/IDs.
const (
	defCharLookup = `SELECT ac."user" FROM dune.accounts ac JOIN dune.player_state ps ON ps.account_id = ac.id WHERE ps.character_name = $1 LIMIT 1`
	defOnlineQ    = `SELECT ac."user" FROM dune.accounts ac JOIN dune.player_state ps ON ps.account_id = ac.id WHERE ps.last_login_at IS NOT NULL AND ps.last_logout_at IS NULL`
)

// runSetup is an interactive wizard that writes a complete, valid config.yaml
// from a handful of prompts. It only asks for values unique to a server (DB
// password, Discord token/IDs, AMP container, payment link); everything else
// has a sensible default the operator can accept with Enter.
func runSetup(args []string) {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	out := fs.String("o", "config.yaml", "output config file path")
	force := fs.Bool("force", false, "overwrite an existing config without asking")
	_ = fs.Parse(args)

	r := bufio.NewReader(os.Stdin)
	if _, err := os.Stat(*out); err == nil && !*force {
		if !askYesNo(r, fmt.Sprintf("%s already exists — overwrite?", *out), false) {
			fmt.Println("setup cancelled.")
			return
		}
	}

	fmt.Println("=== Dune Awakening Shop — setup wizard ===")
	fmt.Println("Press Enter to accept the [default]. Required fields are marked *.")

	var c config.Config
	c.ListenAddr = ask(r, "Webhook/health listen address", "0.0.0.0:8090")

	fmt.Println("\n-- Database (your Dune server's Postgres) --")
	c.Database.Host = ask(r, "DB host", "127.0.0.1")
	c.Database.Port = askInt(r, "DB port", 15432)
	c.Database.User = ask(r, "DB user", "postgres")
	c.Database.Password = ask(r, "DB password *", "")
	c.Database.Name = ask(r, "DB name", "dune")
	c.Database.Schema = ask(r, "Game schema", "dune")

	fmt.Println("\n-- Delivery (how items are granted in-game) --")
	fmt.Println("  rmq  = live, instant; needs the game server in a Docker container (AMP or compose)")
	fmt.Println("  fls  = account-level via PlayFab; works offline and without Docker (bare metal)")
	fmt.Println("  both = try fls first, fall back to rmq")
	c.Delivery.Mode = ask(r, "Delivery mode (rmq/fls/both)", "rmq")
	if c.Delivery.Mode == "rmq" || c.Delivery.Mode == "both" {
		fmt.Println("  (find the container with `docker ps`; AMP names look like AMP_Dune01)")
		c.Delivery.AMPContainer = ask(r, "Game server container name *", "")
		c.Delivery.MQNode = ask(r, "RabbitMQ node", "rabbit@dune01")
		c.Delivery.MQRoot = ask(r, "Server Linux binaries dir inside the container (mq_root)",
			"/AMP/instances/Dune01/0/Dune/DuneServer/DuneServer/Binaries/Linux")
		c.Delivery.MQHome = ask(r, "HOME for rabbitmqctl (mq_home)", "/tmp")
	}
	if c.Delivery.Mode == "fls" || c.Delivery.Mode == "both" {
		c.Delivery.FLSToken = ask(r, "FLS token *", "")
		c.Delivery.PlayFabTitleID = ask(r, "PlayFab title ID", "")
	}

	fmt.Println("\n-- Discord bot --")
	c.Discord.Token = ask(r, "Bot token *", "")
	c.Discord.GuildID = ask(r, "Guild (server) ID *", "")
	c.Discord.AdminRoleID = ask(r, "Admin role ID (optional)", "")

	fmt.Println("\n-- Economy --")
	c.Economy.CurrencyName = ask(r, "Currency name", "Spice")
	c.Economy.Playtime.Enabled = askYesNo(r, "Reward playtime?", true)
	if c.Economy.Playtime.Enabled {
		c.Economy.Playtime.PerMinute = int64(askInt(r, "Currency per interval", 10))
		c.Economy.Playtime.AccrualInterval = ask(r, "Accrual interval", "60s")
		c.Economy.Playtime.OnlineQuery = defOnlineQ
	}
	c.Economy.Votes.Enabled = askYesNo(r, "Enable vote rewards (webhook)?", false)
	if c.Economy.Votes.Enabled {
		c.Economy.Votes.Reward = int64(askInt(r, "Reward per vote", 500))
		c.Economy.Votes.Secret = orRandom(ask(r, "Vote webhook secret (blank = random)", ""))
	}

	fmt.Println("\n-- Currency top-ups (manual PayPal donation flow) --")
	c.Economy.RealMoney.Enabled = askYesNo(r, "Enable currency top-ups?", true)
	if c.Economy.RealMoney.Enabled {
		c.Economy.RealMoney.Provider = "manual"
		c.Economy.RealMoney.WebhookSecret = orRandom("")
		c.Economy.RealMoney.PaymentLink = ask(r, "Payment link (e.g. paypal.me/you)", "")
		c.Economy.RealMoney.DonationNote = ask(r, "Donation note shown to players",
			"Donation — currency is a thank-you gift")
		c.Economy.RealMoney.Packages = []config.SpicePackage{
			{PriceLabel: "5 €", Amount: 3000},
			{PriceLabel: "10 €", Amount: 10000},
			{PriceLabel: "20 €", Amount: 30000},
		}
	}

	fmt.Println("\n-- Web dashboard --")
	c.Web.Enabled = askYesNo(r, "Enable admin dashboard?", true)
	if c.Web.Enabled {
		c.Web.ListenAddr = ask(r, "Dashboard listen address", "0.0.0.0:8091")
		c.Web.AdminUser = ask(r, "Dashboard username", "admin")
		c.Web.AdminPassword = orRandom(ask(r, "Dashboard password (blank = random)", ""))
		c.Web.SessionSecret = orRandom("")
	}

	c.Game.CharacterLookupQuery = defCharLookup

	data, err := yaml.Marshal(&c)
	if err != nil {
		fmt.Fprintln(os.Stderr, "setup: marshal:", err)
		os.Exit(1)
	}
	header := "# Generated by `dune-shop setup`. Contains secrets — keep file mode 600.\n" +
		"# Re-run `dune-shop setup -force` to regenerate.\n\n"
	if err := os.WriteFile(*out, append([]byte(header), data...), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "setup: write:", err)
		os.Exit(1)
	}
	fmt.Printf("\n✓ Wrote %s (mode 600)\n", *out)
	if c.Web.Enabled {
		fmt.Printf("  Dashboard login: %s / %s\n", c.Web.AdminUser, c.Web.AdminPassword)
	}
	fmt.Println("Next: load the starter catalog with `dune-shop seed`, then start the service.")
}

// --- prompt helpers ---

func ask(r *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func askInt(r *bufio.Reader, label string, def int) int {
	s := ask(r, label, strconv.Itoa(def))
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func askYesNo(r *bufio.Reader, label string, def bool) bool {
	d := "y/N"
	if def {
		d = "Y/n"
	}
	fmt.Printf("%s [%s]: ", label, d)
	line, _ := r.ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	if line == "" {
		return def
	}
	return line == "y" || line == "yes"
}

func orRandom(s string) string {
	if strings.TrimSpace(s) != "" {
		return s
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "change-me-" + strconv.Itoa(os.Getpid())
	}
	return hex.EncodeToString(b)
}
