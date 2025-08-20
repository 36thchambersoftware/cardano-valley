// Package discord provides Discord bot command handlers and utilities for Cardano Valley.
package discord

import (
	"cardano-valley/pkg/blockfrost"
	"cardano-valley/pkg/cv"
	"cardano-valley/pkg/logger"
	"context"
	"log"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	S                   *discordgo.Session
	DISCORD_WEBHOOK_URL string
	ADMIN int64 = discordgo.PermissionAdministrator

	verifications   = make(map[string]Verification)
	verificationsMu sync.Mutex
)

type Verification struct {
	TxID          string
	ExpectedLovelace string
	UserID        string
	ResponseChan  chan string
}

func init() {
	// Skip initialization during testing
	if os.Getenv("GO_TESTING") == "true" {
		return
	}
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

	go rewardRoleUpdater(ctx)
	go rewardHolderUpdater(ctx)
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
	cmds, err := S.ApplicationCommandBulkOverwrite(appID, guildID, registeredCommands)
	if err != nil {
		log.Panicf("Cannot overwrite commands:\n%v", err)
	}

	logger.Record.Info("REFRESHED", "COMMANDS", cmds)
}

func rewardRoleUpdater(ctx context.Context) {
	for {
		now := time.Now().UTC()
		// Calculate next 0:00 UTC
		next := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)
		// next := now.Add(10*time.Second) // for testing
		time.Sleep(time.Until(next))

		logger.Record.Info("Updating role rewards...")

		// Get all guilds associated with Cardano Valley
		configs := cv.LoadConfigs()
		users := cv.LoadUsers()
		rewardLog := logger.Record.WithGroup("ROLE CYCLE")
		for _, user := range users {
			userLog := rewardLog.With("USER", user.ID)

			// Ensure user.Rewards is initialized
			if user.Rewards == nil {
				user.Rewards = make(map[cv.ServerID]cv.Balance)
			}

			for _, config := range configs {
				guildLog := userLog.With("GUILD", config.GuildID)

				member, err := S.GuildMember(string(config.GuildID), user.ID)
				if err != nil {
					// User not in guild, skip
					continue
				}

				// Ensure user.Rewards[GuildID] is initialized
				if _, ok := user.Rewards[config.GuildID]; !ok {
					user.Rewards[config.GuildID] = make(cv.Balance)
				}

				for key, reward := range config.Rewards {
					rewardLog := guildLog.With("REWARD", reward.Name, "BALANCE", reward.Balance)
					matchingRoles := cv.SliceMatches(member.Roles, reward.RolesEligible)

					if len(matchingRoles) > 0 {
						if reward.Balance - reward.RoleAmount <= 0 {
							rewardLog.Error("Reward balance is empty!")
						}
						rewardLog.Info("ELIGIBLE", "AMOUNT", reward.RoleAmount)

						// Get current reward entry or create a new one
						entry := user.Rewards[config.GuildID][reward.RewardToken]
						entry.Earned += reward.RoleAmount
						entry.LastClaimed = time.Now()

						// Reduce the reward balance available.
						config.Rewards[key].Balance -= reward.RoleAmount
						config.Save()

						// Save it back to the map
						user.Rewards[config.GuildID][reward.RewardToken] = entry
					}
				}
			}

			user.Save()
		}
	}
}

func rewardHolderUpdater(ctx context.Context) {
	for {
		now := time.Now().UTC()
		// Calculate next 0:00 UTC
		next := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC).Add(24 * time.Hour)
		// next := now.Add(10*time.Second) // for testing
		time.Sleep(time.Until(next))

		logger.Record.Info("Updating holder rewards...")

		// Get all guilds associated with Cardano Valley
		configs := cv.LoadConfigs()

		// Get all wallets of users associated with Cardano Valley
		holders := make(map[string]map[string]uint64) // userID -> token -> amount
		users := cv.LoadUsers()
		tokenSum := make(map[string]uint64) // token -> total amount held
		rewardLog := logger.Record.WithGroup("HOLDER CYCLE")
		for _, user := range users {
			userLog := rewardLog.With("USER", user.ID)
			for _, wallet := range user.LinkedWallets {
				addressLog := userLog.With("WALLET", wallet.Payment)
				// Check if the wallet is valid
				address, err := blockfrost.GetAddress(ctx, wallet.Payment)
				if err != nil {
					addressLog.Warn("Invalid linked wallet address")
					continue
				}

				for _, amount := range address.Amount {
					if amount.Unit == "lovelace" {
						// Skip ADA balance
						continue
					}

					if _, ok := holders[user.ID]; !ok {
						holders[user.ID] = make(map[string]uint64)
					}
					qty, err := strconv.ParseInt(amount.Quantity, 10, 64)
					if err != nil {
						addressLog.Error("Invalid quantity for token", "TOKEN", amount.Unit, "QUANTITY", amount.Quantity, "ERROR", err)
						continue
					}

					holders[user.ID][amount.Unit] += uint64(qty)
					tokenSum[amount.Unit] += uint64(qty)
				}
			}
		}



		for userID, holdings := range holders {
			userLog := rewardLog.With("USER", userID)

			// Ensure user.Rewards is initialized
			user := cv.LoadUser(userID)
			if user.Rewards == nil {
				user.Rewards = make(map[cv.ServerID]cv.Balance)
			}

			for _, config := range configs {
				guildLog := userLog.With("GUILD", config.GuildID)

				_, err := S.GuildMember(string(config.GuildID), userID)
				if err != nil {
					// User not in guild, skip
					continue
				}

				// Ensure user.Rewards[GuildID] is initialized
				if _, ok := user.Rewards[config.GuildID]; !ok {
					user.Rewards[config.GuildID] = make(cv.Balance)
				}

				for key, reward := range config.Rewards {
					rewardLog := guildLog.With("REWARD", reward.Name, "BALANCE", reward.Balance)
					for _, asset := range reward.AssetsEligible {
						// Check if the user holds the asset
						assetLog := rewardLog.With("ASSET", asset)
						if amount, ok := holdings[asset]; ok && amount > reward.AssetMinimum {
							assetLog.Info("HOLDER ELIGIBLE", "ASSET", asset, "AMOUNT", amount)

							// Get current reward entry or create a new one
							entry := user.Rewards[config.GuildID][reward.RewardToken]
							entry.Earned += reward.Balance / tokenSum[asset] * amount
							entry.LastClaimed = time.Now()

							// Reduce the reward balance available.
							config.Rewards[key].Balance -= entry.Earned
							config.Save()

							// Save it back to the map
							user.Rewards[config.GuildID][reward.RewardToken] = entry
						}
						
					}
					// if len(matchingRoles) > 0 {
					// 	if reward.Balance - reward.RoleAmount <= 0 {
					// 		rewardLog.Error("Reward balance is empty!")
					// 	}
					// 	rewardLog.Info("ELIGIBLE", "AMOUNT", reward.RoleAmount)

					// 	// Get current reward entry or create a new one
					// 	entry := user.Rewards[config.GuildID][reward.RewardToken]
					// 	entry.Earned += reward.RoleAmount
					// 	entry.LastClaimed = time.Now()

					// 	// Reduce the reward balance available.
					// 	config.Rewards[key].Balance -= reward.RoleAmount
					// 	config.Save()

					// 	// Save it back to the map
					// 	user.Rewards[config.GuildID][reward.RewardToken] = entry
					// }
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