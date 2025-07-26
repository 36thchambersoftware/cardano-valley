package cv

import (
	mongo "cardano-valley/pkg/db"
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
)

type (
	Reward struct {
		Name           string          `json:"name"`
		Description    string          `json:"description,omitempty"` // Description of the reward
		Icon           string          `json:"icon,omitempty"`       	 // URL to the icon
		AssetType      string          `json:"assetType"`  	 // "token" or "nft" - token ONLY for now
		RewardToken    Asset           `json:"rewardToken"`    // e.g., "abc123.PUNKS" <policyid.assetname>
		RoleAmount     uint64          `json:"roleAmount,omitempty"`  // Amount of token per role
		RolesEligible  []string        `json:"rolesEligible,omitempty"`  // Discord role names or IDs
		AssetsEligible []string        `json:"assetsEligible,omitempty"` // List of asset policy IDs or names
		AssetMinimum   uint64          `json:"assetMinimum,omitempty"` // Minimum amount of asset required to claim
		Balance        uint64          `json:"balance"`
		GuildID		   ServerID        `json:"guild_id"`
	}
)

func LoadRewardsByGuildID(guild_id string) []Reward {
	if mongo.DB == nil {
		log.Println("Waiting for DB...")
		return nil
	}
	collection := mongo.DB.Database("cardano-valley").Collection("reward")
	filter := bson.D{{Key: "guild_id", Value: guild_id}}
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var rewards []Reward
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		log.Fatalf("cannot find rewards: %v", err)
	}

	for {
		if cursor.TryNext(ctx) {
			var reward Reward
			if err := cursor.Decode(&reward); err != nil {
				log.Fatal(err)
			}

			rewards = append(rewards, reward)

			continue
		}
		if err := cursor.Err(); err != nil {
			log.Fatal(err)
		}
		if cursor.ID() == 0 {
			break
		}
	}

	return rewards
}