package discord

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bwmarrin/discordgo"
)
var CREATE_AIRDROP_COMMAND = discordgo.ApplicationCommand{
	Name:        "create-airdrop",
	Description: "Create a new ADA airdrop (file OR policy_id required).",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionInteger,
			Name:        "total_ada",
			Description: "Total ADA for airdrop (e.g., 500 or 767, etc. We'll calculate per NFT.)",
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionAttachment,
			Name:        "holders_file",
			Description: "JSON file: [{\"address\":\"addr...\",\"quantity\":N}, ...]",
			Required:    false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "policy_id",
			Description: "Policy ID to fetch holders from chain",
			Required:    false,
		},
	},
}

var CREATE_AIRDROP_HANDLER = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	var (
		attachment *discordgo.MessageAttachment
		policyID   string
		adaPerNFT  float64
		totalAda   uint64
	)

	for _, opt := range data.Options {
		switch opt.Name {
		case "holders_file":
			attachmentID := opt.Value.(string)
			attachment = i.ApplicationCommandData().Resolved.Attachments[attachmentID]
		case "policy_id":
			policyID = opt.StringValue()
		case "total_ada":
			totalAda = uint64(opt.IntValue())
		}
	}

	if attachment == nil && policyID == "" {
		respondError(s, i, "You must provide either a holders JSON file or a policy_id.")
		return
	}

	// Respond immediately (ephemeral) while we process
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "Creating your airdrop session…",
		},
	})

	// 1) Load holders
	var holders []Holder
	var err error
	if attachment != nil {
		holders, err = loadHoldersFromAttachment(attachment.URL)
		if err != nil {
			followupError(s, i, "Failed to parse holders file. Make sure it follows this format: JSON file: [{\"address\":\"addr...\",\"quantity\":N}, ...] "+err.Error())
			return
		}
	} else {
		holders, err = queryHoldersByPolicy_Blockfrost(policyID, getEnv("BLOCKFROST_API_KEY"))
		if err != nil {
			followupError(s, i, "Failed to fetch holders by policy: "+err.Error())
			return
		}
	}
	if len(holders) == 0 {
		followupError(s, i, "No holders found.")
		return
	}

	// Normalize: drop zero/neg qty and invalid addrs
	filtered := make([]Holder, 0, len(holders))
	for _, h := range holders {
		if h.Quantity > 0 && strings.HasPrefix(h.Address, "addr") {
			filtered = append(filtered, h)
		}
	}
	holders = filtered

	// 2) Totals
	totalNFTs := uint64(0)
	for _, h := range holders {
		totalNFTs += h.Quantity
	}
	adaPerNFT = float64(totalAda) / float64(totalNFTs)
	totalRecipients := uint64(len(holders))
	totalLovelace := totalAda * 1_000_000
	totalWithBuffer := totalLovelace + feeBufferLovelace + serviceFeeLovelace

	// 3) Create ephemeral wallet for this airdrop
	session, err := createTempWallet(i.Member.User.ID)
	if err != nil {
		followupError(s, i, "Wallet creation failed: "+err.Error())
		return
	}

	session.ADAperNFT = adaPerNFT
	session.PolicyID = policyID
	session.Holders = holders
	session.TotalNFTs = totalNFTs
	session.TotalRecipients = totalRecipients
	session.TotalLovelaceRequired = totalWithBuffer
	if attachment != nil {
		// persist the raw JSON holders for later reference
		raw, _ := json.MarshalIndent(holders, "", "  ")
		p := filepath.Join(session.WalletDir, "holders.json")
		_ = os.WriteFile(p, raw, 0600)
		session.HoldersPath = p
	}
	if err := saveSession(session); err != nil {
		followupError(s, i, "Failed to persist session: "+err.Error())
		return
	}

	// 4) Show sanity-check / deposit info
	embed := &discordgo.MessageEmbed{
		Title:       "Airdrop Setup",
		Description: "Please deposit the funds to the address below. We'll automatically start once funds arrive.",
		Color:       0x3aa657,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Policy ID", Value: valOr(policyID, "—"), Inline: true},
			{Name: "Recipients", Value: fmt.Sprintf("%d", totalRecipients), Inline: true},
			{Name: "Total NFTs", Value: fmt.Sprintf("%d", totalNFTs), Inline: true},
			{Name: "ADA per NFT", Value: fmt.Sprintf("%.6f", adaPerNFT), Inline: true},
			{Name: "Required ADA (incl. 5 ADA for tx fees)", Value: fmt.Sprintf("%.6f", float64(totalWithBuffer)/1_000_000.0), Inline: true},
			{Name: "Service Fee", Value: "20 ADA", Inline: true},
			{Name: "Deposit Address", Value: "```\n" + session.Address + "\n```", Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "We’ll watch this address until funded (no timeout).",
		},
	}
	_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
	})

	// 5) Kick off a watcher goroutine (detached); it persists stage, so safe on restarts
	go watchAndRunAirdrop(s, i, session.SessionID)
}