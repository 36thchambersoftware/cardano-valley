package maestro

import (
	"cardano-valley/pkg/logger"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/maestro-org/go-sdk/client"
	"github.com/maestro-org/go-sdk/utils"
)

var (
	MAESTRO_URL = "https://mainnet.gomaestro-api.org/v1/"
	maestroClient *client.Client
	maestroToken  string
	httpClient    *http.Client
)

type PolicyMints struct {
	Data        []MintData      `json:"data,omitempty"`
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

type Holder struct {
	Address  string `json:"address,omitempty"`  // Address of the holder
	Quantity uint64 `json:"quantity,omitempty"` // Quantity of the asset held
}

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

func GetPolicyHolders(policyID string) ([]Holder, error) {
	if maestroClient == nil {
		return nil, fmt.Errorf("maestro client not initialized")
	}
	
	var allHolders []Holder
	var cursor *string // start with nil
	for {
		params := utils.Parameters{}
		if cursor != nil {
			params.Cursor(*cursor)
		}

		resp, err := maestroClient.AddressesHoldingPolicy(policyID, &params)
		if err != nil {
			return nil, fmt.Errorf("could not get policy holders: %w", err)
		}

		// Convert Maestro response into your Holder struct
		for _, h := range resp.Data {
			var quantity uint64 = 0
			if len(h.Assets) > 0 {
				for _, asset := range h.Assets {
					quantity += uint64(asset.Amount)
				}
			}
			allHolders = append(allHolders, Holder{
				Address: h.Address,
				Quantity:  quantity,
			})
		}

		// Check for next cursor
		if &resp.NextCursor == nil {
			break // no more pages
		}
		cursor = &resp.NextCursor
	}

	logger.Record.Info("MAESTRO Fetched all policy holders", "POLICY", policyID, "TOTAL_HOLDERS", len(allHolders))
	return allHolders, nil
}