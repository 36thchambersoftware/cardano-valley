package discord

import (
	"github.com/bwmarrin/discordgo"
)

var LIST_DELEGATOR_ASSETS_COMMAND = discordgo.ApplicationCommand{
	Version:                  "0.01",
	Name:                     "list-delegator-assets",
	Description:              "Find all configured policy ids in linked wallets",
	DefaultMemberPermissions: &ADMIN,
	
}


var LIST_DELEGATOR_ASSETS_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Finding assets...",
			Flags:   discordgo.MessageFlagsEphemeral,
			Title:   "List Assets",
		},
	})


}