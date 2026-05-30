package discord

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/shop"
	"github.com/neophrythe/Dune-Awakening-Shop-System/internal/store"
)

type handlerFunc func(ctx context.Context, i *discordgo.InteractionCreate) error

func (b *Bot) handlers() map[string]handlerFunc {
	return map[string]handlerFunc{
		"link":    b.handleLink,
		"balance": b.handleBalance,
		"shop":    b.handleShop,
		"buy":     b.handleBuy,
		"grant":   b.adminOnly(b.handleGrant),
		"additem": b.adminOnly(b.handleAddItem),
	}
}

func optMap(i *discordgo.InteractionCreate) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	m := map[string]*discordgo.ApplicationCommandInteractionDataOption{}
	for _, o := range i.ApplicationCommandData().Options {
		m[o.Name] = o
	}
	return m
}

func callerID(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

func (b *Bot) adminOnly(h handlerFunc) handlerFunc {
	return func(ctx context.Context, i *discordgo.InteractionCreate) error {
		if b.cfg.AdminRoleID == "" || i.Member == nil {
			b.respondEphemeral(i, "⛔ Admin commands are not configured.")
			return nil
		}
		for _, r := range i.Member.Roles {
			if r == b.cfg.AdminRoleID {
				return h(ctx, i)
			}
		}
		b.respondEphemeral(i, "⛔ You don't have permission to use this command.")
		return nil
	}
}

func (b *Bot) handleLink(ctx context.Context, i *discordgo.InteractionCreate) error {
	o := optMap(i)
	la, err := b.store.LinkAccount(ctx, callerID(i),
		o["account_id"].StringValue(), o["character"].StringValue())
	if err != nil {
		return err
	}
	b.respondEphemeral(i, fmt.Sprintf("✅ Linked to **%s** (account `%s`).", la.CharacterName, la.GameAccountID))
	return nil
}

func (b *Bot) handleBalance(ctx context.Context, i *discordgo.InteractionCreate) error {
	la, err := b.store.LinkByDiscord(ctx, callerID(i))
	if errors.Is(err, store.ErrNotFound) {
		b.respondEphemeral(i, "You haven't linked your account yet — use `/link`.")
		return nil
	}
	if err != nil {
		return err
	}
	bal, err := b.store.Balance(ctx, la.ID)
	if err != nil {
		return err
	}
	b.respondEphemeral(i, fmt.Sprintf("💰 You have **%d %s**.", bal, b.currency))
	return nil
}

func (b *Bot) handleShop(ctx context.Context, i *discordgo.InteractionCreate) error {
	items, err := b.store.ListItems(ctx, true)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		b.respondEphemeral(i, "The shop is empty right now.")
		return nil
	}
	var sb strings.Builder
	cat := "\x00"
	for _, it := range items {
		if it.Category != cat {
			cat = it.Category
			label := cat
			if label == "" {
				label = "General"
			}
			fmt.Fprintf(&sb, "\n__**%s**__\n", label)
		}
		stock := ""
		if it.Stock != nil {
			stock = fmt.Sprintf(" · stock %d", *it.Stock)
		}
		fmt.Fprintf(&sb, "`#%d` **%s** — %d %s%s\n", it.ID, it.Name, it.Price, b.currency, stock)
	}
	b.respondEmbed(i, &discordgo.MessageEmbed{
		Title:       "🛒 Dune Awakening Shop",
		Description: sb.String(),
		Color:       0xC97B3C,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Buy with /buy item_id:<#>"},
	})
	return nil
}

func (b *Bot) handleBuy(ctx context.Context, i *discordgo.InteractionCreate) error {
	itemID := optMap(i)["item_id"].IntValue()
	res, err := b.shop.Buy(ctx, callerID(i), itemID)
	switch {
	case errors.Is(err, shop.ErrNotLinked):
		b.respondEphemeral(i, "Link your account first with `/link`.")
		return nil
	case errors.Is(err, store.ErrInsufficientFunds):
		b.respondEphemeral(i, "❌ You don't have enough balance for that item.")
		return nil
	case errors.Is(err, store.ErrOutOfStock):
		b.respondEphemeral(i, "❌ That item is out of stock.")
		return nil
	case errors.Is(err, store.ErrItemUnavailable) || errors.Is(err, store.ErrNotFound):
		b.respondEphemeral(i, "❌ That item isn't available.")
		return nil
	case errors.Is(err, shop.ErrDeliveryFailed):
		b.respondEphemeral(i, "⚠️ Couldn't deliver in-game — you were refunded. Make sure you're online and try again.")
		return nil
	case err != nil:
		return err
	}
	b.respond(i, fmt.Sprintf("✅ Bought **%s**! New balance: **%d %s**.",
		res.Item.Name, res.NewBalance, b.currency))
	return nil
}

func (b *Bot) handleGrant(ctx context.Context, i *discordgo.InteractionCreate) error {
	o := optMap(i)
	target := o["user"].UserValue(nil)
	amount := o["amount"].IntValue()
	la, err := b.store.LinkByDiscord(ctx, target.ID)
	if errors.Is(err, store.ErrNotFound) {
		b.respondEphemeral(i, "That user hasn't linked an account yet.")
		return nil
	}
	if err != nil {
		return err
	}
	bal, err := b.store.Credit(ctx, la.ID, amount, store.TxnAdjust, "admin grant")
	if err != nil {
		return err
	}
	b.respondEphemeral(i, fmt.Sprintf("✅ Granted **%d %s** to <@%s>. New balance: %d.",
		amount, b.currency, target.ID, bal))
	return nil
}

func (b *Bot) handleAddItem(ctx context.Context, i *discordgo.InteractionCreate) error {
	o := optMap(i)
	qty := 1
	if v, ok := o["quantity"]; ok {
		if n := int(v.IntValue()); n >= 1 {
			qty = n
		}
	}
	cat := ""
	if v, ok := o["category"]; ok {
		cat = v.StringValue()
	}
	it := &store.CatalogItem{
		GameItemID: o["game_item_id"].StringValue(),
		Name:       o["name"].StringValue(),
		Category:   cat,
		Price:      o["price"].IntValue(),
		Quantity:   qty,
		Enabled:    true,
	}
	id, err := b.store.UpsertItem(ctx, it)
	if err != nil {
		return err
	}
	b.respondEphemeral(i, fmt.Sprintf("✅ Added item `#%d` **%s** for %d %s.", id, it.Name, it.Price, b.currency))
	return nil
}
