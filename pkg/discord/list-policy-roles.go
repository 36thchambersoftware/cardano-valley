package discord

// import (
// 	"cardano-valley/pkg/preeb"
// 	"sort"

// 	"github.com/bwmarrin/discordgo"
// 	"golang.org/x/text/language"
// 	"golang.org/x/text/message"
// )

// var LIST_POLICY_ROLES_COMMAND = discordgo.ApplicationCommand{
// 	Version:                  "0.01",
// 	Name:                     "list-policy-roles",
// 	Description:              "See the configured policy roles",
// 	DefaultMemberPermissions: &ADMIN,
// }

// var LIST_POLICY_ROLES_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
// 	config := preeb.LoadConfig(i.GuildID)
// 	p := message.NewPrinter(language.English)

// 	var policy_roles []struct{
// 		roleID string
// 		min int64
// 		max int64
// 		order int64
// 	}

// 	sentence := "## Policy Roles\n"
// 	for policyID, policy := range config.PolicyIDs {
// 		for role, bounds := range policy.Roles {
// 			policy_roles = append(policy_roles, struct {
// 				roleID string
// 				min    int64
// 				max    int64
// 				order  int64
// 			}{
// 				roleID: role,
// 				min:    int64(bounds.Min),
// 				max:    int64(bounds.Max),
// 				order:  bounds.Order,
// 			})
// 		}

// 		sort.Slice(policy_roles, func(i, j int) bool {
// 			return (policy_roles[i].order < policy_roles[j].order)
// 		})

// 		sentence = sentence + p.Sprintf("**%s**\n", policyID)
// 		for _, setting := range policy_roles {
// 			sentence = sentence + p.Sprintf(" <@&%s>\t %v - %v \n", setting.roleID, setting.min, setting.max)
// 		}
// 	}

// 	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
// 		Type: discordgo.InteractionResponseChannelMessageWithSource,
// 		Data: &discordgo.InteractionResponseData{
// 			Content: sentence,
// 			Title:   "List Policy Roles",
// 		},
// 	})
// }
