package discord

import (
	"bytes"
	"context"
	"text/template"
	"time"

	"cardano-valley/pkg/cardano"
	"cardano-valley/pkg/preeb"

	"github.com/bwmarrin/discordgo"
)

var INITIALIZE_COMMAND = discordgo.ApplicationCommand{
	Version:                  "0.01",
	Name:                     "initialize",
	Description:              "Initialize your server with a wallet and other configurations.",
	DefaultMemberPermissions: &ADMIN,
}

var INITIALIZE_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title: "Initialing...",
			Content: "Please wait while we set up your server with a wallet and other configurations.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	wallet, err := cardano.GenerateWallet(i.GuildID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Title: "Initialization Error",
				Content: "Error generating wallet: " + err.Error(),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	config := preeb.LoadConfig(i.GuildID)
	config.Wallet = *wallet
	config.Save()

	content := "Your wallet has been successfully initialized. Here is the address:\n"
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})

	var b bytes.Buffer
	sentence := "{{ .addr }}"
	partial := template.Must(template.New("configure-policy-id-template").Parse(sentence))
	partial.Execute(&b, map[string]interface{}{
		"addr": wallet.Address,
	})

	content = b.String()

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title: "Initialing Complete",
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
