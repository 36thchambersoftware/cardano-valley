package cv

import (
	"cardano-valley/pkg/cardano"
	mongo "cardano-valley/pkg/db"
	"cardano-valley/pkg/logger"
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type (
	Users []User
	User struct {
		ID            string         `json:"id,omitempty"`
		Wallet        cardano.Keys   `json:"wallet,omitempty"`
		LinkedWallets []Wallet      `json:"linked_wallets,omitempty"` // List of linked wallets
		Rewards       map[ServerID]Balance `json:"rewards"`
	}
	Wallet struct {
		Payment string `json:"payment,omitempty"`
		Stake   string `json:"stake,omitempty"`
	}

	Balance map[Asset]struct {
		Earned       uint64    `json:"earned"`
		LastClaimed  time.Time `json:"last_claimed"`
	}
)

/*
CREATE USER WALLET
cardano-cli address key-gen --verification-key-file payment.vkey --signing-key-file payment.skey
cardano-cli conway stake-address key-gen --verification-key-file stake.vkey --signing-key-file stake.skey
cardano-cli address build --payment-verification-key-file payment.vkey --stake-verification-key-file stake.vkey --mainnet --out-file payment.addr

BUILD AND SIGN TX
cardano-cli query utxo --mainnet --address $(cat payment.addr)
cardano-cli transaction build --babbage-era --mainnet --tx-in $tx_in --tx-out $receiver+"1500000 + $quantity $policy_id.$token_hex" --mint "$quantity $policy_id.$token_hex" --mint-script-file $mint_script_file_path --change-address $sender --required-signer payment.skey --out-file mint-native-assets.draft
cardano-cli conway transaction sign --signing-key-file payment.skey --signing-key-file $sender_key --mainnet --tx-body-file mint-native-assets.draft --out-file mint-native-assets.signed
cardano-cli conway transaction submit --tx-file mint-native-assets.signed --mainnet
*/
func LoadUser(userID string) User {
	collection := mongo.DB.Database("cardano-valley").Collection("user")
	filter := bson.D{{Key: "id", Value: userID}}
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var user User
	err := collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		log.Printf("cannot find user: %v", err)
	}

	if user.Rewards == nil {
		user.Rewards = make(map[ServerID]Balance)
	}

	return user
}

func (u User) Save() interface{} {
	collection := mongo.DB.Database("cardano-valley").Collection("user")
	opts := options.Replace().SetUpsert(true)
	filter := bson.D{{Key: "id", Value: u.ID}}
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	result, err := collection.ReplaceOne(ctx, filter, u, opts)
	if err != nil {
		log.Printf("cannot save user: %v", err)
	}

	return result.UpsertedID
}

func LoadUsers() Users {
	if mongo.DB == nil {
		log.Println("Waiting for DB...")
		return nil
	}
	collection := mongo.DB.Database("cardano-valley").Collection("user")
	filter := bson.D{}
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var users Users
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		log.Printf("cannot find users: %v", err)
	}

	for {
		if cursor.TryNext(context.TODO()) {
			var user User
			if err := cursor.Decode(&user); err != nil {
				log.Fatal(err)
			}

			if user.Rewards == nil {
				user.Rewards = make(map[ServerID]Balance)
			}

			users = append(users, user)

			continue
		}
		if err := cursor.Err(); err != nil {
			log.Fatal(err)
		}
		if cursor.ID() == 0 {
			break
		}
	}

	return users
}

func (u User) HarvestRewards(changeAddr string) error {
	// This function should implement the logic to harvest rewards for the user
	// For now, we will just log the action
	logger.Record.Info("Harvesting rewards", "userID", u.ID, "address", changeAddr)

	txOutMap := make(cardano.TxOutMap)

	for serverID, reward := range u.Rewards {
		for asset, balance := range reward {
			txOutMap[string(serverID)] = struct{Asset cardano.Asset; Amount uint64}{
				Asset: cardano.Asset(asset),
				Amount: balance.Earned,
			}
		}
	}

	cardano.BuildTxFromBalance(changeAddr, txOutMap)

	return nil
}