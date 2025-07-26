package discord

import (
	"cardano-valley/pkg/cv"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var DASHBOARD_COMMAND = discordgo.ApplicationCommand{
	Version:     "0.01",
	Name:        "dashboard",
	Description: "View your staking dashboard and wallets.",
}

var DASHBOARD_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.GuildID == "" {
		// For now, we only support guild-based commands
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This command can only be used in a server for now.",
				Flags:  discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// config := cv.LoadConfig(i.GuildID)
	// if config.Name == "" {
	// 	// If the user is not in any of the guilds, send an error message
	// 	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
	// 		Type: discordgo.InteractionResponseChannelMessageWithSource,
	// 		Data: &discordgo.InteractionResponseData{
	// 			Content: "Cardano Valley isn't setup on this server.",
	// 			Flags:  discordgo.MessageFlagsEphemeral,
	// 		},
	// 	})
	// 	return
	// }

	// TODO: Get the guild IDs associated with Cardano Valley
	// TODO: Cross-reference the guild IDs to find the ones associated with Cardano Valley
	
	// If the user is in a guild, fetch the user's data from the database
	user := cv.LoadUser(i.Member.User.ID)
	if user.ID == "" {
		// If the user is not found, send an error message
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You are not registered with Cardano Valley yet. Run `/register` to get started.",
				Flags:  discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Calculate the user's total yield, staked amount, and leaderboard rank
	var fields []*discordgo.MessageEmbedField

	for _, reward := range user.Rewards {
		for asset, balance := range reward {
			tokenBits := strings.Split(string(asset), ".")

			var name []byte
			if len(tokenBits) < 2 {
				name = []byte(fmt.Sprintf("Unknown Token: %s", asset))
			} else {
				// Decode the token name from hex
				name = []byte(tokenBits[1]) // Use the second part as the name
			}

			name, err := hex.DecodeString(string(name))
			if err != nil {
				name = []byte(fmt.Sprintf("Unknown Token: %s", asset))
			}

			value := fmt.Sprintf("%d", balance.Earned)
			fields = append(fields, &discordgo.MessageEmbedField{Name: string(name), Value: value, Inline: true})
		}
	}

	var wallets []string
	for _, wallet := range user.LinkedWallets {
		wallets = append(wallets, fmt.Sprintf("1. %s", cv.TruncateMiddle(wallet.Payment, 32)))
	}
	walletList := strings.Join(wallets, "\n")
	fields = append(fields, &discordgo.MessageEmbedField{Name: fmt.Sprintf("Linked Wallets (%d)", len(wallets)), Value: walletList, Inline: false})

	embed := &discordgo.MessageEmbed{
		Title:       "🌾 Cardano Valley Dashboard",
		Description: fmt.Sprintf("Your farm overview\nCardano Valley Deposit Address: %s", user.Wallet.Address),
		Color:       0x00ff99,
		Fields: 	 fields,
		Thumbnail:   &discordgo.MessageEmbedThumbnail{
			URL: cv.IconImage, // Replace with your icon
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Your farm is growing!",
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}
