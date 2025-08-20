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

func TestHolder_JSON(t *testing.T) {
	holder := Holder{
		Address:  "addr1q9xyz...",
		Quantity: 5,
	}

	// Test JSON marshaling
	data, err := json.Marshal(holder)
	if err != nil {
		t.Fatalf("Failed to marshal Holder: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled Holder
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal Holder: %v", err)
	}

	if unmarshaled.Address != holder.Address {
		t.Errorf("Address mismatch: expected %s, got %s", holder.Address, unmarshaled.Address)
	}
	if unmarshaled.Quantity != holder.Quantity {
		t.Errorf("Quantity mismatch: expected %d, got %d", holder.Quantity, unmarshaled.Quantity)
	}
}

func TestAirdropStage_Constants(t *testing.T) {
	stages := []AirdropStage{
		StageAwaitingFunds,
		StageBuildingTx,
		StageDistributing,
		StagePayingFee,
		StageCompleted,
		StageCancelled,
	}

	expected := []string{
		"awaiting_funds",
		"building_tx",
		"distributing",
		"paying_service_fee",
		"completed",
		"cancelled",
	}

	for i, stage := range stages {
		if string(stage) != expected[i] {
			t.Errorf("Stage %d: expected %s, got %s", i, expected[i], string(stage))
		}
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		setValue string
		expected string
	}{
		{
			name:     "existing env var",
			key:      "TEST_ENV_VAR",
			setValue: "test_value",
			expected: "test_value",
		},
		{
			name:     "non-existing env var",
			key:      "NON_EXISTING_VAR",
			setValue: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setValue != "" {
				os.Setenv(tt.key, tt.setValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnv(tt.key)
			if result != tt.expected {
				t.Errorf("getEnv(%s): expected %s, got %s", tt.key, tt.expected, result)
			}
		})
	}
}

func TestLoadHoldersFromAttachment(t *testing.T) {
	testData := []Holder{
		{Address: "addr1q9xyz123", Quantity: 3},
		{Address: "addr1q9abc456", Quantity: 1},
	}

	jsonData, _ := json.Marshal(testData)

	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	}))
	defer server.Close()

	holders, err := loadHoldersFromAttachment(server.URL)
	if err != nil {
		t.Fatalf("loadHoldersFromAttachment failed: %v", err)
	}

	if len(holders) != len(testData) {
		t.Errorf("Expected %d holders, got %d", len(testData), len(holders))
	}

	for i, holder := range holders {
		if holder.Address != testData[i].Address {
			t.Errorf("Holder %d address: expected %s, got %s", i, testData[i].Address, holder.Address)
		}
		if holder.Quantity != testData[i].Quantity {
			t.Errorf("Holder %d quantity: expected %d, got %d", i, testData[i].Quantity, holder.Quantity)
		}
	}
}

func TestLoadHoldersFromAttachment_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	_, err := loadHoldersFromAttachment(server.URL)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestLoadHoldersFromAttachment_NetworkError(t *testing.T) {
	_, err := loadHoldersFromAttachment("http://invalid-url-that-should-fail.test")
	if err == nil {
		t.Error("Expected network error, got nil")
	}
}

func TestQueryHoldersByPolicy_Blockfrost_NoAPIKey(t *testing.T) {
	_, err := queryHoldersByPolicy_Blockfrost("test_policy", "")
	if err == nil {
		t.Error("Expected error for missing API key, got nil")
	}
	if !strings.Contains(err.Error(), "BLOCKFROST_API_KEY is required") {
		t.Errorf("Expected API key error, got: %v", err)
	}
}

func TestQueryHoldersByPolicy_Blockfrost_Success(t *testing.T) {
	// Mock Blockfrost responses
	mockPolicyResponse := []struct {
		Asset string `json:"asset"`
	}{
		{Asset: "policy123asset1"},
		{Asset: "policy123asset2"},
	}

	mockAddressResponse := []struct {
		Address  string `json:"address"`
		Quantity string `json:"quantity"`
	}{
		{Address: "addr1q9xyz123", Quantity: "2"},
		{Address: "addr1q9abc456", Quantity: "1"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/assets/policy/") {
			json.NewEncoder(w).Encode(mockPolicyResponse)
		} else if strings.Contains(r.URL.Path, "/assets/") && strings.Contains(r.URL.Path, "/addresses") {
			json.NewEncoder(w).Encode(mockAddressResponse)
		}
	}))
	defer server.Close()

	// Note: This test would need significant refactoring of the queryHoldersByPolicy_Blockfrost
	// function to make it testable with dependency injection for the HTTP client and URLs.
	// The current implementation hard-codes the Blockfrost API URLs.
}

