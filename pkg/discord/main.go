// Package discord provides Discord bot command handlers and utilities for Cardano Valley.
package discord

import (
	"cardano-valley/pkg/cv"
	"cardano-valley/pkg/logger"
	"context"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	S                   *discordgo.Session
	DISCORD_WEBHOOK_URL string
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

	go rewardUpdater(ctx)
}

func RefreshCommands() {
	appID, ok := os.LookupEnv("CARDANO_VALLEY_APPLICATION_ID")
	if !ok {
		log.Fatalf("Missing application id")
	}
	registeredCommands, err := S.ApplicationCommands(appID, "")
	if err != nil {
		log.Panicf("Cannot retrieve commands:\n%v", err)
	}

	guildID := ""
	_, err = S.ApplicationCommandBulkOverwrite(appID, guildID, registeredCommands)
	if err != nil {
		log.Panicf("Cannot overwrite commands:\n%v", err)
	}
}

func rewardUpdater(ctx context.Context) {
	for {
		now := time.Now().UTC()
		// Calculate next 0:00 UTC
		next := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)
		// next := now.Add(10*time.Second) // for testing
		time.Sleep(time.Until(next))

		logger.Record.Info("Updating rewards...")
		// Get all guilds associated with Cardano Valley
		var rewards []cv.Rewards
		configs := cv.LoadConfigs()
		for _, config := range configs {
			rewards = append(rewards, config.Rewards...)
		}

		// Get all users associated with Cardano Valley
		users := cv.LoadUsers()

		rewardLog := logger.Record.WithGroup("CYCLE")
		for _, user := range users {
			userLog := rewardLog.With("USER", user.ID)
			for _, config := range configs {
				guildLog := userLog.With("GUILD", config.GuildID)
				member, err := S.GuildMember(config.GuildID, user.ID)
				if err != nil {
					guildLog.Warn("Failed to get guild member", "ERROR", err)
					continue
				}

				for _, reward := range rewards {
					rewardLog := guildLog.With("REWARD", reward.Name)
					matchingRoles := cv.SliceMatches(member.Roles, reward.RolesEligible)
					// Check if user has a role associated with a reward
					if len(matchingRoles) > 0 {
						// User is eligible for the reward!
						rewardLog.Info("ELIGIBLE", "AMOUNT", reward.AmountPerUser)
						user.Balance[reward.RewardToken] += reward.AmountPerUser
					}
				}
			}

			user.Save()
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