// Package discord is the Discord front-end for the shop: it registers slash
// commands and routes them to the store and the shop service.
package discord

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"

	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/config"
	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/shop"
	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/store"
)

// Bot is the Discord bot.
type Bot struct {
	session    *discordgo.Session
	cfg        config.DiscordConfig
	store      *store.Store
	shop       *shop.Service
	currency   string
	charLookup string // game-DB query resolving character name -> account id
}

// New creates the bot. Call Start to connect and register commands. When
// charLookupQuery is non-empty, players can link with just their character name
// (the account id is resolved from the game database automatically).
func New(cfg config.DiscordConfig, st *store.Store, svc *shop.Service, currency, charLookupQuery string) (*Bot, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("discord: token required")
	}
	s, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("discord session: %w", err)
	}
	b := &Bot{session: s, cfg: cfg, store: st, shop: svc, currency: currency, charLookup: charLookupQuery}
	s.AddHandler(b.onInteraction)
	return b, nil
}

// nameOnlyLinking reports whether players can link with just a character name.
func (b *Bot) nameOnlyLinking() bool { return b.charLookup != "" }

// Start opens the gateway and registers slash commands.
func (b *Bot) Start() error {
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("discord open: %w", err)
	}
	if err := b.registerCommands(); err != nil {
		return fmt.Errorf("register commands: %w", err)
	}
	log.Printf("discord bot connected as %s", b.session.State.User.String())
	return nil
}

// Stop closes the gateway connection.
func (b *Bot) Stop() { _ = b.session.Close() }

func (b *Bot) onInteraction(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	name := i.ApplicationCommandData().Name
	h, ok := b.handlers()[name]
	if !ok {
		return
	}
	if err := h(context.Background(), i); err != nil {
		log.Printf("discord: command %s: %v", name, err)
		b.respondEphemeral(i, "⚠️ Something went wrong. Please try again later.")
	}
}

func (b *Bot) respond(i *discordgo.InteractionCreate, content string) {
	_ = b.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content},
	})
}

func (b *Bot) respondEphemeral(i *discordgo.InteractionCreate, content string) {
	_ = b.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral},
	})
}

func (b *Bot) respondEmbed(i *discordgo.InteractionCreate, e *discordgo.MessageEmbed) {
	_ = b.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{e}},
	})
}
