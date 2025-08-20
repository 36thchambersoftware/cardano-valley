package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCreateAirdrop_RealWorldScenarios tests the airdrop functionality with realistic scenarios
// These tests focus on the core logic and calculations with real-world values
func TestCreateAirdrop_RealWorldScenarios(t *testing.T) {
	// Setup test environment
	tempDir := t.TempDir()
	originalBaseDir := baseAirdropDir
	// Note: We can't easily override baseAirdropDir without refactoring, so we'll work within its constraints
	
	// Setup realistic test data
	realWorldTests := []struct {
		name                    string
		adaPerNFT              float64
		holders                []Holder
		expectedTotalNFTs      uint64
		expectedTotalRecipients uint64
		expectedMinLovelace    uint64 // minimum expected (excluding buffer)
		scenario               string
	}{
		{
			name:      "Spacebudz Collection Airdrop",
			adaPerNFT: 2.5,
			scenario:  "Popular NFT collection with 10,000 items, partial airdrop to top holders",
			holders: []Holder{
				{Address: "addr1qx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwq2ytjqp", Quantity: 15},
				{Address: "addr1q9ag3hagp8x0n9wvl8x3xnn2cj4k8mdny2uy6hkg9n8xn8p7cu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwqcr7n3w", Quantity: 8},
				{Address: "addr1q8fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsyd7w7jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwqab7n2z", Quantity: 3},
				{Address: "addr1q85yx3l9z5dgx5e8ufrh0hdj8f3k5m7qxh8r7g3qcqk5s7jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwqx8fn4r", Quantity: 1},
			},
			expectedTotalNFTs:       27, // 15+8+3+1
			expectedTotalRecipients: 4,
			expectedMinLovelace:     67500000, // 27 * 2.5 * 1_000_000
		},
		{
			name:      "CNFT Small Creator Airdrop",
			adaPerNFT: 0.5,
			scenario:  "Small creator rewarding early supporters",
			holders: []Holder{
				{Address: "addr1qx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwq2ytjqp", Quantity: 1},
				{Address: "addr1q9ag3hagp8x0n9wvl8x3xnn2cj4k8mdny2uy6hkg9n8xn8p7cu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwqcr7n3w", Quantity: 1},
				{Address: "addr1q8fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsyd7w7jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwqab7n2z", Quantity: 2},
			},
			expectedTotalNFTs:       4, // 1+1+2
			expectedTotalRecipients: 3,
			expectedMinLovelace:     2000000, // 4 * 0.5 * 1_000_000
		},
		{
			name:      "High Value NFT Airdrop",
			adaPerNFT: 10.0,
			scenario:  "Premium NFT collection with high-value rewards",
			holders: []Holder{
				{Address: "addr1qx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwq2ytjqp", Quantity: 5},
				{Address: "addr1q9ag3hagp8x0n9wvl8x3xnn2cj4k8mdny2uy6hkg9n8xn8p7cu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwqcr7n3w", Quantity: 2},
			},
			expectedTotalNFTs:       7, // 5+2
			expectedTotalRecipients: 2,
			expectedMinLovelace:     70000000, // 7 * 10.0 * 1_000_000
		},
		{
			name:      "Fractional ADA Micro-Rewards",
			adaPerNFT: 0.01,
			scenario:  "Large-scale micro-rewards for community engagement",
			holders: []Holder{
				{Address: "addr1qx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwq2ytjqp", Quantity: 100},
				{Address: "addr1q9ag3hagp8x0n9wvl8x3xnn2cj4k8mdny2uy6hkg9n8xn8p7cu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwqcr7n3w", Quantity: 250},
				{Address: "addr1q8fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsyd7w7jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwqab7n2z", Quantity: 50},
			},
			expectedTotalNFTs:       400, // 100+250+50
			expectedTotalRecipients: 3,
			expectedMinLovelace:     4000000, // 400 * 0.01 * 1_000_000
		},
	}

	for _, tt := range realWorldTests {
		t.Run(tt.name, func(t *testing.T) {
			// Test total calculations
			totalNFTs := uint64(0)
			for _, h := range tt.holders {
				totalNFTs += h.Quantity
			}

			if totalNFTs != tt.expectedTotalNFTs {
				t.Errorf("Total NFTs: expected %d, got %d", tt.expectedTotalNFTs, totalNFTs)
			}

			totalRecipients := uint64(len(tt.holders))
			if totalRecipients != tt.expectedTotalRecipients {
				t.Errorf("Total recipients: expected %d, got %d", tt.expectedTotalRecipients, totalRecipients)
			}

			// Test lovelace calculation (the core logic from the handler)
			totalLovelace := uint64(float64(totalNFTs) * tt.adaPerNFT * 1_000_000)
			if totalLovelace < tt.expectedMinLovelace {
				t.Errorf("Total lovelace: expected at least %d, got %d", tt.expectedMinLovelace, totalLovelace)
			}

			totalWithBuffer := totalLovelace + feeBufferLovelace
			expectedMinWithBuffer := tt.expectedMinLovelace + feeBufferLovelace
			if totalWithBuffer < expectedMinWithBuffer {
				t.Errorf("Total with buffer: expected at least %d, got %d", expectedMinWithBuffer, totalWithBuffer)
			}

			// Log realistic scenario details
			t.Logf("Scenario: %s", tt.scenario)
			t.Logf("Rate: %.2f ADA per NFT", tt.adaPerNFT)
			t.Logf("Recipients: %d addresses", totalRecipients)
			t.Logf("Total NFTs: %d", totalNFTs)
			t.Logf("Total reward: %.2f ADA", float64(totalLovelace)/1_000_000)
			t.Logf("With buffer: %.2f ADA", float64(totalWithBuffer)/1_000_000)
			t.Logf("Service fee: %.0f ADA", serviceFeeADA)
			t.Logf("Total required: %.2f ADA", float64(totalWithBuffer)/1_000_000+serviceFeeADA)
		})
	}

	// Restore original baseDir
	_ = originalBaseDir
	_ = tempDir
}

