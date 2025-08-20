package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// TestHelper provides utilities for testing Discord airdrop functionality
type TestHelper struct {
	TempDir    string
	MockServer *httptest.Server
}

// NewTestHelper creates a new test helper with temporary directories and mock servers
func NewTestHelper(t *testing.T) *TestHelper {
	tempDir := t.TempDir()
	
	helper := &TestHelper{
		TempDir: tempDir,
	}
	
	// Setup mock HTTP server for Blockfrost API
	helper.MockServer = httptest.NewServer(http.HandlerFunc(helper.mockBlockfrostHandler))
	
	return helper
}

// Cleanup cleans up test resources
func (th *TestHelper) Cleanup() {
	if th.MockServer != nil {
		th.MockServer.Close()
	}
}

// SetupTempAirdropDir creates a temporary airdrop directory structure
func (th *TestHelper) SetupTempAirdropDir() string {
	airdropDir := filepath.Join(th.TempDir, "airdrops")
	os.MkdirAll(filepath.Join(airdropDir, "active"), 0755)
	os.MkdirAll(filepath.Join(airdropDir, "sessions"), 0755)
	return airdropDir
}

// CreateMockSession creates a mock AirdropSession for testing
func (th *TestHelper) CreateMockSession(sessionID, userID string) *AirdropSession {
	return &AirdropSession{
		DiscordUserID:         userID,
		SessionID:             sessionID,
		PolicyID:              "test_policy_123",
		ADAperNFT:             2.5,
		TotalNFTs:             100,
		TotalRecipients:       50,
		TotalLovelaceRequired: 255000000, // 255 ADA (250 + 5 buffer)
		Stage:                 StageAwaitingFunds,
		Address:               "addr1q9test123456789",
		WalletDir:             filepath.Join(th.TempDir, "wallets", sessionID),
		Holders: []Holder{
			{Address: "addr1q9holder1", Quantity: 40},
			{Address: "addr1q9holder2", Quantity: 60},
		},
	}
}

// CreateMockDiscordInteraction creates a mock Discord interaction for testing
func (th *TestHelper) CreateMockDiscordInteraction(userID string, options []*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name:    "create-airdrop",
				Options: options,
				Resolved: &discordgo.ApplicationCommandInteractionDataResolved{
					Attachments: map[string]*discordgo.MessageAttachment{
						"test_attachment_id": {
							URL:      th.MockServer.URL + "/test-holders.json",
							Filename: "holders.json",
						},
					},
				},
			},
			Member: &discordgo.Member{
				User: &discordgo.User{ID: userID},
			},
		},
	}
}

// CreateValidAirdropOptions creates valid Discord command options for airdrop
func (th *TestHelper) CreateValidAirdropOptions() []*discordgo.ApplicationCommandInteractionDataOption {
	return []*discordgo.ApplicationCommandInteractionDataOption{
		{
			Name:  "ada_per_nft",
			Type:  discordgo.ApplicationCommandOptionNumber,
			Value: 2.5,
		},
		{
			Name:  "policy_id",
			Type:  discordgo.ApplicationCommandOptionString,
			Value: "test_policy_123",
		},
		{
			Name:  "refund_address",
			Type:  discordgo.ApplicationCommandOptionString,
			Value: "addr1q9refund123",
		},
	}
}

// CreateHoldersFileOptions creates Discord options with a holders file attachment
func (th *TestHelper) CreateHoldersFileOptions() []*discordgo.ApplicationCommandInteractionDataOption {
	return []*discordgo.ApplicationCommandInteractionDataOption{
		{
			Name:  "ada_per_nft",
			Type:  discordgo.ApplicationCommandOptionNumber,
			Value: 1.5,
		},
		{
			Name:  "holders_file",
			Type:  discordgo.ApplicationCommandOptionAttachment,
			Value: "test_attachment_id",
		},
	}
}

