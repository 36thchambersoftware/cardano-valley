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

var REGISTER_COMMAND = discordgo.ApplicationCommand{
	Version:                  "0.01",
	Name:                     "register",
	Description:              "Register with this server to participate in Cardano Valley.",
	DefaultMemberPermissions: &ADMIN,
}

var REGISTER_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title: "Registering",
			Content: "Please wait while we set up your wallet and other configurations.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	// Initialize the user wallet
	userWallet, err := cardano.GenerateWallet(i.Member.User.ID)
	if errors.Is(err, cardano.ErrWalletExists) {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Great news! You have already been registered. Here is your wallet address:\n" + userWallet.Address,
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

	user := cv.LoadUser(i.Member.User.ID)
	user.Wallet = *userWallet
	user.Save()

	logger.Record.Info("WALLET", "USER: ", user)

	content := "Your wallet has been successfully created. Here is the address:\n"
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})

	var b bytes.Buffer
	sentence := "{{ .addr }}"
	partial := template.Must(template.New("configure-policy-id-template").Parse(sentence))
	partial.Execute(&b, map[string]interface{}{
		"addr": userWallet.Address,
	})

	content = b.String()

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: content,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
}