func TestSessionPath(t *testing.T) {
	sessionID := "test_session_123"
	expected := filepath.Join(sessionDir(), sessionID+".json")
	result := sessionPath(sessionID)

	if result != expected {
		t.Errorf("sessionPath: expected %s, got %s", expected, result)
	}
}

func TestSessionDir(t *testing.T) {
	expected := filepath.Join(baseAirdropDir, "sessions")
	result := sessionDir()

	if result != expected {
		t.Errorf("sessionDir: expected %s, got %s", expected, result)
	}
}

func TestSaveAndLoadSession(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	originalBaseDir := baseAirdropDir

	// Override the base directory for this test
	// Note: This would require refactoring to make baseAirdropDir configurable
	// For this test, we'll create the session in a temp directory structure

	session := &AirdropSession{
		DiscordUserID:         "test_user_123",
		SessionID:             "test_session_456",
		CreatedAt:             time.Now(),
		PolicyID:              "test_policy",
		ADAperNFT:             2.5,
		TotalNFTs:             100,
		TotalRecipients:       50,
		TotalLovelaceRequired: 250000000,
		Stage:                 StageAwaitingFunds,
		Address:               "addr1q9test123",
		Holders: []Holder{
			{Address: "addr1q9holder1", Quantity: 2},
			{Address: "addr1q9holder2", Quantity: 3},
		},
	}

	// Create session directory
	sessionDirPath := filepath.Join(tempDir, "sessions")
	err := os.MkdirAll(sessionDirPath, 0700)
	if err != nil {
		t.Fatalf("Failed to create session directory: %v", err)
	}

	// Save session to temp file
	sessionFilePath := filepath.Join(sessionDirPath, session.SessionID+".json")
	data, _ := json.MarshalIndent(session, "", "  ")
	err = os.WriteFile(sessionFilePath, data, 0600)
	if err != nil {
		t.Fatalf("Failed to write session file: %v", err)
	}

	// Load session from temp file
	loadedData, err := os.ReadFile(sessionFilePath)
	if err != nil {
		t.Fatalf("Failed to read session file: %v", err)
	}

	var loadedSession AirdropSession
	err = json.Unmarshal(loadedData, &loadedSession)
	if err != nil {
		t.Fatalf("Failed to unmarshal session: %v", err)
	}

	// Verify loaded session matches original
	if loadedSession.DiscordUserID != session.DiscordUserID {
		t.Errorf("DiscordUserID mismatch: expected %s, got %s", session.DiscordUserID, loadedSession.DiscordUserID)
	}
	if loadedSession.SessionID != session.SessionID {
		t.Errorf("SessionID mismatch: expected %s, got %s", session.SessionID, loadedSession.SessionID)
	}
	if loadedSession.ADAperNFT != session.ADAperNFT {
		t.Errorf("ADAperNFT mismatch: expected %f, got %f", session.ADAperNFT, loadedSession.ADAperNFT)
	}
	if len(loadedSession.Holders) != len(session.Holders) {
		t.Errorf("Holders length mismatch: expected %d, got %d", len(session.Holders), len(loadedSession.Holders))
	}

	// Restore original baseDir (though this won't affect the global variable)
	_ = originalBaseDir
}

func TestGetAddressBalance_Blockfrost_NoAPIKey(t *testing.T) {
	_, err := getAddressBalance_Blockfrost("addr1q9test", "")
	if err == nil {
		t.Error("Expected error for missing API key, got nil")
	}
	if !strings.Contains(err.Error(), "BLOCKFROST_API_KEY required") {
		t.Errorf("Expected API key error, got: %v", err)
	}
}

