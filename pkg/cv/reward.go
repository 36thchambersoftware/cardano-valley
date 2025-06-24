package cv

type (
	Rewards struct {
		Name           string          `json:"name,omitempty"`
		Description    string          `json:"description,omitempty"`
		Icon           string          `json:"icon,omitempty"`       	 // URL to the icon
		AssetType      string          `json:"assetType,omitempty"`  	 // "token" or "nft"
		RewardToken    Asset          `json:"rewardToken,omitempty"`    // e.g., "abc123.PUNKS" <policyid.assetname>
		AmountPerUser  uint64          `json:"amountPerUser,omitempty"`  // Amount of token per reward
		RolesEligible  []string        `json:"rolesEligible,omitempty"`  // Discord role names or IDs
	}
)