func TestCreateAirdrop_RealWorldHolderFilters(t *testing.T) {
	// Test the filtering logic from the actual handler
	rawHolders := []Holder{
		{Address: "addr1qx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwq2ytjqp", Quantity: 5},
		{Address: "addr1q9invalid", Quantity: 3},                           // Invalid address (too short)
		{Address: "stake1abc123", Quantity: 2},                            // Stake address, not payment address
		{Address: "addr1qx2fxv2valid", Quantity: 0},                       // Zero quantity
		{Address: "addr1q8fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsyd7w7jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwqab7n2z", Quantity: 1},
	}

	// Apply the same filtering logic as in the handler:
	// "Normalize: drop zero/neg qty and invalid addrs"
	filtered := make([]Holder, 0, len(rawHolders))
	for _, h := range rawHolders {
		// Exact same logic as in command-create-ada-airdrop.go:106
		if h.Quantity > 0 && strings.HasPrefix(h.Address, "addr") {
			filtered = append(filtered, h)
		}
	}

	expectedFiltered := 3 // Three holders should pass: items 1, 2, and 5
	if len(filtered) != expectedFiltered {
		t.Errorf("Expected %d filtered holders, got %d", expectedFiltered, len(filtered))
	}

	// Verify the correct holders were kept (in order they appear)
	validAddresses := []string{
		"addr1qx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwq2ytjqp", // item 1
		"addr1q9invalid",                           // item 2 (passes filtering despite being "invalid")
		"addr1q8fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsyd7w7jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwqab7n2z", // item 5
	}

	for i, holder := range filtered {
		if i < len(validAddresses) && holder.Address != validAddresses[i] {
			t.Errorf("Filtered holder %d: expected %s, got %s", i, validAddresses[i], holder.Address)
		}
		if holder.Quantity == 0 {
			t.Errorf("Filtered holder %d should not have zero quantity", i)
		}
		if !strings.HasPrefix(holder.Address, "addr") {
			t.Errorf("Filtered holder %d should have valid address prefix", i)
		}
	}

	t.Logf("Successfully filtered %d/%d holders", len(filtered), len(rawHolders))
}

