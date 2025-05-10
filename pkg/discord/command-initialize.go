package discord

import (
	"bytes"
	"errors"
	"text/template"

	"cardano-valley/pkg/cardano"
	"cardano-valley/pkg/cv"
	"cardano-valley/pkg/logger"

	"github.com/bwmarrin/discordgo"
)

var INITIALIZE_COMMAND = discordgo.ApplicationCommand{
	Version:                  "0.01",
	Name:                     "build-farm",
	Description:              "Start your own farm.",
	DefaultMemberPermissions: &ADMIN,
}

var INITIALIZE_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title: "Initialing...",
			Content: "Please wait while we set up your new farm!",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	// Initialize the guild wallet
	guildWallet, err := cardano.GenerateWallet(i.GuildID)
	if errors.Is(err, cardano.ErrWalletExists) {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "This server has already started a farm. Here is your farm's wallet address:\n" + guildWallet.Address,
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

	config := cv.LoadConfig(i.GuildID)
	config.Wallet = *guildWallet
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
		"addr": guildWallet.Address,
	})

	content = b.String()

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: content,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
}
