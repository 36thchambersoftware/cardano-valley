package discord

import (
	"github.com/bwmarrin/discordgo"
)

func FindRoleByName(i *discordgo.InteractionCreate, name string) (*discordgo.Role, error) {
	var desiredRole *discordgo.Role

	perms, err := S.GuildRoles(i.GuildID)
	if err != nil {
		return nil, err
	}

	for _, role := range perms {
		if role.Name == name {
			desiredRole = role
		}
	}

	return desiredRole, nil
}

func FindRoleByRoleID(guildID, id string) (*discordgo.Role, error) {
	var desiredRole *discordgo.Role

	perms, err := S.GuildRoles(guildID)
	if err != nil {
		return nil, err
	}

	for _, role := range perms {
		if role.ID == id {
			desiredRole = role
		}
	}

	return desiredRole, nil
}

func UserHasRole(memberRoles []string, role discordgo.Role) bool {
	var user_has_role bool
	for _, r := range memberRoles {
		if r == role.ID {
			user_has_role = true
			break
		}
	}

	return user_has_role
}

func ToggleRole(i *discordgo.InteractionCreate, role *discordgo.Role) error {
	var response string
	user_has_role := UserHasRole(i.Member.Roles, *role)
	if !user_has_role {
		err := S.GuildMemberRoleAdd(i.GuildID, i.Member.User.ID, role.ID)
		if err != nil {
			return err
		}
		response = "Role added: <@&" + role.ID + ">"
	} else {
		err := S.GuildMemberRoleRemove(i.GuildID, i.Member.User.ID, role.ID)
		if err != nil {
			return err
		}
		response = "Role removed: <@&" + role.ID + ">"
	}

	S.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: response,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	return nil
}

func AssignRoleByRoleName(i *discordgo.InteractionCreate, roleName string) (*discordgo.Role, error) {
	role, err := FindRoleByName(i, roleName)
	if err != nil {
		return nil, err
	}

	err = S.GuildMemberRoleAdd(i.GuildID, i.Member.User.ID, role.ID)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func AssignRoleByID(guildID, userID, roleID string) (*discordgo.Role, error) {
	role, err := FindRoleByRoleID(guildID, roleID)
	if err != nil {
		return nil, err
	}

	err = S.GuildMemberRoleAdd(guildID, userID, role.ID)
	if err != nil {
		return nil, err
	}
	return role, nil
}



// func FormatNewRolesMessage(user preeb.User, roleIDs []string) (string) {
// 	walletWord := "wallet"
// 	if len(user.Wallets) > 1 {
// 		walletWord = "wallets"
// 	}

// 	walletList := ""
// 	n := 0
// 	for _, addr := range user.Wallets {
// 		n++
// 		walletList = walletList + strconv.Itoa(n) + ". -# " + string(addr) + "\n"
// 	}

// 	var b bytes.Buffer
// 	sentence := "After looking at your {{ .walletCount }} {{ .walletWord }}\n{{ .walletList }}"

// 	if roleIDs != nil {
// 		sentence = sentence + "You have been assigned the following!\n"
// 		for _, roleID := range roleIDs {
// 			sentence = sentence + "<@&" + roleID + ">\n"
// 		}
// 	} else {
// 		sentence = sentence + "You don't qualify for any roles."
// 	}

// 	partial := template.Must(template.New("check-delegation-template").Parse(sentence))
// 	partial.Execute(&b, map[string]interface{}{
// 		"walletCount": len(user.Wallets),
// 		"walletWord":  walletWord,
// 		"walletList":  walletList,
// 	})

// 	content := b.String()
// 	return content
// }