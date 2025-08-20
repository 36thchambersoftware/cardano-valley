package discord

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAirdropCrashRecovery tests what happens when the bot crashes at various stages of airdrop execution
func TestAirdropCrashRecovery(t *testing.T) {
	tempDir := t.TempDir()
	
	crashRecoveryTests := []struct {
		name                    string
		crashStage             AirdropStage
		scenario               string
		sessionSetup           func() *AirdropSession
		expectRecoverable      bool
		expectedRecoveryAction string
		riskLevel              string
	}{
		{
			name:       "Crash During Fund Waiting",
			crashStage: StageAwaitingFunds,
			scenario:   "Bot crashes while waiting for user to deposit funds",
			sessionSetup: func() *AirdropSession {
				return createTestSession("user_123", "waiting_funds_session", StageAwaitingFunds)
			},
			expectRecoverable:      true,
			expectedRecoveryAction: "Resume waiting for funds deposit",
			riskLevel:              "LOW - No funds at risk, can safely resume",
		},
		{
			name:       "Crash During Transaction Building", 
			crashStage: StageBuildingTx,
			scenario:   "Bot crashes while building transaction but before submission",
			sessionSetup: func() *AirdropSession {
				session := createTestSession("user_456", "building_tx_session", StageBuildingTx)
				session.TotalLovelaceRequired = 50000000 // 50 ADA deposited
				return session
			},
			expectRecoverable:      true,
			expectedRecoveryAction: "Resume transaction building from scratch",
			riskLevel:              "MEDIUM - Funds deposited but no TX submitted yet",
		},
		{
			name:       "Crash During Distribution",
			crashStage: StageDistributing,
			scenario:   "Bot crashes after submitting some transactions but not all",
			sessionSetup: func() *AirdropSession {
				session := createTestSession("user_789", "distributing_session", StageDistributing) 
				session.DistributionTxIDs = []string{
					"tx_123_submitted_successfully",
					"tx_456_submitted_successfully",
				}
				// Simulate partial distribution - some TXs submitted, others not
				return session
			},
			expectRecoverable:      false, // Current system doesn't handle partial recovery
			expectedRecoveryAction: "MANUAL INTERVENTION REQUIRED - Check which TXs succeeded",
			riskLevel:              "HIGH - Partial distribution, some users may have received rewards",
		},
		{
			name:       "Crash During Service Fee Payment",
			crashStage: StagePayingFee,
			scenario:   "Bot crashes while paying service fee after successful distribution",
			sessionSetup: func() *AirdropSession {
				session := createTestSession("user_101", "paying_fee_session", StagePayingFee)
				session.DistributionTxIDs = []string{
					"tx_123_distribution_complete",
					"tx_456_distribution_complete", 
					"tx_789_distribution_complete",
				}
				// All distribution complete, now paying fee
				return session
			},
			expectRecoverable:      true,
			expectedRecoveryAction: "Resume service fee payment and wallet draining",
			riskLevel:              "LOW - Distribution complete, only fee payment remaining",
		},
		{
			name:       "Crash After Completion",
			crashStage: StageCompleted,
			scenario:   "Bot crashes after marking airdrop complete but before cleanup",
			sessionSetup: func() *AirdropSession {
				session := createTestSession("user_202", "completed_session", StageCompleted)
				session.DistributionTxIDs = []string{"tx_123_done"}
				session.ServiceFeeTxID = "fee_tx_789_done"
				return session
			},
			expectRecoverable:      true,
			expectedRecoveryAction: "Airdrop already complete, send notification if needed",
			riskLevel:              "NONE - Airdrop fully complete",
		},
		{
			name:       "Crash With Corrupted Session",
			crashStage: StageAwaitingFunds,
			scenario:   "Bot crashes and session file becomes corrupted",
			sessionSetup: func() *AirdropSession {
				// This will be intentionally corrupted in the test
				return createTestSession("user_303", "corrupted_session", StageAwaitingFunds)
			},
			expectRecoverable:      false,
			expectedRecoveryAction: "ERROR - Session unrecoverable, manual investigation needed",
			riskLevel:              "VARIABLE - Depends on what stage was lost",
		},
	}

	for _, tt := range crashRecoveryTests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup session
			session := tt.sessionSetup()
			sessionDir := filepath.Join(tempDir, "sessions")
			os.MkdirAll(sessionDir, 0700)
			
			sessionPath := filepath.Join(sessionDir, session.SessionID+".json")
			
			if tt.name == "Crash With Corrupted Session" {
				// Intentionally create corrupted session file
				err := os.WriteFile(sessionPath, []byte("invalid json {{{"), 0600)
				if err != nil {
					t.Fatalf("Failed to create corrupted session: %v", err)
				}
			} else {
				// Save valid session
				data, _ := json.MarshalIndent(session, "", "  ")
				err := os.WriteFile(sessionPath, data, 0600)
				if err != nil {
					t.Fatalf("Failed to save session: %v", err)
				}
			}

			// Simulate bot restart by trying to load and analyze session
			recoveryResult := analyzeSessionForRecovery(sessionPath, tt.crashStage)
			
			// Validate recovery analysis
			if recoveryResult.IsRecoverable != tt.expectRecoverable {
				t.Errorf("Recovery expectation mismatch: expected %v, got %v", 
					tt.expectRecoverable, recoveryResult.IsRecoverable)
			}

			// Log detailed scenario analysis
			t.Logf("=== Crash Recovery Test: %s ===", tt.name)
			t.Logf("Scenario: %s", tt.scenario)
			t.Logf("Crash Stage: %s", tt.crashStage)
			t.Logf("Risk Level: %s", tt.riskLevel)
			t.Logf("Recoverable: %v", recoveryResult.IsRecoverable)
			t.Logf("Recommended Action: %s", recoveryResult.RecommendedAction)
			t.Logf("Analysis: %s", recoveryResult.Analysis)
			t.Logf("Session Data: %+v", recoveryResult.SessionSummary)
			t.Logf("==========================================")

			// Verify critical information is captured
			if recoveryResult.SessionSummary.UserID == "" && tt.expectRecoverable {
				t.Error("Should capture user ID for recoverable sessions")
			}

			if tt.crashStage == StageDistributing && len(recoveryResult.SessionSummary.TxIDs) == 0 {
				t.Error("Should capture submitted transaction IDs for distribution stage")
			}
		})
	}
}