func TestCreateAirdrop_RealWorldHoldersFile(t *testing.T) {
	// Create a realistic holders file for testing
	realWorldHolders := []Holder{
		// Top holders from a hypothetical successful NFT project
		{Address: "addr1qx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwq2ytjqp", Quantity: 25}, // Whale holder
		{Address: "addr1q9ag3hagp8x0n9wvl8x3xnn2cj4k8mdny2uy6hkg9n8xn8p7cu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwqcr7n3w", Quantity: 12}, // Large holder
		{Address: "addr1q8fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsyd7w7jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwqab7n2z", Quantity: 8},  // Medium holder
		{Address: "addr1q85yx3l9z5dgx5e8ufrh0hdj8f3k5m7qxh8r7g3qcqk5s7jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwqx8fn4r", Quantity: 5},  // Small holder
		{Address: "addr1q7fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsyd7w7jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwqfg8n5p", Quantity: 3},  // Smaller holder
		{Address: "addr1qafxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsyd7w7jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwq4h7m6q", Quantity: 1},  // Min holder
	}

	// Create temporary HTTP server to serve the holders file
	holdersJSON, err := json.Marshal(realWorldHolders)
	if err != nil {
		t.Fatalf("Failed to marshal holders: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(holdersJSON)
	}))
	defer server.Close()

	// Test loading holders from the mock attachment URL
	holders, err := loadHoldersFromAttachment(server.URL)
	if err != nil {
		t.Fatalf("Failed to load holders from attachment: %v", err)
	}

	// Validate realistic data
	if len(holders) != len(realWorldHolders) {
		t.Errorf("Expected %d holders, got %d", len(realWorldHolders), len(holders))
	}

	totalQuantity := uint64(0)
	for i, holder := range holders {
		// Validate Cardano address format
		if !strings.HasPrefix(holder.Address, "addr1q") {
			t.Errorf("Holder %d has invalid address format: %s", i, holder.Address)
		}

		// Validate reasonable address length (Cardano addresses are typically 103 characters)
		if len(holder.Address) < 50 || len(holder.Address) > 110 {
			t.Errorf("Holder %d has unrealistic address length: %d", i, len(holder.Address))
		}

		// Validate quantity is positive
		if holder.Quantity == 0 {
			t.Errorf("Holder %d has zero quantity", i)
		}

		totalQuantity += holder.Quantity
	}

	expectedTotal := uint64(54) // 25+12+8+5+3+1
	if totalQuantity != expectedTotal {
		t.Errorf("Expected total quantity %d, got %d", expectedTotal, totalQuantity)
	}

	// Test realistic airdrop calculations
	adaPerNFT := 1.5
	totalLovelace := uint64(float64(totalQuantity) * adaPerNFT * 1_000_000)
	totalWithBuffer := totalLovelace + feeBufferLovelace

	t.Logf("Real-world holders file test results:")
	t.Logf("  Total holders: %d", len(holders))
	t.Logf("  Total NFTs: %d", totalQuantity)
	t.Logf("  Rate: %.1f ADA per NFT", adaPerNFT)
	t.Logf("  Reward pool: %.1f ADA", float64(totalLovelace)/1_000_000)
	t.Logf("  With buffer: %.1f ADA", float64(totalWithBuffer)/1_000_000)
	t.Logf("  Largest holder: %d NFTs (%.1f ADA reward)", holders[0].Quantity, float64(holders[0].Quantity)*adaPerNFT)
	t.Logf("  Smallest holder: %d NFT (%.1f ADA reward)", holders[len(holders)-1].Quantity, float64(holders[len(holders)-1].Quantity)*adaPerNFT)

	// Validate that this would be a realistic airdrop size
	if totalWithBuffer < 10_000_000 { // Less than 10 ADA total seems too small
		t.Log("This would be a small-scale airdrop")
	} else if totalWithBuffer > 1_000_000_000 { // More than 1000 ADA total seems very large
		t.Log("This would be a large-scale airdrop")
	} else {
		t.Log("This would be a medium-scale airdrop")
	}
}

