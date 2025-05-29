package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
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

var CONFIGURE_REWARD_COMMAND = discordgo.ApplicationCommand{
	Version:                  "0.01",
	Name:                     "configure-reward",
	Description:              "Create a reward.",
	DefaultMemberPermissions: &ADMIN,
}

var CONFIGURE_REWARD_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "configure-reward-modal_" + i.Interaction.Member.User.ID,
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
			},
		},
	})

}

var CONFIGURE_REWARD_MODAL_NAME = "configure-reward-modal"

var CONFIGURE_REWARD_MODAL_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ModalSubmitInteractionData) {
	// name := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	content := fmt.Sprintf("Reward `%#v` has been created successfully!", data.Components[0].(*discordgo.ActionsRow).Components[0])
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
}