// RecoveryAnalysis represents the analysis of a crashed session
type RecoveryAnalysis struct {
	IsRecoverable       bool
	RecommendedAction  string
	Analysis           string
	RiskAssessment     string
	SessionSummary     SessionSummary
}

// SessionSummary contains key info needed for recovery decisions
type SessionSummary struct {
	SessionID    string
	UserID       string
	Stage        AirdropStage
	Address      string
	RequiredADA  float64
	Recipients   int
	TxIDs        []string
	ServiceTxID  string
	LastError    string
}

// analyzeSessionForRecovery simulates what should happen when bot restarts
func analyzeSessionForRecovery(sessionPath string, expectedStage AirdropStage) RecoveryAnalysis {
	result := RecoveryAnalysis{
		IsRecoverable: false,
		RecommendedAction: "UNKNOWN",
		Analysis: "Failed to analyze session",
	}

	// Try to load session
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		result.Analysis = "Session file not found or unreadable"
		result.RecommendedAction = "ERROR - Session file missing"
		return result
	}

	var session AirdropSession
	err = json.Unmarshal(data, &session)
	if err != nil {
		result.Analysis = "Session file corrupted or invalid JSON"
		result.RecommendedAction = "ERROR - Session unrecoverable, manual investigation needed"
		return result
	}

	// Build session summary
	result.SessionSummary = SessionSummary{
		SessionID:    session.SessionID,
		UserID:       session.DiscordUserID,
		Stage:        session.Stage,
		Address:      session.Address,
		RequiredADA:  float64(session.TotalLovelaceRequired) / 1_000_000,
		Recipients:   len(session.Holders),
		TxIDs:        session.DistributionTxIDs,
		ServiceTxID:  session.ServiceFeeTxID,
		LastError:    session.LastError,
	}

	// Analyze based on stage
	switch session.Stage {
	case StageAwaitingFunds:
		result.IsRecoverable = true
		result.RecommendedAction = "Resume waiting for funds deposit"
		result.Analysis = "Safe to resume - no funds at risk, can continue waiting for deposit"
		result.RiskAssessment = "LOW"

	case StageBuildingTx:
		result.IsRecoverable = true
		result.RecommendedAction = "Resume transaction building from scratch"
		result.Analysis = "Funds deposited but no transactions submitted yet - can safely rebuild"
		result.RiskAssessment = "MEDIUM"

	case StageDistributing:
		if len(session.DistributionTxIDs) > 0 {
			result.IsRecoverable = false
			result.RecommendedAction = "MANUAL INTERVENTION REQUIRED - Check which TXs succeeded"
			result.Analysis = "Partial distribution detected - need to verify which transactions succeeded before proceeding"
			result.RiskAssessment = "HIGH"
		} else {
			result.IsRecoverable = true
			result.RecommendedAction = "Resume distribution from beginning"
			result.Analysis = "Distribution stage but no TXs submitted yet"
			result.RiskAssessment = "MEDIUM"
		}

	case StagePayingFee:
		result.IsRecoverable = true
		result.RecommendedAction = "Resume service fee payment and wallet draining"
		result.Analysis = "Distribution complete, only fee payment remaining"
		result.RiskAssessment = "LOW"

	case StageCompleted:
		result.IsRecoverable = true
		result.RecommendedAction = "Airdrop already complete, send notification if needed"
		result.Analysis = "Airdrop fully complete, may just need cleanup/notification"
		result.RiskAssessment = "NONE"

	case StageCancelled:
		result.IsRecoverable = false
		result.RecommendedAction = "Airdrop was cancelled, investigate reason"
		result.Analysis = "Session was previously cancelled"
		result.RiskAssessment = "VARIABLE"

	default:
		result.RecommendedAction = "UNKNOWN STAGE - Manual investigation required"
		result.Analysis = "Session in unknown state"
		result.RiskAssessment = "UNKNOWN"
	}

	return result
}