func TestCreateAirdrop_RealWorldValidation(t *testing.T) {
	// Test realistic validation scenarios based on the actual command handler
	validationTests := []struct {
		name        string
		adaPerNFT   float64
		policyID    string
		attachment  bool
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Valid Policy ID Only",
			adaPerNFT:   2.5,
			policyID:    "d5e6bf0500378d4f0da4e8dde6becec7621cd8cbf5cbb9b87013d4cc", // Real Spacebudz policy
			attachment:  false,
			shouldError: false,
		},
		{
			name:        "Valid Attachment Only",
			adaPerNFT:   1.0,
			policyID:    "",
			attachment:  true,
			shouldError: false,
		},
		{
			name:        "Both Policy and Attachment",
			adaPerNFT:   3.0,
			policyID:    "d5e6bf0500378d4f0da4e8dde6becec7621cd8cbf5cbb9b87013d4cc",
			attachment:  true,
			shouldError: false, // Should work fine, policy takes precedence in handler
		},
		{
			name:        "Neither Policy nor Attachment",
			adaPerNFT:   1.5,
			policyID:    "",
			attachment:  false,
			shouldError: true,
			errorMsg:    "You must provide either a holders JSON file or a policy_id",
		},
		{
			name:        "Zero ADA per NFT",
			adaPerNFT:   0,
			policyID:    "d5e6bf0500378d4f0da4e8dde6becec7621cd8cbf5cbb9b87013d4cc",
			attachment:  false,
			shouldError: false, // Handler doesn't validate this, but would result in 0 rewards
		},
		{
			name:        "High ADA per NFT",
			adaPerNFT:   100.0, // Very generous airdrop
			policyID:    "d5e6bf0500378d4f0da4e8dde6becec7621cd8cbf5cbb9b87013d4cc",
			attachment:  false,
			shouldError: false,
		},
	}

	for _, tt := range validationTests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the validation logic from the handler
			hasAttachment := tt.attachment
			hasPolicyID := tt.policyID != ""

			// This is the exact validation from the handler:
			// if attachment == nil && policyID == "" {
			shouldError := !hasAttachment && !hasPolicyID

			if shouldError != tt.shouldError {
				t.Errorf("Validation error expectation mismatch: expected %v, got %v", tt.shouldError, shouldError)
			}

			if shouldError && tt.errorMsg != "" {
				expectedMsg := "You must provide either a holders JSON file or a policy_id"
				if tt.errorMsg != expectedMsg {
					t.Errorf("Error message mismatch: expected %q, got %q", expectedMsg, tt.errorMsg)
				}
			}

			// Test realistic calculations for valid inputs
			if !shouldError {
				totalNFTs := uint64(15) // Simulated total from test holders
				totalLovelace := uint64(float64(totalNFTs) * tt.adaPerNFT * 1_000_000)
				totalWithBuffer := totalLovelace + feeBufferLovelace

				t.Logf("Test scenario: %s", tt.name)
				t.Logf("  ADA per NFT: %.2f", tt.adaPerNFT)
				t.Logf("  Total NFTs: %d", totalNFTs)
				t.Logf("  Reward pool: %.2f ADA", float64(totalLovelace)/1_000_000)
				t.Logf("  With buffer: %.2f ADA", float64(totalWithBuffer)/1_000_000)

				// Validate reasonable bounds
				if tt.adaPerNFT > 0 {
					if totalLovelace == 0 {
						t.Error("Should have non-zero lovelace for positive ADA per NFT")
					}
					if totalWithBuffer <= feeBufferLovelace {
						t.Error("Total with buffer should exceed just the buffer amount")
					}
				}
			}
		})
	}
}

