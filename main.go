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
		&discord.LIST_SERVER_REWARDS_COMMAND,
		&discord.DASHBOARD_COMMAND,
		&discord.DEPOSIT_COMMAND,
		&discord.HELP_COMMAND,
		&discord.CONFIGURE_REWARD_COMMAND,
		&discord.LINK_WALLET_COMMAND,
		&discord.WITHDRAW_COMMAND,
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		discord.INITIALIZE_COMMAND.Name:      		discord.INITIALIZE_HANDLER,
		discord.REGISTER_COMMAND.Name:      		discord.REGISTER_HANDLER,
		discord.LIST_SERVER_REWARDS_COMMAND.Name: 	discord.LIST_SERVER_REWARDS_HANDLER,
		discord.DASHBOARD_COMMAND.Name:      		discord.DASHBOARD_HANDLER,
		discord.DEPOSIT_COMMAND.Name:      			discord.DEPOSIT_HANDLER,
		discord.HELP_COMMAND.Name:      			discord.HELP_HANDLER,
		discord.CONFIGURE_REWARD_COMMAND.Name:     discord.CONFIGURE_REWARD_HANDLER,
		discord.LINK_WALLET_COMMAND.Name:          discord.LINK_WALLET_HANDLER,
		discord.WITHDRAW_COMMAND.Name:             discord.WITHDRAW_HANDLER,
	}

	// Modal Handlers: Must be in this format! `name-of-modal` then finished with `_something`
	modals = []string{
		discord.CONFIGURE_REWARD_NAME_MODAL_NAME,
		discord.LINK_WALLET_MODAL_NAME,
	}
	modalHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ModalSubmitInteractionData){
		discord.CONFIGURE_REWARD_NAME_MODAL_NAME: discord.CONFIGURE_REWARD_NAME_MODAL_HANDLER,
		discord.LINK_WALLET_MODAL_NAME:            discord.LINK_WALLET_MODAL_HANDLER,
	}

	components = []string{
		discord.CONFIGURE_REWARD_ASSET_COMPONENT_NAME,
		discord.WITHDRAW_COMMAND_OPTIONLIST_NAME,
	}
	componentHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate, selected discordgo.MessageComponentInteractionData){
		discord.CONFIGURE_REWARD_ASSET_COMPONENT_NAME: discord.CONFIGURE_REWARD_ASSET_COMPONENT_HANDLER,
		discord.WITHDRAW_COMMAND_OPTIONLIST_NAME:      discord.WITHDRAW_COMMAND_OPTIONLIST_HANDLER,
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
		Arguments   []discord.Args `json:"options"`
	}

	Feature struct {
		Icon string
		Title string
		Description string
	}

	PageData struct {
		Title    string
		Subtitle string
		Features []Feature
		Year     int
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
}

func main() {
	defer mongo.Close(mongo.DB, dbctx, dbcancel)
	l := logger.Record

	// http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	tmp := template.Must(template.ParseFiles("templates/index.html"))
	// 	data := PageData{
	// 		Title:    "FarmFi by Cardano Valley",
	// 		Subtitle: "Staking as a service. Built for meme coins, powered by Cardano.",
	// 		Features: []Feature{
	// 			{Icon: "🌾", Title: "Stake Pools", Description: "Launch farming for your meme coin with zero setup."},
	// 			{Icon: "📊", Title: "Yield Dashboard", Description: "Real-time earnings, staking stats, and visuals."},
	// 			{Icon: "🤝", Title: "Community Focus", Description: "Built with the community in mind, plug-and-play for any Discord."},
	// 		},
	// 		Year:     2025,
	// 	}
	// 	err := tmp.Execute(w, data)
	// 	if err != nil {
	// 		http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	}
	// })

	// log.Println("Server started at http://localhost:8080")
	// http.ListenAndServe(":8080", nil)

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
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
			defer cancel()
			if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
				if _, ok := lockout[i.Member.User.ID]; !ok {
					lockout[i.Member.User.ID] = struct{}{}
					defer func() {
						delete(lockout, i.Member.User.ID)
					}()

					CommandHistory = mongo.DB.Database("cardano-valley").Collection("command-history")
					if _, err := CommandHistory.InsertOne(ctx, Command{
						Name:      i.ApplicationCommandData().Name,
						Timestamp: time.Now(),
						UserID:    i.Member.User.ID,
						GuildID:   i.GuildID,
						ChannelID: i.ChannelID,
						Arguments: discord.ExtractArgsFromSlash(i.ApplicationCommandData()),
					}); err != nil {
						logger.Record.Error("Could not log command", "CTX", dbctx, "ERROR", err)
					}
					h(s, i)
				} else {
					s.InteractionRespond(i.Interaction, lockoutResponse)
				}
			}
		case discordgo.InteractionModalSubmit:
			ctx, cancel := context.WithTimeout(context.Background(), 60 * time.Second)
			defer cancel()

			data := i.ModalSubmitData()

			pieces := strings.Split(data.CustomID, "_")
			if len(pieces) < 2 {
				content := "Invalid modal name format. Expected format: `name_of_modal_userid`."
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: &content,
				})
				return
			}

			if h, ok := modalHandlers[pieces[0]]; ok {
				CommandHistory = mongo.DB.Database("cardano-valley").Collection("modal-history")
				if _, err := CommandHistory.InsertOne(ctx, Command{
					Name:      pieces[0],
					Timestamp: time.Now(),
					UserID:    i.Member.User.ID,
					GuildID:   i.GuildID,
					ChannelID: i.ChannelID,
					Arguments: discord.ExtractArgsFromModal(i.ModalSubmitData()),
				}); err != nil {
					logger.Record.Error("Could not log modal input", "CTX", dbctx, "ERROR", err)
				}
				h(s, i, data)
			}
		case discordgo.InteractionMessageComponent:
			ctx, cancel := context.WithTimeout(context.Background(), 60 * time.Second)
			defer cancel()
			
			data := i.MessageComponentData()

			pieces := strings.Split(data.CustomID, "_")
			if len(pieces) < 2 {
				content := "Invalid modal name format. Expected format: `name_of_modal_userid`."
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: &content,
				})
				return
			}

			if h, ok := componentHandlers[pieces[0]]; ok {
				CommandHistory = mongo.DB.Database("cardano-valley").Collection("component-history")
				if _, err := CommandHistory.InsertOne(ctx, Command{
					Name:      pieces[0],
					Timestamp: time.Now(),
					UserID:    i.Member.User.ID,
					GuildID:   i.GuildID,
					ChannelID: i.ChannelID,
					Arguments: discord.ExtractArgsFromSelect(data),
				}); err != nil {
					logger.Record.Error("Could not log component interaction", "CTX", dbctx, "ERROR", err)
				}
				h(s, i, data)
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