// createTestSession creates a test session for crash testing
func createTestSession(userID, sessionID string, stage AirdropStage) *AirdropSession {
	return &AirdropSession{
		DiscordUserID:         userID,
		SessionID:             sessionID,
		CreatedAt:             time.Now(),
		PolicyID:              "d5e6bf0500378d4f0da4e8dde6becec7621cd8cbf5cbb9b87013d4cc",
		ADAperNFT:             2.0,
		TotalNFTs:             25,
		TotalRecipients:       10,
		TotalLovelaceRequired: 55000000, // 50 ADA + 5 buffer
		Stage:                 stage,
		Address:               "addr1q9test_crash_recovery_address_123456789",
		Holders: []Holder{
			{Address: "addr1q9holder1", Quantity: 10},
			{Address: "addr1q9holder2", Quantity: 15},
		},
	}
}

func TestSessionRecoveryAfterRestart(t *testing.T) {
	tempDir := t.TempDir()
	
	// Simulate multiple sessions at different stages when bot crashes
	sessions := []*AirdropSession{
		createTestSession("user_1", "session_waiting", StageAwaitingFunds),
		createTestSession("user_2", "session_building", StageBuildingTx), 
		createTestSession("user_3", "session_distributing", StageDistributing),
		createTestSession("user_4", "session_paying", StagePayingFee),
		createTestSession("user_5", "session_complete", StageCompleted),
	}

	// Add some transaction IDs to the distributing session
	sessions[2].DistributionTxIDs = []string{"tx_123", "tx_456"}
	sessions[3].DistributionTxIDs = []string{"tx_789", "tx_abc"}
	sessions[3].ServiceFeeTxID = "fee_tx_def"

	sessionDir := filepath.Join(tempDir, "sessions")
	os.MkdirAll(sessionDir, 0700)

	// Save all sessions
	for _, session := range sessions {
		sessionPath := filepath.Join(sessionDir, session.SessionID+".json")
		data, _ := json.MarshalIndent(session, "", "  ")
		os.WriteFile(sessionPath, data, 0600)
	}

	// Simulate bot restart - discover and analyze all sessions
	recoveryResults := simulateBotRestartRecovery(sessionDir)

	// Validate recovery analysis
	if len(recoveryResults) != len(sessions) {
		t.Errorf("Expected to find %d sessions, found %d", len(sessions), len(recoveryResults))
	}

	var recoverableSessions, manualInterventionSessions, completedSessions int

	for _, result := range recoveryResults {
		switch {
		case result.IsRecoverable && result.SessionSummary.Stage != StageCompleted:
			recoverableSessions++
		case !result.IsRecoverable && strings.Contains(result.RecommendedAction, "MANUAL"):
			manualInterventionSessions++
		case result.SessionSummary.Stage == StageCompleted:
			completedSessions++
		}

		t.Logf("Session %s (Stage: %s) - Recoverable: %v, Action: %s",
			result.SessionSummary.SessionID,
			result.SessionSummary.Stage,
			result.IsRecoverable,
			result.RecommendedAction)
	}

	// Verify expected categorization
	expectedRecoverable := 3      // waiting, building, paying_fee
	expectedManual := 1          // distributing (has partial TXs)
	expectedCompleted := 1       // completed

	if recoverableSessions != expectedRecoverable {
		t.Errorf("Expected %d recoverable sessions, got %d", expectedRecoverable, recoverableSessions)
	}

	if manualInterventionSessions != expectedManual {
		t.Errorf("Expected %d manual intervention sessions, got %d", expectedManual, manualInterventionSessions)
	}

	if completedSessions != expectedCompleted {
		t.Errorf("Expected %d completed sessions, got %d", expectedCompleted, completedSessions)
	}

	t.Logf("Recovery Summary:")
	t.Logf("  - Automatically recoverable: %d sessions", recoverableSessions)
	t.Logf("  - Require manual intervention: %d sessions", manualInterventionSessions)
	t.Logf("  - Already completed: %d sessions", completedSessions)
}

