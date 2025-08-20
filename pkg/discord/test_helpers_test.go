package discord

import (
	"testing"
)

func TestTestHelper_NewTestHelper(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	if helper.TempDir == "" {
		t.Error("TempDir should not be empty")
	}

	if helper.MockServer == nil {
		t.Error("MockServer should not be nil")
	}
}

func TestTestHelper_SetupTempAirdropDir(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	airdropDir := helper.SetupTempAirdropDir()
	if airdropDir == "" {
		t.Error("Airdrop directory should not be empty")
	}
}

func TestTestHelper_CreateMockSession(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	sessionID := "test_session_123"
	userID := "user_456"

	session := helper.CreateMockSession(sessionID, userID)

	if session.SessionID != sessionID {
		t.Errorf("Expected SessionID %s, got %s", sessionID, session.SessionID)
	}

	if session.DiscordUserID != userID {
		t.Errorf("Expected DiscordUserID %s, got %s", userID, session.DiscordUserID)
	}

	if session.Stage != StageAwaitingFunds {
		t.Errorf("Expected stage %s, got %s", StageAwaitingFunds, session.Stage)
	}

	if len(session.Holders) != 2 {
		t.Errorf("Expected 2 holders, got %d", len(session.Holders))
	}
}

func TestTestHelper_CreateMockDiscordInteraction(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	userID := "user_123"
	options := helper.CreateValidAirdropOptions()

	interaction := helper.CreateMockDiscordInteraction(userID, options)

	if interaction.Member.User.ID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, interaction.Member.User.ID)
	}

	if len(interaction.ApplicationCommandData().Options) != len(options) {
		t.Errorf("Expected %d options, got %d", len(options), len(interaction.ApplicationCommandData().Options))
	}
}

func TestTestHelper_CreateValidAirdropOptions(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	options := helper.CreateValidAirdropOptions()

	expectedOptions := []string{"ada_per_nft", "policy_id", "refund_address"}
	if len(options) != len(expectedOptions) {
		t.Errorf("Expected %d options, got %d", len(expectedOptions), len(options))
	}

	for i, opt := range options {
		if opt.Name != expectedOptions[i] {
			t.Errorf("Option %d: expected name %s, got %s", i, expectedOptions[i], opt.Name)
		}
	}
}

func TestTestHelper_CreateHoldersFileOptions(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	options := helper.CreateHoldersFileOptions()

	expectedOptions := []string{"ada_per_nft", "holders_file"}
	if len(options) != len(expectedOptions) {
		t.Errorf("Expected %d options, got %d", len(expectedOptions), len(options))
	}

	for i, opt := range options {
		if opt.Name != expectedOptions[i] {
			t.Errorf("Option %d: expected name %s, got %s", i, expectedOptions[i], opt.Name)
		}
	}
}

func TestTestHelper_AssertHolderEquals(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	holder1 := Holder{Address: "addr1", Quantity: 5}
	holder2 := Holder{Address: "addr1", Quantity: 5}

	// This should not fail (same holders)
	helper.AssertHolderEquals(t, holder1, holder2)
}

func TestTestHelper_AssertSessionEquals(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	session1 := helper.CreateMockSession("session1", "user1")
	session2 := helper.CreateMockSession("session1", "user1")

	// This should not fail (same sessions)
	helper.AssertSessionEquals(t, session1, session2)
}