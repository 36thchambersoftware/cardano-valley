package blockfrost

import "context"

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