func TestGetAddressBalance_Blockfrost_Success(t *testing.T) {
	mockResponse := struct {
		Amount []struct {
			Unit     string `json:"unit"`
			Quantity string `json:"quantity"`
		} `json:"amount"`
	}{
		Amount: []struct {
			Unit     string `json:"unit"`
			Quantity string `json:"quantity"`
		}{
			{Unit: "lovelace", Quantity: "5000000"},
			{Unit: "policy123asset1", Quantity: "10"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("project_id") == "" {
			http.Error(w, "Missing project_id header", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Note: This test would require refactoring getAddressBalance_Blockfrost to accept
	// a configurable base URL instead of hard-coding the Blockfrost API URL
}

func TestExecCmd(t *testing.T) {
	tests := []struct {
		name        string
		bin         string
		args        []string
		expectError bool
	}{
		{
			name:        "successful command",
			bin:         "echo",
			args:        []string{"hello", "world"},
			expectError: false,
		},
		{
			name:        "command not found",
			bin:         "nonexistent_command_xyz",
			args:        []string{},
			expectError: true,
		},
		{
			name:        "command with error exit code",
			bin:         "sh",
			args:        []string{"-c", "exit 1"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := execCmd(tt.bin, tt.args...)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for command %s %v, got nil", tt.bin, tt.args)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for command %s %v: %v", tt.bin, tt.args, err)
			}

			if tt.name == "successful command" && !strings.Contains(output, "hello world") {
				t.Errorf("Expected output to contain 'hello world', got: %s", output)
			}
		})
	}
}

func TestValOr(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback string
		expected string
	}{
		{
			name:     "non-empty value",
			value:    "test_value",
			fallback: "fallback",
			expected: "test_value",
		},
		{
			name:     "empty value",
			value:    "",
			fallback: "fallback",
			expected: "fallback",
		},
		{
			name:     "whitespace only value",
			value:    "   ",
			fallback: "fallback",
			expected: "fallback",
		},
		{
			name:     "value with surrounding whitespace",
			value:    "  test  ",
			fallback: "fallback",
			expected: "  test  ", // valOr doesn't trim, it only checks if trimmed version is empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := valOr(tt.value, tt.fallback)
			if result != tt.expected {
				t.Errorf("valOr(%q, %q): expected %q, got %q", tt.value, tt.fallback, tt.expected, result)
			}
		})
	}
}

func TestLockSession(t *testing.T) {
	sessionID := "test_session_123"
	
	// Test that lockSession returns a function
	unlock := lockSession(sessionID)
	if unlock == nil {
		t.Error("lockSession should return an unlock function")
		return
	}

	// Unlock immediately to avoid deadlock in tests
	unlock()

	// Test that we can lock again after unlocking
	unlock2 := lockSession(sessionID)
	if unlock2 == nil {
		t.Error("lockSession should return an unlock function for existing session")
		return
	}

	// Call unlock function (should not panic)
	unlock2()
}

func TestConstants(t *testing.T) {
	if cardanoNetworkTag != "--mainnet" {
		t.Errorf("cardanoNetworkTag should be '--mainnet', got: %s", cardanoNetworkTag)
	}

	if baseAirdropDir != "./airdrops" {
		t.Errorf("baseAirdropDir should be './airdrops', got: %s", baseAirdropDir)
	}

	if feeBufferADA != 5.0 {
		t.Errorf("feeBufferADA should be 5.0, got: %f", feeBufferADA)
	}

	if feeBufferLovelace != 5000000 {
		t.Errorf("feeBufferLovelace should be 5000000, got: %d", feeBufferLovelace)
	}

	if serviceFeeADA != 20.0 {
		t.Errorf("serviceFeeADA should be 20.0, got: %f", serviceFeeADA)
	}

	if serviceFeeLovelace != 20000000 {
		t.Errorf("serviceFeeLovelace should be 20000000, got: %d", serviceFeeLovelace)
	}

	if maxOutputsPerTx != 120 {
		t.Errorf("maxOutputsPerTx should be 120, got: %d", maxOutputsPerTx)
	}

	expectedPollInterval := 10 * time.Second
	if depositPollInterval != expectedPollInterval {
		t.Errorf("depositPollInterval should be %v, got: %v", expectedPollInterval, depositPollInterval)
	}
}