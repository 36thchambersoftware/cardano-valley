package cv

import (
	"cardano-valley/pkg/cardano"
	mongo "cardano-valley/pkg/db"
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Configs []Config

type (
	Role string
	Roles map[Role][]string // Map of role names to user IDs
	Config struct {
		GuildID         string      	 `bson:"guild_id,omitempty"`
		Wallet          cardano.Wallet   `bson:"wallet,omitempty"`
		Rewards     	[]Rewards        `json:"rewards,omitempty"`
	}

	Rewards struct {
		Name           string          `json:"name,omitempty"`
		Description    string          `json:"description,omitempty"`
		Icon           string          `json:"icon,omitempty"`       	 // URL to the icon
		AssetType      string          `json:"assetType,omitempty"`  	 // "token" or "nft"
		RewardToken    string          `json:"rewardToken,omitempty"`    // e.g., "abc123.PUNKS" <policyid.assetname>
		AmountPerUser  uint64          `json:"amountPerUser,omitempty"`  // Amount of token per reward
		RolesEligible  []string        `json:"rolesEligible,omitempty"`  // Discord role names or IDs
		MinHoldDays    uint            `json:"minHoldDays,omitempty"`    // Minimum holding period
		Frequency      uint       	   `json:"frequency,omitempty"`      // Frequency of rewards in days
	}
)

func (c Config) Save() interface{} {
	collection := mongo.DB.Database("cardano-valley").Collection("config")
	opts := options.Replace().SetUpsert(true)
	filter := bson.D{{Key: "guild_id", Value: c.GuildID}}
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	
	result, err := collection.ReplaceOne(ctx, filter, c, opts)
	if err != nil {
		log.Fatalf("cannot save config: %v", err)
	}

	return result.UpsertedID
}

func LoadConfig(guild_id string) Config {
	collection := mongo.DB.Database("cardano-valley").Collection("config")
	filter := bson.D{{Key: "guild_id", Value: guild_id}}
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var config Config
	err := collection.FindOne(ctx, filter).Decode(&config)
	if err != nil {
		log.Printf("cannot find config: %v", err)
	}

	if config.GuildID == "" {
		config.GuildID = guild_id
	}

	return config
}

func LoadConfigs() Configs {
	if mongo.DB == nil {
		log.Println("Waiting for DB...")
		return nil
	}
	collection := mongo.DB.Database("cardano-valley").Collection("config")
	filter := bson.D{}
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var configs Configs
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		log.Fatalf("cannot find configs: %v", err)
	}

	for {
		if cursor.TryNext(context.TODO()) {
			var config Config
			if err := cursor.Decode(&config); err != nil {
				log.Fatal(err)
			}

			configs = append(configs, config)

			continue
		}
		if err := cursor.Err(); err != nil {
			log.Fatal(err)
		}
		if cursor.ID() == 0 {
			break
		}
	}

	return configs
}