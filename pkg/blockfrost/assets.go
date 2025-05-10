package blockfrost

import (
	"context"

	bfg "github.com/blockfrost/blockfrost-go"
)

type Amount struct {
	Unit                  string `json:"unit,omitempty"`
	Quantity              string `json:"quantity,omitempty"`
	Decimals              int    `json:"decimals,omitempty"`
	HasNftOnchainMetadata bool   `json:"has_nft_onchain_metadata,omitempty"`
}

func AssetInfo(ctx context.Context, policyID string) (bfg.Asset, error) {
	info, err := client.Asset(ctx, policyID)
	if err != nil {
		return bfg.Asset{}, err
	}

	return info, nil
}