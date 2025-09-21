package discord

import (
	"bytes"
	"cardano-valley/pkg/blockfrost"
	"cardano-valley/pkg/logger"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

//
// ────────────────────────────────────────────────────────────────────────────────
//  CONFIG
// ────────────────────────────────────────────────────────────────────────────────
//

const (
	// Change to "--mainnet" if you're on mainnet
	CardanoNetworkTag = "--mainnet"
	// For pre-prod/testnet use: "--testnet-magic", "1xxxxxx"
	// cardanoNetworkTag = "--testnet-magic"
	// testnetMagicValue = "1097911063"

	// Where to store temp wallets/sessions
	baseAirdropDir = "./airdrops"

	// Buffer to make sure we cover network fees comfortably
	feeBufferADA       = 5.0
	feeBufferLovelace  = uint64(feeBufferADA * 1_000_000)

	// Flat service fee in ADA (separate tx AFTER the airdrop)
	serviceFeeADA      = 20.0
	serviceFeeLovelace = uint64(serviceFeeADA * 1_000_000)

	// Safety: outputs per TX (tune for your environment; 80–120 is common)
	maxOutputsPerTx = 120

	// How often to poll for deposit
	depositPollInterval = 1 * time.Minute
)

// Required ENV:
//   BLOCKFROST_API_KEY: string
//   CARDANO_VALLEY_ADDRESS:    cardano addr for the 20 ADA fee
// Optional:
//   AIRDROP_PUBLIC_CHANNEL_ID: to post the announcement embed

func getEnv(key string) string {
	v := os.Getenv(key)
	return v
}

//
// ────────────────────────────────────────────────────────────────────────────────
//  TYPES
// ────────────────────────────────────────────────────────────────────────────────
//

type Holder struct {
	Address  string `json:"address"`
	Quantity uint64    `json:"quantity"`
}

type AirdropStage string

const (
	StageAwaitingFunds AirdropStage = "awaiting_funds"
	StageBuildingTx    AirdropStage = "building_tx"
	StageDistributing  AirdropStage = "distributing"
	StagePayingFee     AirdropStage = "paying_service_fee"
	StageCompleted     AirdropStage = "completed"
	StageCancelled     AirdropStage = "cancelled"
)

type AirdropSession struct {
	DiscordUserID string        `json:"discord_user_id"`
	SessionID     string        `json:"session_id"`
	CreatedAt     time.Time     `json:"created_at"`

	// input config
	PolicyID       string   `json:"policy_id,omitempty"`
	HoldersPath    string   `json:"holders_path,omitempty"` // JSON file path (if uploaded)
	ADAperNFT      float64  `json:"ada_per_nft"`
	Holders        []Holder `json:"holders"`

	// computed
	TotalNFTs              uint64      `json:"total_nfts"`
	TotalRecipients        uint64      `json:"total_recipients"`
	TotalLovelaceRequired  uint64    `json:"total_lovelace_required"` // includes 5 ADA buffer
	DistributionTxIDs      []string `json:"distribution_tx_ids"`
	ServiceFeeTxID         string   `json:"service_fee_tx_id"`
	AnnouncementMessageURL string   `json:"announcement_message_url"`

	// wallet
	WalletDir string `json:"wallet_dir"`
	AddrFile  string `json:"addr_file"`
	VKeyFile  string `json:"vkey_file"`
	SKeyFile  string `json:"skey_file"`
	Address   string `json:"address"`

	// lifecycle
	Stage AirdropStage `json:"stage"`

	// bookkeeping
	LastError string `json:"last_error,omitempty"`
}

type out struct {
	Addr     string
	Lovelace int64
}

type UTxOMap map[string]struct {
		TxHash string `json:"tx_hash"`
		TxIx   int    `json:"tx_index"`
	}

// in-memory locker so concurrent workers don't trample the same session
var sessionLocks sync.Map // map[sessionID]*sync.Mutex

func lockSession(id string) func() {
	muAny, _ := sessionLocks.LoadOrStore(id, &sync.Mutex{})
	mu := muAny.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

//
// ────────────────────────────────────────────────────────────────────────────────
//  HOLDERS: Load from file or Blockfrost policy lookup
// ────────────────────────────────────────────────────────────────────────────────
//

func loadHoldersFromAttachment(url string) ([]Holder, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var holders []Holder
	if err := json.Unmarshal(body, &holders); err != nil {
		return nil, err
	}
	return holders, nil
}

// Strategy:
// 1) list assets under policy
// 2) for each asset, fetch current addresses/qty (should be 1 with qty=1 for NFTs)
// 3) accumulate by address (quantity == number of NFTs)
func queryHoldersByPolicy_Blockfrost(policyID, apiKey string) ([]Holder, error) {
	if apiKey == "" {
		return nil, errors.New("BLOCKFROST_API_KEY is required")
	}
	type asset struct {
		Asset string `json:"asset"` // policy + hex asset name
	}
	assets := []asset{}

	// paginate /assets/policy/{policy_id}
	page := 1
	for {
		req, _ := http.NewRequest("GET",
			fmt.Sprintf("https://cardano-mainnet.blockfrost.io/api/v0/assets/policy/%s?page=%d", policyID, page),
			nil,
		)
		req.Header.Set("project_id", apiKey)
		rsp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer rsp.Body.Close()
		if rsp.StatusCode == 404 {
			break
		}
		if rsp.StatusCode >= 300 {
			b, _ := io.ReadAll(rsp.Body)
			return nil, fmt.Errorf("blockfrost policy assets: %s", string(b))
		}
		var pageAssets []asset
		if err := json.NewDecoder(rsp.Body).Decode(&pageAssets); err != nil {
			return nil, err
		}
		if len(pageAssets) == 0 {
			break
		}
		assets = append(assets, pageAssets...)
		page++
	}

	// Collect holders count
	counts := map[string]int{}
	for _, a := range assets {
		// /assets/{asset}/addresses
		req, _ := http.NewRequest("GET",
			fmt.Sprintf("https://cardano-mainnet.blockfrost.io/api/v0/assets/%s/addresses", a.Asset),
			nil,
		)
		req.Header.Set("project_id", apiKey)
		rsp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		if rsp.StatusCode >= 300 {
			b, _ := io.ReadAll(rsp.Body)
			_ = rsp.Body.Close()
			return nil, fmt.Errorf("blockfrost asset addresses: %s", string(b))
		}
		var addrs []struct {
			Address  string `json:"address"`
			Quantity string `json:"quantity"` // string integer
		}
		if err := json.NewDecoder(rsp.Body).Decode(&addrs); err != nil {
			_ = rsp.Body.Close()
			return nil, err
		}
		_ = rsp.Body.Close()
		for _, rec := range addrs {
			qty, _ := strconv.ParseInt(rec.Quantity, 10, 64)
			if qty > 0 {
				counts[rec.Address] += int(qty)
			}
		}
	}

	holders := make([]Holder, 0, len(counts))
	for addr, qty := range counts {
		holders = append(holders, Holder{Address: addr, Quantity: uint64(qty)})
	}
	// deterministic order
	sort.Slice(holders, func(i, j int) bool { return holders[i].Address < holders[j].Address })
	return holders, nil
}

//
// ────────────────────────────────────────────────────────────────────────────────
//  SESSION PERSISTENCE (JSON files; simple, robust)
// ────────────────────────────────────────────────────────────────────────────────
//

func sessionDir() string { return filepath.Join(baseAirdropDir, "sessions") }

func sessionPath(sessionID string) string {
	return filepath.Join(sessionDir(), sessionID+".json")
}

func saveSession(ses *AirdropSession) error {
	if err := os.MkdirAll(sessionDir(), 0700); err != nil {
		return err
	}
	tmp := sessionPath(ses.SessionID) + ".tmp"
	data, _ := json.MarshalIndent(ses, "", "  ")
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, sessionPath(ses.SessionID))
}

func loadSession(sessionID string) (*AirdropSession, error) {
	b, err := os.ReadFile(sessionPath(sessionID))
	if err != nil {
		return nil, err
	}
	var s AirdropSession
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

//
// ────────────────────────────────────────────────────────────────────────────────
//  TEMP WALLET CREATION (names include Discord ID; retained for 30 days)
// ────────────────────────────────────────────────────────────────────────────────
//

func createTempWallet(discordID string) (*AirdropSession, error) {
	now := time.Now()
	sessionID := fmt.Sprintf("%s_%d", discordID, now.Unix())
	dir := filepath.Join(baseAirdropDir, "active", sessionID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	vkey := filepath.Join(dir, fmt.Sprintf("airdrop_%s_%d.vkey", discordID, now.Unix()))
	skey := filepath.Join(dir, fmt.Sprintf("airdrop_%s_%d.skey", discordID, now.Unix()))
	addr := filepath.Join(dir, fmt.Sprintf("airdrop_%s_%d.addr", discordID, now.Unix()))

	// Generate keys
	args := []string{"address", "key-gen", "--verification-key-file", vkey, "--signing-key-file", skey}
	if out, err := execCmd("cardano-cli", args...); err != nil {
		return nil, fmt.Errorf("key-gen: %v (%s)", err, out)
	}

	// Build address
	buildArgs := []string{"address", "build", "--payment-verification-key-file", vkey, "--out-file", addr, CardanoNetworkTag}
	if out, err := execCmd("cardano-cli", buildArgs...); err != nil {
		return nil, fmt.Errorf("address build: %v (%s)", err, out)
	}

	// Load address text
	ab, _ := os.ReadFile(addr)
	address := strings.TrimSpace(string(ab))

	return &AirdropSession{
		DiscordUserID: discordID,
		SessionID:     sessionID,
		CreatedAt:     now,
		WalletDir:     dir,
		VKeyFile:      vkey,
		SKeyFile:      skey,
		AddrFile:      addr,
		Address:       address,
		Stage:         StageAwaitingFunds,
	}, nil
}

//
// ────────────────────────────────────────────────────────────────────────────────
//  WATCHER: Wait for deposit → distribute → pay service fee → announce
// ────────────────────────────────────────────────────────────────────────────────
//

func watchAndRunAirdrop(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID string) {
	unlock := lockSession(sessionID)
	defer unlock()

	ses, err := loadSession(sessionID)
	if err != nil {
		// can't report; log to stdout
		fmt.Println("watch: loadSession:", err)
		return
	}

	// 1) Wait for deposit
	ses.Stage = StageAwaitingFunds
	_ = saveSession(ses)

	required := ses.TotalLovelaceRequired
	var have uint64
	ctx := context.Background()
	for {
		time.Sleep(depositPollInterval)
		have, err = blockfrost.GetAddressBalance(ctx, ses.Address)
		if err != nil {
			ses.LastError = "balance check: " + err.Error()
			_ = saveSession(ses)
			continue
		}
		if have >= required {
			break
		}
	}

	// 2) Build distribution TXs
	ses.Stage = StageBuildingTx
	_ = saveSession(ses)

	txids, err := buildSignSubmitAirdropTxs(ses)
	if err != nil {
		ses.LastError = "distribution failed: " + err.Error()
		_ = saveSession(ses)
		sendDM(s, ses.DiscordUserID, fmt.Sprintf("❌ Airdrop failed while building/submitting TXs: %v", err))
		return
	}
	ses.DistributionTxIDs = txids
	ses.Stage = StageDistributing
	_ = saveSession(ses)

	// 3) Pay 20 ADA service fee, and drain any leftover
	ses.Stage = StagePayingFee
	_ = saveSession(ses)

	if err := payServiceFeeAndDrain(ses); err != nil {
		ses.LastError = "service fee failed: " + err.Error()
		_ = saveSession(ses)
		sendDM(s, ses.DiscordUserID, fmt.Sprintf("⚠️ Airdrop sent, but fee/drain step had an issue: %v. You may need to top up or handle leftovers manually.", err))
		// continue to announcement anyway
	}

	// 4) Mark complete
	ses.Stage = StageCompleted
	_ = saveSession(ses)

	// 5) DM receipt
	var buf strings.Builder
	fmt.Fprintf(&buf, "🎉 **Airdrop Complete!**\n\n")
	fmt.Fprintf(&buf, "- Recipients: %d\n", ses.TotalRecipients)
	fmt.Fprintf(&buf, "- Total NFTs: %d\n", ses.TotalNFTs)
	fmt.Fprintf(&buf, "- ADA/NFT: %.6f\n", ses.ADAperNFT)
	fmt.Fprintf(&buf, "- Distribution TXs:\n")
	for _, id := range ses.DistributionTxIDs {
		fmt.Fprintf(&buf, "  • %s\n", id)
	}
	if ses.ServiceFeeTxID != "" {
		fmt.Fprintf(&buf, "- Service Fee TX: %s\n", ses.ServiceFeeTxID)
	}
	sendDM(s, ses.DiscordUserID, buf.String())

	// 6) Public announcement
	publicChan := getEnv("AIRDROP_PUBLIC_CHANNEL_ID")
	if publicChan != "" {
		embed := &discordgo.MessageEmbed{
			Title:       "🌾 Cardano Valley Airdrop Complete",
			Description: fmt.Sprintf("Distributed to **%d** wallets across **%d** NFTs.", ses.TotalRecipients, ses.TotalNFTs),
			Color:       0xF59E0B,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "ADA/NFT", Value: fmt.Sprintf("%.6f", ses.ADAperNFT), Inline: true},
				{Name: "TX Count", Value: fmt.Sprintf("%d", len(ses.DistributionTxIDs)), Inline: true},
			},
			Footer: &discordgo.MessageEmbedFooter{Text: "Cardano Valley • PREEB"},
		}
		msg, _ := s.ChannelMessageSendEmbed(publicChan, embed)
		if msg != nil {
			ses.AnnouncementMessageURL = fmt.Sprintf("https://discord.com/channels/%s/%s/%s", msg.GuildID, msg.ChannelID, msg.ID)
			_ = saveSession(ses)
		}
	}
}

//
// ────────────────────────────────────────────────────────────────────────────────
//  CARDANO CHAIN HELPERS (Blockfrost + cardano-cli)
// ────────────────────────────────────────────────────────────────────────────────
//

func buildSignSubmitAirdropTxs(ses *AirdropSession) ([]string, error) {
	// Chunk outputs into batches
	var outputs []out
	for _, h := range ses.Holders {
		amt := int64(math.Round(float64(h.Quantity) * ses.ADAperNFT * 1_000_000))
		if amt > 0 {
			outputs = append(outputs, out{Addr: h.Address, Lovelace: amt})
		}
	}
	// Split into batches
	var batches [][]out
	for i := 0; i < len(outputs); i += maxOutputsPerTx {
		j := i + maxOutputsPerTx
		if j > len(outputs) {
			j = len(outputs)
		}
		batches = append(batches, outputs[i:j])
	}

	var txIDs []string
	for _, batch := range batches {
		txid, err := buildSignSubmitSingleTx(ses, batch)
		if err != nil {
			return nil, err
		}
		txIDs = append(txIDs, txid)
	}
	return txIDs, nil
}

// Build a single transaction with multiple --tx-out outputs and change back to the same airdrop address.
func buildSignSubmitSingleTx(ses *AirdropSession, batch []out) (string, error) {

	txBody := filepath.Join(ses.WalletDir, fmt.Sprintf("txbody_%d.raw", time.Now().UnixNano()))
	txSigned := filepath.Join(ses.WalletDir, fmt.Sprintf("txsigned_%d.signed", time.Now().UnixNano()))
	socketPath := os.Getenv("CARDANO_NODE_SOCKET_PATH")

	// Gather tx-out args
	var outArgs []string
	var totalBatch int64
	for _, o := range batch {
		outArgs = append(outArgs, "--tx-out", fmt.Sprintf("%s+%d", o.Addr, o.Lovelace))
		totalBatch += o.Lovelace
	}

	// We have to have a tx-in
	out, err := execCmd("cardano-cli", "query", "utxo",
		"--address", ses.Address,
		CardanoNetworkTag,
		"--socket-path", socketPath,
		"--out-file", "/dev/stdout",
		"--output-json",
	)
	if err != nil {
		return "", fmt.Errorf("failed to query UTXOs: %w", err)
	}

	// Parse JSON into map
	var utxos UTxOMap
	if err := json.Unmarshal([]byte(out), &utxos); err != nil {
		return "", fmt.Errorf("failed to parse UTXO JSON: %w", err)
	}

	txIns := []string{}
	for utxo := range utxos {
		// key is like "txhash#txix"
		txIns = append(txIns, "--tx-in", utxo)
	}
	if len(txIns) == 0 {
		return "", fmt.Errorf("no UTXOs found at address %s", ses.Address)
	}

	// Build (letting cardano-cli calculate fee and change)
	args := []string{"conway", "transaction", "build",
		"--change-address", ses.Address,
		CardanoNetworkTag,
		"--socket-path", socketPath,
		"--out-file", txBody,
		// "--metadata-json-file", "/home/cardano/cardano-valley-metadata.json",
	}
	args = append(args, txIns...)
	args = append(args, outArgs...)

	logger.Record.Info("building tx", "ARGS", args)

	if out, err := execCmd("cardano-cli", args...); err != nil {
		return "", fmt.Errorf("tx build: %v (%s)", err, out)
	}

	// Sign
	signArgs := []string{"conway", "transaction", "sign",
		"--tx-body-file", txBody,
		"--signing-key-file", ses.SKeyFile,
		CardanoNetworkTag,
		"--socket-path", socketPath,
		"--out-file", txSigned,
	}
	logger.Record.Info("signing tx", "ARGS", signArgs)
	if out, err := execCmd("cardano-cli", signArgs...); err != nil {
		return "", fmt.Errorf("tx sign: %v (%s)", err, out)
	}

	// Submit
	submitArgs := []string{"conway", "transaction", "submit", CardanoNetworkTag, "--tx-file", txSigned, "--socket-path", socketPath,}
	logger.Record.Info("submitting tx", "ARGS", submitArgs)
	if out, err := execCmd("cardano-cli", submitArgs...); err != nil {
		return "", fmt.Errorf("tx submit: %v (%s)", err, out)
	}

	// Query the txid from the signed file
	idArgs := []string{"conway", "transaction", "txid", "--tx-file", txSigned, "--socket-path", socketPath,}
	out, err = execCmd("cardano-cli", idArgs...)
	if err != nil {
		return "", fmt.Errorf("txid: %v (%s)", err, out)
	}
	return strings.TrimSpace(out), nil
}

// After distribution, send 20 ADA to CARDANO_VALLEY, send leftover to refund address or to CARDANO_VALLEY too.
// Ensure wallet ends up 0.
func payServiceFeeAndDrain(s *AirdropSession) error {
	cardano_valley_address := getEnv("CARDANO_VALLEY_ADDRESS")
	if cardano_valley_address == "" {
		return errors.New("CARDANO_VALLEY_ADDRESS env var is required")
	}

	ctx := context.Background()
	// Check current balance
	bal, err := blockfrost.GetAddressBalance(ctx, s.Address)
	if err != nil {
		return err
	}
	if bal <= 0 {
		return nil // already empty
	}

	// We try to send service fee + remainder out in one go.
	// If balance is < serviceFee, user underfunded; return error.
	if bal < serviceFeeLovelace {
		return fmt.Errorf("insufficient balance for 20 ADA fee: have %d lovelace", bal)
	}

	// Build outputs:
	//  - 20 ADA to cardano_valley
	//  - remainder to refund (or cardano_valley) ; cardano-cli will compute change if needed
	refund := cardano_valley_address

	txBody := filepath.Join(s.WalletDir, "fee_tx.raw")
	txSigned := filepath.Join(s.WalletDir, "fee_tx.signed")

	args := []string{"conway", "transaction", "build",
		CardanoNetworkTag,
		"--change-address", s.Address,
		"--tx-out", fmt.Sprintf("%s+%d", cardano_valley_address, serviceFeeLovelace),
		"--tx-out", fmt.Sprintf("%s+%d", refund, bal-serviceFeeLovelace-1), // rough; change will fix exacts
		"--out-file", txBody,
	}
	if out, err := execCmd("cardano-cli", args...); err != nil {
		return fmt.Errorf("fee tx build: %v (%s)", err, out)
	}

	signArgs := []string{"conway", "transaction", "sign",
		"--tx-body-file", txBody,
		"--signing-key-file", s.SKeyFile,
		CardanoNetworkTag,
		"--out-file", txSigned,
	}
	if out, err := execCmd("cardano-cli", signArgs...); err != nil {
		return fmt.Errorf("fee tx sign: %v (%s)", err, out)
	}

	submitArgs := []string{"conway", "transaction", "submit", CardanoNetworkTag, "--tx-file", txSigned}
	if out, err := execCmd("cardano-cli", submitArgs...); err != nil {
		return fmt.Errorf("fee tx submit: %v (%s)", err, out)
	}

	idArgs := []string{"conway", "transaction", "txid", "--tx-file", txSigned}
	out, err := execCmd("cardano-cli", idArgs...)
	if err != nil {
		return fmt.Errorf("fee txid: %v (%s)", err, out)
	}
	s.ServiceFeeTxID = strings.TrimSpace(out)

	// Re-check balance; if any dust remains, attempt final drain to cardano_valley
	time.Sleep(1 * time.Minute)
	left, _ := blockfrost.GetAddressBalance(ctx, s.Address)
	if left > 0 {
		// Try to empty completely
		_ = drainAllTo(s, cardano_valley_address)
	}
	return nil
}

func drainAllTo(s *AirdropSession, to string) error {
	txBody := filepath.Join(s.WalletDir, "drain_tx.raw")
	txSigned := filepath.Join(s.WalletDir, "drain_tx.signed")
	args := []string{"conway", "transaction", "build",
		CardanoNetworkTag,
		"--change-address", to, // push change to "to"
		"--tx-out", fmt.Sprintf("%s+1", to), // dummy; change will take the rest
		"--out-file", txBody,
	}
	if out, err := execCmd("cardano-cli", args...); err != nil {
		return fmt.Errorf("drain build: %v (%s)", err, out)
	}
	signArgs := []string{"conway", "transaction", "sign",
		"--tx-body-file", txBody,
		"--signing-key-file", s.SKeyFile,
		CardanoNetworkTag,
		"--out-file", txSigned,
	}
	if out, err := execCmd("cardano-cli", signArgs...); err != nil {
		return fmt.Errorf("drain sign: %v (%s)", err, out)
	}
	submitArgs := []string{"conway", "transaction", "submit", CardanoNetworkTag, "--tx-file", txSigned}
	if out, err := execCmd("cardano-cli", submitArgs...); err != nil {
		return fmt.Errorf("drain submit: %v (%s)", err, out)
	}
	return nil
}

//
// ────────────────────────────────────────────────────────────────────────────────
//  UTIL
// ────────────────────────────────────────────────────────────────────────────────
//

func execCmd(bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := stdout.String()
	if err != nil {
		return out, fmt.Errorf("%w: %s", err, stderr.String())
	}
	return out, nil
}

func sendDM(s *discordgo.Session, userID, content string) {
	ch, err := s.UserChannelCreate(userID)
	if err != nil {
		return
	}
	_, _ = s.ChannelMessageSend(ch.ID, content)
}

func respondError(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "❌ " + msg,
		},
	})
}

func followupError(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "❌ " + msg,
	})
}

func valOr(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
