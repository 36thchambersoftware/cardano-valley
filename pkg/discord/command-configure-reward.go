package discord

import (
	"cardano-valley/pkg/blockfrost"
	"cardano-valley/pkg/cv"
	"cardano-valley/pkg/logger"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

//   {
//       "name": "PUNKS",
//       "description": "Stay Punked!",
//       "icon": "https://punks.staking.zip/_next/image?url=https%3A%2F%2Ffirebasestorage.googleapis.com%2Fv0%2Fb%2Fstakingdotzip.appspot.com%2Fo%2Fpunks%252FAdaPunks_LOGO_Transparent%2520-%2520Jonas%2520H%25C3%25BCrbin.png%3Falt%3Dmedia%26token%3D2ea01a97-9394-42f8-ad97-882c07331975&w=1920&q=75",
//       "assetType": "token",
//       "rewardToken": "e633efbf19a37500c6f22965af3130baa34c3a644a146662dd2d74a2.50554e4b53",
//       "amountPerUser": 100,
//       "rolesEligible": [
//         "1273259456389714054"
//       ],
//   }
var (
	CONFIGURE_REWARD_COMMAND = discordgo.ApplicationCommand{
		Version:                  "0.01",
		Name:                     "configure-reward",
		Description:              "Create a reward.",
		DefaultMemberPermissions: &ADMIN,
	}

	CONFIGURE_REWARD_NAME_MODAL_NAME = "configure-reward-name-modal"
	CONFIGURE_REWARD_ASSET_COMPONENT_NAME = "configure-reward-asset-component"

	availableAssets map[string][]discordgo.SelectMenuOption
)

var CONFIGURE_REWARD_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("%s_%s", CONFIGURE_REWARD_NAME_MODAL_NAME, i.Interaction.Member.User.ID),
			Title: "Create Reward",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "name",
							Label:       "Name of the Reward",
							Style:       discordgo.TextInputShort,
							Placeholder: "i.e. PUNKS or SOCKZ",
							Required:    true,
							MaxLength:   25,
							MinLength:   3,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "description",
							Label:       "Description of the Reward",
							Style:       discordgo.TextInputParagraph,
							Placeholder: "i.e. Stay Punked!",
							Required:    true,
							MaxLength:   255,
							MinLength:   3,
						},
					},
				},
			},
		},
	})


	config := cv.LoadConfig(i.GuildID)
	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
	defer cancel()

	guildAddress, err := blockfrost.GetAddress(ctx, config.Wallet.Address)
	if err != nil {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Error retrieving guild address. Please try again later.",
		})
		return
	}


	availableAssets = make(map[string][]discordgo.SelectMenuOption)
	values := []discordgo.SelectMenuOption{}
	p := message.NewPrinter(language.English)
	for _, asset := range guildAddress.Amount {
		qty, err := strconv.Atoi(asset.Quantity)
		if err != nil {
			logger.Record.Error("Error converting asset quantity to integer: %v", err)
			continue
		}

		if asset.Quantity != "" && qty > 1 && asset.Unit != "" && asset.Unit != "lovelace" {
			assetInfo, err := blockfrost.AssetInfo(ctx, asset.Unit)
			if err != nil {
				logger.Record.Error("Error retrieving asset info: %v", err)
				continue
			}

			if assetInfo.Metadata != nil {
				values = append(values, discordgo.SelectMenuOption{
					Label:       p.Sprintf("%s (qty: %d)", assetInfo.Metadata.Name, qty),
					Value:       assetInfo.Asset,
					Description: assetInfo.Metadata.Description,
				})
			}
		} else if asset.Quantity != "" && asset.Unit == "lovelace" {
			ada := qty / blockfrost.LOVELACE
			values = append(values, discordgo.SelectMenuOption{
				Label:       p.Sprintf("ADA (qty: %d)", ada),
				Value:       "lovelace",
				Description: "Cardano native token (ADA)",
			})
		}
	}
	availableAssets[i.Interaction.Member.User.ID] = values
}

var CONFIGURE_REWARD_NAME_MODAL_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ModalSubmitInteractionData) {
	// name := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	logger.Record.Info("Reward creation initiated", "assets", availableAssets)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
			CustomID: fmt.Sprintf("%s_%s", CONFIGURE_REWARD_ASSET_COMPONENT_NAME, i.Interaction.Member.User.ID),
			Title: "Create Reward",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    fmt.Sprintf("%s_%s", CONFIGURE_REWARD_ASSET_COMPONENT_NAME, i.Interaction.Member.User.ID),
							Placeholder: "Select Asset",
							Options: availableAssets[i.Interaction.Member.User.ID],
						},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Record.Error("Could not create follow up modal", "ERROR", err)
		return
	}

	// content := fmt.Sprintf("Reward `%#v` has been created successfully!", data.Components[0].(*discordgo.ActionsRow).Components[0])
	// _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
	// 	Content: &content,
	// })
	// if err != nil {
	// 	logger.Record.Error("Could not respond to modal submit", "ERROR", err)
	// 	return
	// }
}

var CONFIGURE_REWARD_ASSET_COMPONENT_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.MessageComponentInteractionData) {
	selection := ExtractArgsFromSelect(data)
	logger.Record.Info("Asset selected", "ASSET", selection)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Reward configuration is not yet implemented.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

}