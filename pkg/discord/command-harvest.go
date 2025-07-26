package discord

import (
	"context"
	"fmt"
	"strings"

	"cardano-valley/pkg/cv"
	"cardano-valley/pkg/logger"

	"github.com/bwmarrin/discordgo"
)

var WITHDRAW_COMMAND = discordgo.ApplicationCommand{
	Version:                  "0.01",
	Name:                     "harvest",
	Description:              "Withdraw your earned rewards from Cardano Valley.",
	// Options: []*discordgo.ApplicationCommandOption{{
	// 	Type:        discordgo.ApplicationCommandOptionString,
	// 	Name:        "address",
	// 	Description: "The address you want to link to your discord",
	// 	Required:    false,
	// 	MaxLength:   255,
	// }},
}

var WITHDRAW_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// to := i.ApplicationCommandData().Options[0].StringValue()
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	user := cv.LoadUser(i.Member.User.ID)
    if user.Wallet.PaymentKey == "" {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Error: You have not registered yet. Please /register first.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	// var withdrawalAddress string
	// if to != "" {
	// 	valid := blockfrost.VerifyAddress(ctx, to)
	// 	if !valid {
	// 		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
	// 			Content: "We had trouble verifying the address you entered. Please make sure it is a valid Cardano address.",
	// 			Flags:   discordgo.MessageFlagsEphemeral,
	// 		})
	// 		return
	// 	}

	// 	withdrawalAddress = to
	// } else {
	linkedWallets := cv.LoadUser(i.Member.User.ID).LinkedWallets
	if len(linkedWallets) == 0 {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "You need to provide an address to harvest your rewards. If you want to use your linked wallet, please use the `/harvest` command with the address option. Or you can link a new wallet using the `/link-wallet` command.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	logger.Record.Info("WITHDRAW_HANDLER called", "user", i.Member.User.ID, "linkedWallets", linkedWallets)

	options := []discordgo.SelectMenuOption{}
	for _, wallet := range linkedWallets {
		options = append(options, discordgo.SelectMenuOption{
			Label:       cv.TruncateMiddle(wallet.Payment, 32),
			Value:       cv.TruncateMiddle(wallet.Payment, 32),
			Description: "Harvest your rewards to this wallet",
			//addr1q8ur464mlqsqslh0dn9dqg88zn0q0sqag2hkxc0vhtrn5c7wkhumlr876ehcm8ltdwt7s49mwxfw47c4hcf5p6qdlavqaawfcs
		})
		logger.Record.Info("WITHDRAW_HANDLER options", "payment", cv.TruncateMiddle(wallet.Payment, 32))
	}

	min := 1
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title: "Harvesting...",
			Content: "Please wait while we calculate your withdrawal.",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    fmt.Sprintf("%s_%s", WITHDRAW_COMMAND_OPTIONLIST_NAME, i.Interaction.Member.User.ID),
							Placeholder: "Select an address",
							Options: options,
							MinValues:  &min,
							MaxValues:  1,
						},
					},
				},
			},
		},
	})
	// }
}

var WITHDRAW_COMMAND_OPTIONLIST_NAME = "harvest"
var WITHDRAW_COMMAND_OPTIONLIST_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate, selected discordgo.MessageComponentInteractionData) {
	logger.Record.Info("WITHDRAW_COMMAND_OPTIONLIST_HANDLER called", "user", i.Member.User.ID, "selected", selected)

	if len(selected.Values) == 0 {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "You need to select an address to harvest your rewards.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	pieces := strings.Split(selected.Values[0], "...")
	suffix := pieces[len(pieces)-1]
	user := cv.LoadUser(i.Member.User.ID)
	for _, v := range user.LinkedWallets {
		if strings.HasSuffix(v.Payment, suffix) {
			logger.Record.Info("WITHDRAW_COMMAND_OPTIONLIST_HANDLER found wallet", "wallet", v.Payment)
			// Call the harvest function with the selected wallet
			err := user.HarvestRewards(v.Payment)
			if err != nil {
				s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: fmt.Sprintf("Error harvesting rewards: %s", err.Error()),
					Flags:   discordgo.MessageFlagsEphemeral,
				})
				return
			}
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: fmt.Sprintf("Successfully harvested rewards to %s!", v.Payment),
				Flags:   discordgo.MessageFlagsEphemeral,
			})

			break
		}
		
	}
}