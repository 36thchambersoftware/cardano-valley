package discord

import (
	"bytes"
	"errors"
	"text/template"

	"cardano-valley/pkg/cardano"
	"cardano-valley/pkg/logger"
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
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title: "Initialing...",
			Content: "Please wait while we set up your server with a wallet and other configurations.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	wallet, err := cardano.GenerateWallet(i.GuildID)
	if errors.Is(err, cardano.WALLET_EXISTS_ERROR) {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "This server has already been initialized",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	} else if err != nil {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Error generating wallet: " + err.Error(),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	config := preeb.LoadConfig(i.GuildID)
	config.Wallet = *wallet
	config.Save()

	logger.Record.Info("WALLET", "CONFIG:", config)

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

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: content,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
}