// mockBlockfrostHandler handles mock Blockfrost API requests
func (th *TestHelper) mockBlockfrostHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Mock holders file endpoint
	if r.URL.Path == "/test-holders.json" {
		holders := []Holder{
			{Address: "addr1q9mock1", Quantity: 3},
			{Address: "addr1q9mock2", Quantity: 2},
			{Address: "addr1q9mock3", Quantity: 1},
		}
		json.NewEncoder(w).Encode(holders)
		return
	}

	// Mock Blockfrost policy assets endpoint
	if r.URL.Path == "/api/v0/assets/policy/test_policy_123" {
		assets := []struct {
			Asset string `json:"asset"`
		}{
			{Asset: "test_policy_123asset1"},
			{Asset: "test_policy_123asset2"},
		}
		json.NewEncoder(w).Encode(assets)
		return
	}

	// Mock Blockfrost asset addresses endpoint
	if r.URL.Path == "/api/v0/assets/test_policy_123asset1/addresses" {
		addresses := []struct {
			Address  string `json:"address"`
			Quantity string `json:"quantity"`
		}{
			{Address: "addr1q9holder1", Quantity: "1"},
			{Address: "addr1q9holder2", Quantity: "2"},
		}
		json.NewEncoder(w).Encode(addresses)
		return
	}

	if r.URL.Path == "/api/v0/assets/test_policy_123asset2/addresses" {
		addresses := []struct {
			Address  string `json:"address"`
			Quantity string `json:"quantity"`
		}{
			{Address: "addr1q9holder2", Quantity: "1"},
			{Address: "addr1q9holder3", Quantity: "1"},
		}
		json.NewEncoder(w).Encode(addresses)
		return
	}

	// Mock Blockfrost address balance endpoint
	if r.URL.Path == "/api/v0/addresses/addr1q9test123456789" {
		balance := struct {
			Amount []struct {
				Unit     string `json:"unit"`
				Quantity string `json:"quantity"`
			} `json:"amount"`
		}{
			Amount: []struct {
				Unit     string `json:"unit"`
				Quantity string `json:"quantity"`
			}{
				{Unit: "lovelace", Quantity: "255000000"}, // 255 ADA
			},
		}
		json.NewEncoder(w).Encode(balance)
		return
	}

	// Default: return empty response
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error": "not found"}`))
}

// SetupTestEnvironment sets up common environment variables for testing
func (th *TestHelper) SetupTestEnvironment(t *testing.T) func() {
	// Set required environment variables
	envVars := map[string]string{
		"BLOCKFROST_API_KEY":      "test_api_key_123",
		"CARDANO_VALLEY_ADDRESS":  "addr1q9cardano_valley_test",
		"AIRDROP_PUBLIC_CHANNEL_ID": "123456789",
	}

	// Store original values to restore later
	originalVars := make(map[string]string)
	for key, value := range envVars {
		originalVars[key] = os.Getenv(key)
		os.Setenv(key, value)
	}

	// Return cleanup function
	return func() {
		for key, originalValue := range originalVars {
			if originalValue == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, originalValue)
			}
		}
	}
}

// AssertHolderEquals asserts that two holders are equal
func (th *TestHelper) AssertHolderEquals(t *testing.T, expected, actual Holder) {
	if expected.Address != actual.Address {
		t.Errorf("Holder address mismatch: expected %s, got %s", expected.Address, actual.Address)
	}
	if expected.Quantity != actual.Quantity {
		t.Errorf("Holder quantity mismatch: expected %d, got %d", expected.Quantity, actual.Quantity)
	}
}

// AssertSessionEquals asserts that key fields of two sessions are equal
func (th *TestHelper) AssertSessionEquals(t *testing.T, expected, actual *AirdropSession) {
	if expected.DiscordUserID != actual.DiscordUserID {
		t.Errorf("Session DiscordUserID mismatch: expected %s, got %s", expected.DiscordUserID, actual.DiscordUserID)
	}
	if expected.SessionID != actual.SessionID {
		t.Errorf("Session SessionID mismatch: expected %s, got %s", expected.SessionID, actual.SessionID)
	}
	if expected.ADAperNFT != actual.ADAperNFT {
		t.Errorf("Session ADAperNFT mismatch: expected %f, got %f", expected.ADAperNFT, actual.ADAperNFT)
	}
	if expected.Stage != actual.Stage {
		t.Errorf("Session Stage mismatch: expected %s, got %s", expected.Stage, actual.Stage)
	}
}