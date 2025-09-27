package koios

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strconv"

	"github.com/cardano-community/koios-go-client/v4"
)

var client *koios.Client
var koiosToken string
type EpochNo koios.EpochNo

func init() {
	var err error
	client, err = koios.New()
	if err != nil {
		slog.Error("could not connect koios api", "ERROR", err)
	}
}

// func AddressInformation(ctx context.Context, addresses []preeb.Address) ([]koios.AddressInfo, error) {
// 	var options *koios.RequestOptions
// 	var addrs []koios.Address

// 	for _, address := range addresses {
// 		addrs = append(addrs, koios.Address(address))
// 	}

// 	result, err := client.GetAddressesInfo(ctx, addrs, options)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if result.StatusCode != 200 {
// 		return nil, errors.New(result.Response.Error.Message)
// 	}

// 	return result.Data, nil
// }

// func AccountHistory(ctx context.Context, stake_addresses []preeb.StakeAddress) ([]koios.AccountHistory, error) {
// 	var addrs []koios.Address
// 	var epoch *koios.EpochNo
// 	var options *koios.RequestOptions
// 	for _, address := range stake_addresses {
// 		addrs = append(addrs, koios.Address(address))
// 	}
// 	result, err := client.GetAccountHistory(ctx, addrs, epoch, options)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if result.StatusCode != 200 {
// 		return nil, errors.New(result.Response.Error.Message)
// 	}

// 	return result.Data, nil
// }

// func GetTransactionDetails(ctx context.Context, txHash string) {
// 	var options *koios.RequestOptions
// 	client.GetTxInfo(ctx, txs []koios.TxHash, opts *koios.RequestOptions)
// }

func loadKoiosToken() string {
	koiosToken, ok := os.LookupEnv("KOIOS_TOKEN")
	if !ok {
		slog.Error("Could not get koios token")
	}

	return koiosToken
}

func init() {
	var err error
	client, err = koios.New()
	if err != nil {
		slog.Error("could not connect koios api", "ERROR", err)
	}

	err = client.SetAuth(loadKoiosToken())
	if err != nil {
		slog.Error("could not set koios token", "ERROR", err)
	}	
}

func AddressInformation(ctx context.Context, addresses []string) ([]koios.AddressInfo, error) {
	var options *koios.RequestOptions
	var addrs []koios.Address

	for _, address := range addresses {
		addrs = append(addrs, koios.Address(address))
	}

	result, err := client.GetAddressesInfo(ctx, addrs, options)
	if err != nil {
		return nil, err
	}

	if result.StatusCode != 200 {
		return nil, errors.New(result.Response.Error.Message)
	}

	return result.Data, nil
}

func AccountHistory(ctx context.Context, stake_addresses []string) ([]koios.AccountHistory, error) {
	var addrs []koios.Address
	var epoch *koios.EpochNo
	var options *koios.RequestOptions
	for _, address := range stake_addresses {
		addrs = append(addrs, koios.Address(address))
	}
	result, err := client.GetAccountHistory(ctx, addrs, epoch, options)
	if err != nil {
		return nil, err
	}

	if result.StatusCode != 200 {
		return nil, errors.New(result.Response.Error.Message)
	}

	return result.Data, nil
}

func Tip(ctx context.Context) (res *koios.Tip, err error) {
	var options *koios.RequestOptions
	result, err := client.GetTip(ctx, options)
	if err != nil {
		return nil, err
	}

	if result.StatusCode != 200 {
		return nil, errors.New(result.Response.Error.Message)
	}

	return &result.Data, nil
}

func GetPolicyTxs(ctx context.Context, policyID, assetName string) ([]koios.AddressTx, error) {
	var options *koios.RequestOptions
	txs, err := client.GetAssetTxs(ctx, koios.PolicyID(policyID), koios.AssetName(assetName), 0, false, options)
	if err != nil {
		return nil, err
	}

	return txs.Data, nil
}

func GetPolicyAssetMints(ctx context.Context, policyID string) ([]koios.PolicyAssetMint, error) {
	var options *koios.RequestOptions
	mints, err := client.GetPolicyAssetMints(ctx, koios.PolicyID(policyID), options)
	if err != nil {
		return nil, err
	}

	if mints.StatusCode != 200 {
		return nil, errors.New(mints.Response.Error.Message)
	}

	sort.Slice(mints.Data, func(i, j int) bool {
		return mints.Data[i].CreationTime.After(mints.Data[j].CreationTime.Time)
	})

	return mints.Data, nil
}

func GetAssetUTXOs(ctx context.Context, policyID, assetName string) (koios.UTxO, error) {
	var options *koios.RequestOptions

	asset := []koios.Asset{{
		PolicyID: koios.PolicyID(policyID),
		AssetName: koios.AssetName(assetName),
	}}
	utxos, err := client.GetAssetUTxOs(ctx, asset, options)
	if err != nil {
		return koios.UTxO{}, err
	}

	if utxos.StatusCode != 200 {
		return koios.UTxO{}, errors.New(utxos.Response.Error.Message)
	}

	if len(utxos.Data) == 0 {
		return koios.UTxO{}, errors.New(fmt.Sprintf("no UTxOs found: %s %v", policyID, utxos))
	}

	return utxos.Data[0], nil
}