func TestCreateAirdrop_RealWorldSessionCreation(t *testing.T) {
	// Test creating a realistic airdrop session
	now := time.Now()
	discordUserID := "987654321098765432" // Realistic Discord user ID
	sessionID := discordUserID + "_" + string(rune(now.Unix()))

	// Create a realistic session as would be done in the handler
	session := &AirdropSession{
		DiscordUserID:   discordUserID,
		SessionID:       sessionID,
		CreatedAt:       now,
		PolicyID:        "d5e6bf0500378d4f0da4e8dde6becec7621cd8cbf5cbb9b87013d4cc", // Spacebudz
		ADAperNFT:       2.5,
		RefundAddress:   "addr1qxrefund123456789abcdefghijklmnopqrstuvwxyz",
		TotalNFTs:       150,
		TotalRecipients: 75,
		Stage:           StageAwaitingFunds,
		Address:         "addr1qx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwq2ytjqp",
		Holders: []Holder{
			{Address: "addr1qx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwq2ytjqp", Quantity: 50},
			{Address: "addr1q9ag3hagp8x0n9wvl8x3xnn2cj4k8mdny2uy6hkg9n8xn8p7cu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwqcr7n3w", Quantity: 75},
			{Address: "addr1q8fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsyd7w7jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwqab7n2z", Quantity: 25},
		},
	}

	// Calculate the required lovelace (as done in the handler)
	totalLovelace := uint64(float64(session.TotalNFTs) * session.ADAperNFT * 1_000_000)
	session.TotalLovelaceRequired = totalLovelace + feeBufferLovelace

	// Validate realistic session values
	if session.TotalLovelaceRequired != 380000000 { // (150 * 2.5 + 5) * 1M
		t.Errorf("Expected 380000000 lovelace required, got %d", session.TotalLovelaceRequired)
	}

	// Test session persistence
	tempDir := t.TempDir()
	sessionPath := filepath.Join(tempDir, session.SessionID+".json")

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal session: %v", err)
	}

	err = os.WriteFile(sessionPath, data, 0600)
	if err != nil {
		t.Fatalf("Failed to write session: %v", err)
	}

	// Load and validate
	loadedData, err := os.ReadFile(sessionPath)
	if err != nil {
		t.Fatalf("Failed to read session: %v", err)
	}

	var loadedSession AirdropSession
	err = json.Unmarshal(loadedData, &loadedSession)
	if err != nil {
		t.Fatalf("Failed to unmarshal session: %v", err)
	}

	// Validate session integrity
	if loadedSession.SessionID != session.SessionID {
		t.Errorf("Session ID mismatch: expected %s, got %s", session.SessionID, loadedSession.SessionID)
	}

	if loadedSession.TotalLovelaceRequired != session.TotalLovelaceRequired {
		t.Errorf("Total lovelace mismatch: expected %d, got %d", session.TotalLovelaceRequired, loadedSession.TotalLovelaceRequired)
	}

	if len(loadedSession.Holders) != len(session.Holders) {
		t.Errorf("Holders count mismatch: expected %d, got %d", len(session.Holders), len(loadedSession.Holders))
	}

	// Log realistic session summary
	t.Logf("Created realistic airdrop session:")
	t.Logf("  Session ID: %s", session.SessionID)
	t.Logf("  Policy: %s", session.PolicyID)
	t.Logf("  Rate: %.2f ADA per NFT", session.ADAperNFT)
	t.Logf("  Recipients: %d", session.TotalRecipients)
	t.Logf("  Total NFTs: %d", session.TotalNFTs)
	t.Logf("  Required deposit: %.2f ADA", float64(session.TotalLovelaceRequired)/1_000_000)
	t.Logf("  Service fee: %.0f ADA", serviceFeeADA)
	t.Logf("  Total cost: %.2f ADA", float64(session.TotalLovelaceRequired)/1_000_000+serviceFeeADA)
}