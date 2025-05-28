package discord

import (
	"github.com/bwmarrin/discordgo"
)

var HELP_COMMAND = discordgo.ApplicationCommand{
	Version:     "0.01",
	Name:        "help",
	Description: "Display helpful information.",
}

var HELP_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	message := `
# Server Setup
Open a ticket to get started with installing Cardano Valley in your server. 
1. Run ` + "`/build-farm`" + ` to initialize your server. This will give you an empty config and a wallet to get started. 
1. Next run ` + "`/deposit`" + ` to get your server's address for rewards. You can then simply send tokens to it like any other address.
1. After configuration of your server and rewards is complete, you can run ` + "`/list-server-rewards`" + ` to display the rewards available to your holders.

# User Setup
If you are interested in earning rewards through Cardano Valley and a server you are in is participating, then you just need to run ` + "`/register`" + ` once in any server using Cardano Valley. This will create your user with the bot in order to start accumulating reward

After you're setup, you can run ` + "`/dashboard`" + ` to see any rewards you've accumulated.
`
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	})
}
