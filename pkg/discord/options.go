package discord

import "github.com/bwmarrin/discordgo"

type Options map[string]*discordgo.ApplicationCommandInteractionDataOption

func GetOptions(i *discordgo.InteractionCreate) Options {
	options := i.ApplicationCommandData().Options

	// Or convert the slice into a map
	optionMap := make(Options, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	return optionMap
}
