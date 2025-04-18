package blockfrost

import (
	"context"

	bfg "github.com/blockfrost/blockfrost-go"
)

func AssetInfo(ctx context.Context, policyID string) (bfg.Asset, error) {
	info, err := client.Asset(ctx, policyID)
	if err != nil {
		return bfg.Asset{}, err
	}

	return info, nil
}