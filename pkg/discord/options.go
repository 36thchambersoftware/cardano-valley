package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type (
	Options map[string]*discordgo.ApplicationCommandInteractionDataOption

	Args struct {
		Name	  string
		Value     string
	}
)

func GetOptions(i *discordgo.InteractionCreate) Options {
	options := i.ApplicationCommandData().Options

	// Or convert the slice into a map
	optionMap := make(Options, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	return optionMap
}

func ExtractArgsFromSlash(data discordgo.ApplicationCommandInteractionData) []Args {
	args := []Args{}
	for _, opt := range data.Options {
		args = append(args, Args{
			Name:  opt.Name,
			Value: fmt.Sprintf("%v", opt.Value),
		})
	}
	return args
}

func ExtractArgsFromModal(data discordgo.ModalSubmitInteractionData) []Args {
	args := []Args{}
	for _, row := range data.Components {
		actionRow, ok := row.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, comp := range actionRow.Components {
			if input, ok := comp.(*discordgo.TextInput); ok {
				args = append(args, Args{
					Name:  input.CustomID,
					Value: input.Value,
				})
			}
		}
	}
	return args
}

func ExtractArgsFromSelect(data discordgo.MessageComponentInteractionData) []Args {
	return []Args{
		{
			Name:  data.CustomID,
			Value: strings.Join(data.Values, ","),
		},
	}
}
