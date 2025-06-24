package cv

type (
	PolicyID string
	Policy struct {
		Reward uint64 `bson:"reward,omitempty"`
		HexName string `bson:"hex_name,omitempty"` // Tokens only - NFTs will have empty hex name
	}

	// String here being the policy id
	PolicyIDs map[PolicyID]Policy

	Asset string // policyid.assetname
)