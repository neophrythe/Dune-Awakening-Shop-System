package discord

import "github.com/bwmarrin/discordgo"

func commandDefs() []*discordgo.ApplicationCommand {
	minOne := float64(1)
	return []*discordgo.ApplicationCommand{
		{
			Name:        "link",
			Description: "Connect your Discord to your in-game character (run /howtolink if unsure)",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "character", Description: "Your exact in-game character name (case-sensitive)", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "account_id", Description: "Only if asked: your account/FLS id. Usually leave this blank.", Required: false},
			},
		},
		{Name: "howtolink", Description: "Step-by-step help for linking your account"},
		{Name: "balance", Description: "Show your shop balance"},
		{Name: "shop", Description: "Browse the shop"},
		{
			Name:        "buy",
			Description: "Buy an item from the shop",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "item_id", Description: "Item id (see /shop)", Required: true, MinValue: &minOne},
			},
		},
		{Name: "kits", Description: "Browse item packs/kits (bundles of items)"},
		{Name: "buyspice", Description: "Support the server & get Spice (donation)"},
		{
			Name:        "buykit",
			Description: "Buy a kit/pack — delivers all its items at once",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "kit_id", Description: "Kit id (see /kits)", Required: true, MinValue: &minOne},
			},
		},
		{
			Name:        "addkit",
			Description: "(admin) Create a new kit/pack",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "name", Description: "Kit display name", Required: true},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "price", Description: "Kit price", Required: true, MinValue: &minOne},
				{Type: discordgo.ApplicationCommandOptionString, Name: "category", Description: "Category (optional)"},
				{Type: discordgo.ApplicationCommandOptionString, Name: "description", Description: "Short description (optional)"},
			},
		},
		{
			Name:        "addkititem",
			Description: "(admin) Add an item to an existing kit",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "kit_id", Description: "Kit id (see /kits)", Required: true, MinValue: &minOne},
				{Type: discordgo.ApplicationCommandOptionString, Name: "game_item_id", Description: "In-game item id", Required: true},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "quantity", Description: "Amount (default 1)", MinValue: &minOne},
				{Type: discordgo.ApplicationCommandOptionString, Name: "name", Description: "Item display name (optional)"},
			},
		},
		{
			Name:        "grant",
			Description: "(admin) Grant currency to a user",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "Target user", Required: true},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Amount to grant", Required: true, MinValue: &minOne},
			},
		},
		{
			Name:        "additem",
			Description: "(admin) Add or update a shop item",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "game_item_id", Description: "In-game item id", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "name", Description: "Display name", Required: true},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "price", Description: "Price", Required: true, MinValue: &minOne},
				{Type: discordgo.ApplicationCommandOptionString, Name: "category", Description: "Category (optional)"},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "quantity", Description: "Amount delivered per purchase (default 1)", MinValue: &minOne},
			},
		},
	}
}

// registerCommands installs the slash commands. When a guild id is configured
// they are registered guild-scoped (instant); otherwise globally (slower to
// propagate).
func (b *Bot) registerCommands() error {
	appID := b.session.State.User.ID
	_, err := b.session.ApplicationCommandBulkOverwrite(appID, b.cfg.GuildID, commandDefs())
	return err
}
