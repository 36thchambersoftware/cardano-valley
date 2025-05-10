package discord

import (
	"cardano-valley/pkg/cv"

	"github.com/bwmarrin/discordgo"
)

var DASHBOARD_COMMAND = discordgo.ApplicationCommand{
	Name:        "dashboard",
	Description: "View your staking dashboard.",
}

var DASHBOARD_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "🌾 Cardano Valley Dashboard",
		Description: "Your farm overview",
		Color:       0x00ff99,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Total Yield", Value: "1,234 $MEME", Inline: true},
			{Name: "Staked Amount", Value: "5,000 $MEME", Inline: true},
			{Name: "Leaderboard Rank", Value: "#7", Inline: true},
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: cv.IconImage, // Replace with your icon
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Updated just now",
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
