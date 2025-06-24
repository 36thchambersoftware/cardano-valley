package discord

import (
	"cardano-valley/pkg/blockfrost"
	"cardano-valley/pkg/cv"
	"cardano-valley/pkg/logger"
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	LINK_WALLET_COMMAND = discordgo.ApplicationCommand{
		Version:                  "0.01",
		Name:                     "link-wallet",
		Description:              "Link your Cardano wallet to your Discord account.",
		DefaultMemberPermissions: &ADMIN,
	}

	LINK_WALLET_MODAL_NAME = "link-wallet"
	
	LINK_WALLET_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		amount := rand.Intn(1_000)
		linkAmount := fmt.Sprintf("1%s", strconv.Itoa(amount))
		linkAmountDisplay := fmt.Sprintf("1.%s", strconv.Itoa(amount)) // strconv.FormatFloat(1.0 + (float64(amount) / float64(1000000)), 'f', -1, 64)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: fmt.Sprintf("%s_%s_%s", LINK_WALLET_MODAL_NAME, i.Interaction.Member.User.ID, linkAmount),
				Title:    "Link Your Wallet",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "tx_id",
								Label:       fmt.Sprintf("Send %s ADA to yourself, then paste tx ID", linkAmountDisplay),
								Style:       discordgo.TextInputShort,
								Placeholder: fmt.Sprintf("After sending %s ADA to yourself, paste your tx ID here", linkAmountDisplay),
								Required:    true,
								MaxLength:   64,
								MinLength:   64,
							},
						},
					},
				},
			},
		})

		logger.Record.Info(fmt.Sprintf("User %s initiated wallet linking with amount %s", i.Interaction.Member.User.ID, linkAmount))
	}

	LINK_WALLET_MODAL_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ModalSubmitInteractionData) {
		txID := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value // Assuming the tx ID is the first option
		logger.Record.Info(fmt.Sprintf("User %s submitted wallet linking with tx ID: %s", i.Interaction.Member.User.ID, txID))
		// Acknowledge the interaction
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Checking the transaction ID you provided. This may take a minute...",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

		amount := strings.Split(data.CustomID, "_")[2] // Extract the amount from the custom ID
		
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		time.Sleep(45 * time.Second) // Simulate a delay for checking the transaction
	    // Check blockfrost using the tx ID provided by the user
		utxo, err := blockfrost.GetTransaction(ctx, txID)
		if err != nil {
			content := fmt.Sprintf("Error checking transaction ID %s: %v", txID, err)
			logger.Record.Error(fmt.Sprintf("Error checking transaction ID %s: %v", txID, err))
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &content,
			})
			return
		}

		if len(utxo.Inputs) < 1 && len(utxo.Outputs) < 1 && utxo.Outputs[0].Address != utxo.Inputs[0].Address && utxo.Outputs[0].Amount[0].Unit != "lovelace" && utxo.Outputs[0].Amount[0].Quantity != amount {
			content := fmt.Sprintf("Transaction ID %s is invalid or does not match the expected amount of %s lovelace.", txID, amount)
			logger.Record.Error(fmt.Sprintf("Invalid transaction ID %s provided by user %s", txID, i.Interaction.Member.User.ID))
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &content,
			})
			return
		}

		addr := utxo.Outputs[0].Address
		address, err := blockfrost.GetAddress(ctx, addr)
		if err != nil {
			content := fmt.Sprintf("Error retrieving address information for %s: %v", addr, err)
			logger.Record.Error(fmt.Sprintf("Error retrieving address information for %s: %v", addr, err))
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &content,
			})
			return
		}

		// See if they have an ada handle
		var handles []string
		for _, v := range address.Amount {
			if strings.HasPrefix(v.Unit, blockfrost.ADA_HANDLE_POLICY_ID) {
				hexHandle := strings.Split(v.Unit, blockfrost.CIP68v1_NONSENSE)[1]
				handle, err := hex.DecodeString(hexHandle)
				if err != nil {
					logger.Record.Error(fmt.Sprintf("Error decoding ADA handle for user %s: %v", i.Interaction.Member.User.ID, err))
				}
				handles = append(handles, fmt.Sprintf("%s%s", blockfrost.ADA_HANDLE_PREFIX, string(handle)))
			}
		}

		user := cv.LoadUser(i.Member.User.ID)
		user.LinkedWallets = append(user.LinkedWallets, cv.Address{
			Payment: address.Address,
			Stake:   *address.StakeAddress,
		})
		user.Save()

		content := ""
		if len(handles) > 0 {
			content = fmt.Sprintf("Your wallet has been linked successfully! %s", strings.Join(handles, ", "))
			logger.Record.Info(fmt.Sprintf("User %s successfully linked wallet with tx ID: %s and ADA handle: %s", i.Interaction.Member.User.ID, txID, strings.Join(handles, ", ")))
		} else {
			content = fmt.Sprintf("Your wallet has been linked successfully! %s", address.Address)
		}

		logger.Record.Info(fmt.Sprintf("User %s successfully linked wallet with tx ID: %s", i.Interaction.Member.User.ID, txID))
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
	}
)