func GetDatum(ctx context.Context, datumHash koios.DatumHash) (*koios.DatumInfo, error) {
	var options *koios.RequestOptions
	datum, err := client.GetDatumInfo(ctx, datumHash, options)
	if err != nil {
		return nil, err
	}

	if datum.StatusCode != 200 {
		return nil, errors.New(datum.Response.Error.Message)
	}

	return datum.Data, nil
}

func GetPolicyAssetList(ctx context.Context, policyID string) ([]koios.PolicyAssetListItem, error) {
	// The Koios Go library has a bug where RequestOptions.SetCurrentPage() tries to write to a nil map
	// Until that's fixed, we can only get the first 1000 assets reliably
	// This is still better than getting truncated lists that cause duplicate notifications
	
	var options *koios.RequestOptions
	assets, err := client.GetPolicyAssetList(ctx, koios.PolicyID(policyID), options)
	if err != nil {
		return nil, err
	}

	if assets.StatusCode != 200 {
		return nil, errors.New(assets.Response.Error.Message)
	}

	slog.Info("Fetched policy assets (first 1000 max due to library limitation)", "POLICY", policyID, "TOTAL_ASSETS", len(assets.Data))
	
	// If we got exactly 1000, warn that there may be more
	if len(assets.Data) == 1000 {
		slog.Warn("Policy returned exactly 1000 assets - there may be more, but Koios library pagination is broken", "POLICY", policyID)
	}
	
	return assets.Data, nil
}

func GetBatchedStakeAddressAssets(ctx context.Context, stakeAddresses []string) (map[string]uint64, error) {
	const batchSize = 100 // Koios allows up to 100 addresses per call
	allAssets := make(map[string]uint64)
	
	// Process in batches of 100
	for i := 0; i < len(stakeAddresses); i += batchSize {
		end := i + batchSize
		if end > len(stakeAddresses) {
			end = len(stakeAddresses)
		}
		
		batch := stakeAddresses[i:end]
		batchAssets, err := getBatchStakeAssets(ctx, batch)
		if err != nil {
			return nil, err
		}
		
		// Merge results
		for unit, quantity := range batchAssets {
			allAssets[unit] += quantity
		}
	}
	
	return allAssets, nil
}

func getBatchStakeAssets(ctx context.Context, stakeAddresses []string) (map[string]uint64, error) {
	// Convert stake addresses to koios.Address slice for batched call
	koiosStakeAddrs := make([]koios.Address, len(stakeAddresses))
	for i, stake := range stakeAddresses {
		koiosStakeAddrs[i] = koios.Address(stake)
	}
	
	// ✅ Single batched call to get all addresses for all stake addresses
	var options *koios.RequestOptions
	result, err := client.GetAccountAddresses(ctx, koiosStakeAddrs, false, false, options)
	if err != nil {
		return nil, err
	}
	
	if result.StatusCode != 200 {
		return nil, errors.New(result.Response.Error.Message)
	}
	
	// Collect all addresses from all stake addresses
	var allAddresses []string
	for _, accountAddr := range result.Data {
		for _, addr := range accountAddr.Addresses {
			allAddresses = append(allAddresses, string(addr))
		}
	}
	
	if len(allAddresses) == 0 {
		return make(map[string]uint64), nil
	}
	
	// ✅ Batched calls to get address info for all addresses (batch by 100)
	addressInfos, err := getBatchedAddressInformation(ctx, allAddresses)
	if err != nil {
		return nil, err
	}
	
	// Convert to asset map
	assets := make(map[string]uint64)
	for _, addrInfo := range addressInfos {
		for _, utxo := range addrInfo.UTxOs {
			for _, asset := range utxo.AssetList {
				qty, _ := strconv.ParseUint(asset.Quantity.String(), 10, 64)
				assets[string(asset.PolicyID)+string(asset.AssetName)] += qty
			}
		}
	}
	
	return assets, nil
}

func getBatchedAddressInformation(ctx context.Context, addresses []string) ([]koios.AddressInfo, error) {
	const addressBatchSize = 100 // Koios allows up to 100 addresses per call
	var allAddressInfos []koios.AddressInfo
	
	// Process addresses in batches of 100
	for i := 0; i < len(addresses); i += addressBatchSize {
		end := i + addressBatchSize
		if end > len(addresses) {
			end = len(addresses)
		}
		
		batch := addresses[i:end]
		batchInfos, err := AddressInformation(ctx, batch)
		if err != nil {
			return nil, err
		}
		
		allAddressInfos = append(allAddressInfos, batchInfos...)
	}
	
	return allAddressInfos, nil
}

func GetPolicyHolders(policyID string) (map[string]uint64, error) {
	all := make(map[string]uint64)
	offset := 0

	for {
		var options *koios.RequestOptions
		if offset != 0 {
			if options == nil {
				options = &koios.RequestOptions{}
			}
			options.SetCurrentPage(uint(offset))
		}

		result, err := client.GetPolicyAssetAddresses(context.Background(), koios.PolicyID(policyID), options)
		if err != nil {
			return nil, err
		}

		if result.StatusCode != 200 {
			return nil, errors.New(result.Response.Error.Message)
		}

		for _, holder := range result.Data {
			qty, _ := strconv.ParseUint(holder.Quantity.String(), 10, 64)
			all[holder.PaymentAddress.String()] += qty
		}

		if len(result.Data) == 1000 {
			offset += 1000
		}
	}

	return all, nil
}