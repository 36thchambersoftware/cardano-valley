package blockfrost

import (
	"context"
	"strconv"

	"github.com/blockfrost/blockfrost-go"
)

type AddressExtended struct {
	Address      string   `json:"address,omitempty"`
	Amount       []Amount `json:"amount,omitempty"`
	StakeAddress string   `json:"stake_address,omitempty"`
	Type         string   `json:"type,omitempty"`
	Script       bool     `json:"script,omitempty"`
}

func VerifyAddress(ctx context.Context, address string) bool {
	_, err := client.Address(ctx, address)
	return err == nil
}

func GetAddress(ctx context.Context, address string) (blockfrost.Address, error) {
	addr, err := client.Address(ctx, address)
	if err != nil {
		return blockfrost.Address{}, err
	}

	return addr, nil
}

func GetAddressBalance_Blockfrost(ctx context.Context, address string) (uint64, error) {
	addr, err := client.Address(ctx, address)
	if err != nil {
		return 0, err
	}

	var lovelace uint64
	for _, a := range addr.Amount {
		if a.Unit == "lovelace" {
			v, _ := strconv.ParseInt(a.Quantity, 10, 64)
			lovelace += uint64(v)
		}
	}
	return lovelace, nil
}