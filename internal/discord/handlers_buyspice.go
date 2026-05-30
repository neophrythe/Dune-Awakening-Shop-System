package discord

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// handleBuySpice shows the manual (admin-confirmed) Spice donation flow: the
// configured donation packages, the payment link, and how a player receives
// their Spice after donating. No real-money is processed by the bot — an admin
// confirms the payment and runs /grant.
func (b *Bot) handleBuySpice(_ context.Context, i *discordgo.InteractionCreate) error {
	rm := b.realMoney

	var sb strings.Builder
	if len(rm.Packages) == 0 {
		fmt.Fprintf(&sb, "Spice donation packages are not configured yet — please check back soon.")
	} else {
		fmt.Fprintf(&sb, "Support **BuG Island** and receive **%s** as a thank-you. / "+
			"Unterstütze **BuG Island** und erhalte **%s** als Dankeschön.\n\n", b.currency, b.currency)
		for _, p := range rm.Packages {
			fmt.Fprintf(&sb, "• **%s** → %d %s\n", p.PriceLabel, p.Amount, b.currency)
		}
	}

	link := rm.PaymentLink
	if link == "" {
		link = "(payment link not set yet / Zahlungslink folgt)"
	}

	fmt.Fprintf(&sb, "\n**How to / So geht's:**\n"+
		"🇬🇧 1. Send your donation here: %s\n"+
		"   2. Add your **Discord name + character** in the note.\n"+
		"   3. Post a screenshot in this channel — an admin credits your %s.\n\n"+
		"🇩🇪 1. Sende deine Spende hier: %s\n"+
		"   2. Schreibe deinen **Discord-Namen + Charakter** in die Notiz.\n"+
		"   3. Poste einen Screenshot hier — ein Admin schreibt dir %s gut.",
		link, b.currency, link, b.currency)

	if rm.DonationNote != "" {
		fmt.Fprintf(&sb, "\n\n%s", rm.DonationNote)
	}

	b.respondEmbed(i, &discordgo.MessageEmbed{
		Title:       "🌶️ Get Spice / Spice erhalten",
		Description: sb.String(),
		Color:       0xC97B3C,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Donations support the server. Spice is a thank-you, not a purchase of in-game value.",
		},
	})
	return nil
}