// simulateBotRestartRecovery simulates what should happen when the bot restarts
func simulateBotRestartRecovery(sessionDir string) []RecoveryAnalysis {
	var results []RecoveryAnalysis

	// Find all session files
	files, err := filepath.Glob(filepath.Join(sessionDir, "*.json"))
	if err != nil {
		return results
	}

	// Analyze each session
	for _, file := range files {
		// Load session to determine expected stage
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var session AirdropSession
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}

		// Analyze for recovery
		result := analyzeSessionForRecovery(file, session.Stage)
		results = append(results, result)
	}

	return results
}

func TestConcurrentSessionRecovery(t *testing.T) {
	tempDir := t.TempDir()
	sessionDir := filepath.Join(tempDir, "sessions")
	os.MkdirAll(sessionDir, 0700)

	// Create multiple sessions that could be running concurrently
	concurrentSessions := []*AirdropSession{
		createTestSession("user_100", "concurrent_1", StageAwaitingFunds),
		createTestSession("user_200", "concurrent_2", StageAwaitingFunds),
		createTestSession("user_300", "concurrent_3", StageBuildingTx),
		createTestSession("user_400", "concurrent_4", StageDistributing),
	}

	// Set up different scenarios for each
	concurrentSessions[1].TotalLovelaceRequired = 100000000 // 100 ADA - large airdrop
	concurrentSessions[2].LastError = "network timeout during tx build"
	concurrentSessions[3].DistributionTxIDs = []string{"tx_partial_1"} // Partial completion

	// Save all sessions
	for _, session := range concurrentSessions {
		sessionPath := filepath.Join(sessionDir, session.SessionID+".json")
		data, _ := json.MarshalIndent(session, "", "  ")
		os.WriteFile(sessionPath, data, 0600)
	}

	// Test concurrent recovery (simulating multiple goroutines trying to recover)
	recoveryResults := make(chan RecoveryAnalysis, len(concurrentSessions))

	for _, session := range concurrentSessions {
		go func(sessionID string) {
			sessionPath := filepath.Join(sessionDir, sessionID+".json")
			result := analyzeSessionForRecovery(sessionPath, StageAwaitingFunds) // We'll determine actual stage
			recoveryResults <- result
		}(session.SessionID)
	}

	// Collect results
	var results []RecoveryAnalysis
	for i := 0; i < len(concurrentSessions); i++ {
		result := <-recoveryResults
		results = append(results, result)
	}

	// Validate that all sessions were processed
	if len(results) != len(concurrentSessions) {
		t.Errorf("Expected %d results, got %d", len(concurrentSessions), len(results))
	}

	// Verify each session has proper recovery analysis
	sessionMap := make(map[string]RecoveryAnalysis)
	for _, result := range results {
		sessionMap[result.SessionSummary.SessionID] = result
	}

	// Check specific recovery scenarios
	if result, exists := sessionMap["concurrent_3"]; exists {
		if !strings.Contains(result.Analysis, "building") && result.SessionSummary.LastError == "" {
			t.Error("Should capture last error for failed transaction building")
		}
	}

	if result, exists := sessionMap["concurrent_4"]; exists {
		if result.IsRecoverable {
			t.Error("Partially distributed session should not be automatically recoverable")
		}
	}

	t.Logf("Concurrent recovery test completed successfully")
	t.Logf("Processed %d sessions concurrently", len(results))

	// Log summary of recovery statuses
	for _, result := range results {
		t.Logf("Session %s: %s -> %s",
			result.SessionSummary.SessionID,
			result.SessionSummary.Stage,
			result.RecommendedAction)
	}
}

