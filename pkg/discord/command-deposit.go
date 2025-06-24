package discord

import (
	"cardano-valley/pkg/cv"

	"github.com/bwmarrin/discordgo"
)

var DEPOSIT_COMMAND = discordgo.ApplicationCommand{
	Version:     "0.01",
	Name:        "deposit",
	Description: "Get deposit instructions for your farm wallet.",
}

var DEPOSIT_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	config := cv.LoadConfig(i.GuildID)
	if config.Wallet.Address == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No farm found. Please use /build-farm first.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	msg := "To deposit tokens into your farm wallet, send them to the following address:\n" + config.Wallet.Address
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: config.Wallet.Address,
		Flags:   discordgo.MessageFlagsEphemeral,
	})


	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "You can also use the following QR code to deposit tokens into your farm wallet:",
		Flags:   discordgo.MessageFlagsEphemeral,
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "Deposit Address",
				Description: "Use this address to deposit tokens into your farm wallet.",
			},
		},
	})
}
