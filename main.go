package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	mongo "cardano-valley/pkg/db"
	"cardano-valley/pkg/discord"
	"cardano-valley/pkg/logger"

	mongodb "go.mongodb.org/mongo-driver/mongo"

	"github.com/bwmarrin/discordgo"
)

// DB Variables
var (
	mdb *mongodb.Client
	dbctx context.Context
	dbcancel context.CancelFunc
	CommandHistory *mongodb.Collection
)

// Discord Variables
var (
	integerOptionMinValue          = 1.0
	dmPermission                   = false
	defaultMemberPermissions int64 = discordgo.PermissionManageServer

	commands = []*discordgo.ApplicationCommand{
		&discord.INITIALIZE_COMMAND,
		&discord.REGISTER_COMMAND,
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		discord.INITIALIZE_COMMAND.Name:      		discord.INITIALIZE_HANDLER,
		discord.REGISTER_COMMAND.Name:      		discord.REGISTER_HANDLER,
	}
	lockout         = make(map[string]struct{})
	lockoutResponse = &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Please wait for your last command to finish. :D",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}
)

type (
	Command struct {
		Name        string
		Timestamp   time.Time
		UserID      string
		GuildID     string
		ChannelID   string
		Arguments   []*discordgo.ApplicationCommandInteractionDataOption `json:"options"`
	}
)

func init() {
	// Setup DB
    mdb, ctx, cancel, err := mongo.Connect()
    if err != nil {
        panic(err)
    }

	dbctx = ctx
	dbcancel = cancel
	mongo.DB = mdb

	CommandHistory = mongo.DB.Database("cardano-valley").Collection("command-history")
}

func main() {
	defer mongo.Close(mongo.DB, dbctx, dbcancel)
	l := logger.Record

	// Setup discord
	discord.S.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if strings.Contains(strings.ToUpper(m.Author.GlobalName), "ANNOUNCEMENTS") || strings.Contains(strings.ToUpper(m.Author.GlobalName), "ADMIN") {
			s.ChannelMessageDelete(m.ChannelID, m.ID)
		}

		if m.Author.Bot {
			return
		}
	})

	// Setup Command Handler
	discord.S.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			if _, ok := lockout[i.Member.User.ID]; !ok {
				lockout[i.Member.User.ID] = struct{}{}
				defer func() {
					delete(lockout, i.Member.User.ID)
				}()

				if _, err := CommandHistory.InsertOne(dbctx, Command{
					Name:      i.ApplicationCommandData().Name,
					Timestamp: time.Now(),
					UserID:    i.Member.User.ID,
					GuildID:   i.GuildID,
					ChannelID: i.ChannelID,
					Arguments: i.ApplicationCommandData().Options,
				}); err != nil {
					logger.Record.Error("Could not log command", "ERROR", err)
				}
				h(s, i)
			} else {
				s.InteractionRespond(i.Interaction, lockoutResponse)
			}
		}
	})

	discord.S.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		logger.Record.Info("LOGGED IN", "USER", fmt.Sprintf("%v#%v", s.State.User.Username, s.State.User.Discriminator))
	})
	err := discord.S.Open()
	if err != nil {
		l.Info("Cannot open the session", "ERROR", err)
	}

	l.Info("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := discord.S.ApplicationCommandCreate(discord.S.State.User.ID, discord.S.State.Application.GuildID, v)
		if err != nil {
			l.Error("could not add command", "COMMAND", v.Name, "ERROR", err)
		}
		registeredCommands[i] = cmd
	}

	defer discord.S.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop


	log.Println("Removing commands...")

	for _, v := range registeredCommands {
		err := discord.S.ApplicationCommandDelete(discord.S.State.User.ID, discord.S.State.Application.GuildID, v.ID)
		if err != nil {
			log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
		}
	}


	log.Println("Gracefully shutting down.")
}