func TestPartialTransactionRecovery(t *testing.T) {
	// Test scenarios where some transactions succeeded but others failed
	partialScenarios := []struct {
		name            string
		submittedTxs    []string
		expectedTotalTxs int
		scenario        string
	}{
		{
			name:            "First TX Batch Succeeded",
			submittedTxs:    []string{"tx_batch_1_success"},
			expectedTotalTxs: 3,
			scenario:        "Only 1 of 3 transaction batches was submitted before crash",
		},
		{
			name:            "Multiple Batches Succeeded", 
			submittedTxs:    []string{"tx_batch_1_success", "tx_batch_2_success"},
			expectedTotalTxs: 4,
			scenario:        "2 of 4 transaction batches were submitted before crash",
		},
		{
			name:            "All But Last Batch Succeeded",
			submittedTxs:    []string{"tx_1", "tx_2", "tx_3", "tx_4"},
			expectedTotalTxs: 5,
			scenario:        "4 of 5 batches succeeded, only service fee payment remaining",
		},
	}

	for _, scenario := range partialScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			session := createTestSession("user_partial", scenario.name, StageDistributing)
			session.DistributionTxIDs = scenario.submittedTxs

			tempDir := t.TempDir()
			sessionPath := filepath.Join(tempDir, session.SessionID+".json")
			
			data, _ := json.MarshalIndent(session, "", "  ")
			os.WriteFile(sessionPath, data, 0600)

			result := analyzeSessionForRecovery(sessionPath, StageDistributing)

			t.Logf("=== Partial TX Recovery Test: %s ===", scenario.name)
			t.Logf("Scenario: %s", scenario.scenario)
			t.Logf("Submitted TXs: %d", len(scenario.submittedTxs))
			t.Logf("Expected Total TXs: %d", scenario.expectedTotalTxs)
			t.Logf("Recovery Action: %s", result.RecommendedAction)
			t.Logf("Risk Assessment: %s", result.RiskAssessment)

			// Verify that partial transactions are detected
			if len(scenario.submittedTxs) > 0 && result.IsRecoverable {
				t.Error("Sessions with submitted transactions should require manual intervention")
			}

			if !strings.Contains(result.RecommendedAction, "MANUAL") {
				t.Error("Partial transaction scenarios should require manual intervention")
			}

			if result.RiskAssessment != "HIGH" {
				t.Error("Partial transactions should be assessed as HIGH risk")
			}
		})
	}
}