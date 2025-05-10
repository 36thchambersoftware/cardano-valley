package discord

import (
	"bytes"
	"context"
	"text/template"

	"cardano-valley/pkg/blockfrost"
	"cardano-valley/pkg/cv"

	"github.com/bwmarrin/discordgo"
)

var WITHDRAW_COMMAND = discordgo.ApplicationCommand{
	Version:                  "0.01",
	Name:                     "harvest",
	Description:              "Withdraw your earned rewards from Cardano Valley.",
	Options: []*discordgo.ApplicationCommandOption{{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        "address",
		Description: "The address you want to link to your discord",
		Required:    true,
		MaxLength:   255,
	}},
}

var WITHDRAW_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	
	to := i.ApplicationCommandData().Options[0].StringValue()
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	valid := blockfrost.VerifyAddress(ctx, to)
	if !valid {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "We had trouble verifying the address you entered. Please make sure it is a valid Cardano address.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title: "Harvesting...",
			Content: "Please wait while we calculate your withdrawal.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	user := cv.LoadUser(i.Member.User.ID)
    if user.Wallet.PaymentKey == "" {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Error: You have not registered yet. Please register first.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}



	content := "Your wallet has been successfully created. Here is the address:\n"
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})

	var b bytes.Buffer
	sentence := "{{ .addr }}"
	partial := template.Must(template.New("configure-policy-id-template").Parse(sentence))
	partial.Execute(&b, map[string]interface{}{
		"addr": user.Wallet.Address,
	})

	content = b.String()

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: content,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
}
