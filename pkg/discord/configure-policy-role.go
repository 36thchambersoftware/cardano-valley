package discord

// import (
// 	"fmt"

// 	"cardano-valley/pkg/preeb"

// 	"github.com/bwmarrin/discordgo"
// )

// var CONFIGURE_POLICY_ROLE_COMMAND = discordgo.ApplicationCommand{
// 	Version:                  "0.01",
// 	Name:                     "configure-policy-role",
// 	Description:              "Set roles based on the configured pools",
// 	DefaultMemberPermissions: &ADMIN,
// 	Options: []*discordgo.ApplicationCommandOption{
// 		{
// 			Type:        discordgo.ApplicationCommandOptionString,
// 			Name:        "policyid",
// 			Description: "The a policy ID to link to your discord",
// 			Required:    true,
// 			MaxLength:   255,
// 		},
// 		{
// 			Type:        discordgo.ApplicationCommandOptionRole,
// 			Name:        "role",
// 			Description: "The role to associate with the attached policy range",
// 			Required:    true,
// 			MaxLength:   255,
// 		},
// 		{
// 			Type:        discordgo.ApplicationCommandOptionInteger,
// 			Name:        "min",
// 			Description: "The minimum quantity to qualify for the attached role",
// 			Required:    true,
// 			MaxLength:   255,
// 		},
// 		{
// 			Type:        discordgo.ApplicationCommandOptionInteger,
// 			Name:        "max",
// 			Description: "The max quantity to qualify for the attached role",
// 			Required:    false,
// 		},
// 	},
// }

// var CONFIGURE_POLICY_ROLE_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
// 	options := GetOptions(i)

// 	policyID := options["policyid"].StringValue()
// 	role := options["role"].RoleValue(s, i.GuildID)
// 	min := options["min"].IntValue()
// 	maybemax, ok := options["max"]
// 	var max int64 = 999_999_999_999
// 	if ok {
// 		max = maybemax.IntValue()
// 	}

// 	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
// 		Type: discordgo.InteractionResponseChannelMessageWithSource,
// 		Data: &discordgo.InteractionResponseData{
// 			Content: fmt.Sprintf("Adding policy role <@&%v>: Min %d - Max %d\n", role.ID, min, max),
// 			Flags:   discordgo.MessageFlagsEphemeral,
// 			Title:   "Configure Policy ID",
// 		},
// 	})

// 	config := preeb.LoadConfig(i.GuildID)

// 	policy, ok := config.PolicyIDs[policyID];
// 	if ok {
// 		if policy.Roles == nil {
// 			policy.Roles = make(preeb.PolicyRoles)
// 		}
// 	}

// 	order := int64(len(policy.Roles) + 1)

// 	bounds := preeb.Bounds{Min: preeb.Bound(min), Max: preeb.Bound(max), Order: order}

// 	policy.Roles[role.ID] = bounds

// 	if bounds.IsValid() {
// 		config.PolicyIDs[policyID] = policy
// 		config.Save()
// 		return
// 	}
// }
