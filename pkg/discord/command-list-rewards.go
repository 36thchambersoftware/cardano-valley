package discord

import (
	"cardano-valley/pkg/cv"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var LIST_SERVER_REWARDS_COMMAND = discordgo.ApplicationCommand{
	Version:     "0.01",
	Name:        "list-server-rewards",
	Description: "View all available staking rewards in this server",
	Type:        discordgo.ChatApplicationCommand,
}

var LIST_SERVER_REWARDS_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	config := cv.LoadConfig(i.GuildID)
	if config.Rewards == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Cardano Valley isn't setup on this server.",
				Flags:  discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	guild, err := s.State.Guild(i.GuildID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Error fetching guild information.",
				Flags:  discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	var embeds []*discordgo.MessageEmbed

	for _, reward := range config.Rewards {
		var fields []*discordgo.MessageEmbedField
		roles := strings.Join(reward.RolesEligible, ", ")
		for _, r := range reward.RolesEligible {
			roles = strings.ReplaceAll(roles, r, fmt.Sprintf("<@&%s>", r))
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name: reward.Name,
			Value: fmt.Sprintf(
				"**Type:** %s\n**Amount:** %d\n**Frequency:** Daily\n**Roles Eligible:** %s",
				reward.AssetType,
				reward.RoleAmount,
				roles,
			),
			Inline: false,
		})

		embeds = append(embeds, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("🌾 %s Rewards", guild.Name),
			Description: reward.Description,
			Fields:      fields,
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("Requested by %s", i.Member.User.Username),
			},
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: reward.Icon,
			},
			Color:       0x00cc99,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: embeds,
		},
	})
}
