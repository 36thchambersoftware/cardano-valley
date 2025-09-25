// Package maestro provides functions for interacting with the Maestro API for Cardano blockchain data.
package maestro

import (
	"cardano-valley/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/maestro-org/go-sdk/client"
)

var (
	MAESTRO_URL = "https://mainnet.gomaestro-api.org/v1/"
	maestroClient *client.Client
	maestroToken  string
	httpClient    *http.Client
)

type PolicyMints struct {
	Data        []MintData  `json:"data,omitempty"`
	LastUpdated LastUpdated `json:"last_updated,omitempty"`
	NextCursor  any         `json:"next_cursor,omitempty"`
}
type MintData struct {
	TxHash string   `json:"tx_hash,omitempty"`
	Slot   int      `json:"slot,omitempty"`
	Assets []string `json:"assets,omitempty"`
}
type LastUpdated struct {
	Timestamp string `json:"timestamp,omitempty"`
	BlockHash string `json:"block_hash,omitempty"`
	BlockSlot int    `json:"block_slot,omitempty"`
}

type PolicyHoldersResponse struct {
	Data       []Holder `json:"data"`
	NextCursor *string  `json:"next_cursor"`
}

type Holder struct {
	Address  string   `json:"address,omitempty"`  // Address of the holder
	Assets   []Amount `json:"assets,omitempty"` // Quantity of the asset held
}

type Amount struct {
	Name     string `json:"name,omitempty"`      // Asset unit (policyID + asset name in hex)
	Amount   uint64 `json:"amount,omitempty"` // Quantity of the asset
}

var AllHolders map[string]uint64

func loadMaestroToken() string {
	token, ok := os.LookupEnv("MAESTRO_TOKEN")
	if !ok {
		slog.Error("Could not get maestro token")
	}
	return token
}

func init() {
	maestroToken = loadMaestroToken()
	if maestroToken != "" {
		maestroClient = client.NewClient(maestroToken, "mainnet")
		httpClient = &http.Client{}
		slog.Info("Maestro client initialized successfully")
	} else {
		slog.Warn("Maestro client not initialized - missing MAESTRO_TOKEN")
	}
}

// Test function to verify client is working
func GetBlockInfo(ctx context.Context, blockNumber int64) error {
	if maestroClient == nil {
		return fmt.Errorf("maestro client not initialized")
	}

	blockInfo, err := maestroClient.BlockInfo(blockNumber)
	if err != nil {
		return err
	}
	
	slog.Info("Block info retrieved", "BLOCK", blockNumber, "SLOT", blockInfo.Data.AbsoluteSlot)
	return nil
}

// Get mint transactions for a policy - this could replace koios.GetPolicyAssetMints
// func GetPolicyMintTransactions(ctx context.Context, policyID string, params *utils.Parameters) ([]MintData, error) {
// 	if maestroClient == nil {
// 		return nil, fmt.Errorf("maestro client not initialized")
// 	}

// 	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
// 	defer cancel()

// 	url, err := url.Parse(fmt.Sprintf("%policy/%s/mints&order=desc", MAESTRO_URL, policyID))
// 	if err != nil {
// 		return nil, err
// 	}
	
// 	// Use TransactionsMovingPolicy to get mint transactions
// 	txs, err := maestroClient.TransactionsMovingPolicy(policyID, nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("could not get policy transactions: %v", err)
// 	}
	
// 	slog.Info("Policy transactions retrieved", "POLICY", policyID, "TOTAL_TXS", len(txs.Data))
// 	return nil
// }

// Get assets for a stake address - this could replace blockfrost.AccountAssociatedAssets
func GetAccountAssets(ctx context.Context, stakeAddress string) error {
	if maestroClient == nil {
		return fmt.Errorf("maestro client not initialized")
	}
	
	assets, err := maestroClient.AccountAssets(stakeAddress, nil)
	if err != nil {
		return fmt.Errorf("could not get account assets: %v", err)
	}
	
	slog.Info("Account assets retrieved", "STAKE", stakeAddress, "TOTAL_ASSETS", len(assets.Data))
	return nil
}

// Get policy information - could provide better mint tracking data
func GetPolicyInformation(ctx context.Context, policyID string) error {
	if maestroClient == nil {
		return fmt.Errorf("maestro client not initialized")
	}
	
	info, err := maestroClient.SpecificPolicyInformations(policyID, nil)
	if err != nil {
		return fmt.Errorf("could not get policy information: %v", err)
	}
	
	slog.Info("Policy information retrieved", "POLICY", policyID, "TOTAL_ASSETS", len(info.Data))
	return nil
}

func GetPolicyHolders(policyID string) (map[string]uint64, error) {
	all := make(map[string]uint64)
	var cursor string

	client := &http.Client{}

	for {
		// Build request URL with optional cursor
		//curl --request GET --url 'https://mainnet.gomaestro-api.org/v1/policy/4fe9470db1c495804278c40d9ded1a46cae725a87c5280f17bab281c/addresses?count=100' --header 'api-key: hidden'
		endpoint, _ := url.Parse(fmt.Sprintf("%spolicy/%s/addresses", MAESTRO_URL, policyID))
		q := endpoint.Query()
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		endpoint.RawQuery = q.Encode()

		// Build request
		req, err := http.NewRequest("GET", endpoint.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("api-key", loadMaestroToken())

		// Execute request
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("API error: %s", string(body))
		}

		// Decode response
		var page PolicyHoldersResponse
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		logger.Record.Info("PAGE", "data", string(data))

		err = json.Unmarshal(data, &page)
		if err != nil {
			return nil, err
		}

		// Append this page’s holders
		for _, h := range page.Data {
			all[h.Address] = 0
			for _, a := range h.Assets {
				all[h.Address] += a.Amount
			}
		}

		// Break if no more cursor
		if page.NextCursor == nil || *page.NextCursor == "" {
			break
		}
		cursor = *page.NextCursor
		logger.Record.Info("CURSOR", "cursor", cursor)
	}

	return all, nil
}
