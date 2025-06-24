package blockfrost

import (
	"context"

	"github.com/blockfrost/blockfrost-go"
)

type AddressExtended struct {
	Address      string   `json:"address,omitempty"`
	Amount       []Amount `json:"amount,omitempty"`
	StakeAddress string   `json:"stake_address,omitempty"`
	Type         string   `json:"type,omitempty"`
	Script       bool     `json:"script,omitempty"`
}

func VerifyAddress(ctx context.Context, address string) (bool) {
	_, err := client.Address(ctx, address)
	if err != nil {
		return false
	}

	return true
}

func GetAddress(ctx context.Context, address string) (blockfrost.Address, error) {
	addr, err := client.Address(ctx, address)
	if err != nil {
		return blockfrost.Address{}, err
	}

	return addr, nil
}