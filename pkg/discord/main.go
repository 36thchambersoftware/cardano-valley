package discord

import (
	"cardano-valley/pkg/cv"
	"cardano-valley/pkg/logger"
	"context"
	"log"
	"net/url"
	"os"
	"time"

	mongo "cardano-valley/pkg/db"

	"github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	S                   *discordgo.Session
	DISCORD_WEBHOOK_URL string
	LAST_UPDATE_TIME    map[string]int
	ADMIN int64 = discordgo.PermissionAdministrator
)

func init() {
	initDiscord()
	initWebhook()
}

func initDiscord() {
	token, ok := os.LookupEnv("CARDANO_VALLEY_TOKEN")
	if !ok {
		log.Fatalf("Missing token")
	}
	var err error
	S, err = discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	RefreshCommands()

	ctx := context.Background()
	LAST_UPDATE_TIME = make(map[string]int)

	go rewardUpdater(ctx)
}

func RefreshCommands() {
	appId, ok := os.LookupEnv("CARDANO_VALLEY_APPLICATION_ID")
	if !ok {
		log.Fatalf("Missing application id")
	}
	registeredCommands, err := S.ApplicationCommands(appId, "")
	if err != nil {
		log.Panicf("Cannot retrieve commands:\n%v", err)
	}

	guildID := ""
	_, err = S.ApplicationCommandBulkOverwrite(appId, guildID, registeredCommands)
	if err != nil {
		log.Panicf("Cannot overwrite commands:\n%v", err)
	}
}

func rewardUpdater(ctx context.Context) {
	// Connect to MongoDB
	collection := mongo.DB.Database("cardano-valley").Collection("user")

	for {
		now := time.Now().UTC()
		// Calculate next 0:00 UTC
		next := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)
		time.Sleep(time.Until(next))

		// Get all guilds associated with Cardano Valley
		configs := cv.LoadConfigs()

		// Get all users associated with Cardano Valley
		users := cv.LoadUsers()

		for _, user := range users {
			for _, config := range configs {
				// Check if the user is in the guild
				guild, err := S.Guild(config.GuildID)
				if err != nil {
					logger.Record.Warn("Failed to get guild", "GUILD", config.GuildID, "ERROR", err)
					continue
				}

				member, err := S.GuildMember(config.GuildID, user.ID)
				if err != nil {
					logger.Record.Warn("Failed to get guild member", "GUILD", config.GuildID, "USER", user.ID, "ERROR", err)
					continue
				}

				for _, reward := range config.Rewards {
					matchingRoles := cv.SliceMatches(member.Roles, reward.RolesEligible)
					// Check if user has a role associated with a reward
					
				}

			}
		}

		filter := bson.M{"_id": rewardID}
		update := bson.M{"$inc": bson.M{rewardFieldName: incrementValue}}
		_, err := collection.UpdateOne(context.Background(), filter, update, options.Update().SetUpsert(true))
		if err != nil {
			log.Printf("Failed to increment reward: %v", err)
		} else {
			log.Printf("Reward incremented at %v", next)
		}
	}
}

func initWebhook() {
	// DISCORD_WEBHOOK_URL
	webhook, ok := os.LookupEnv("DISCORD_WEBHOOK_URL")
	if !ok {
		log.Fatalf("Could not get DISCORD_WEBHOOK_URL")
	}

	webhookURL, err := url.Parse(webhook)
	if err != nil {
		log.Fatalf("Invalid webhook url %v", err)
	}

	DISCORD_WEBHOOK_URL = webhookURL.String()
}