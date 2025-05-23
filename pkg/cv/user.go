package cv

import (
	"cardano-valley/pkg/cardano"
	mongo "cardano-valley/pkg/db"
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type (
	UsersMap map[string]User
	Users []User
	User struct {
		ID string `json:"id,omitempty"`
		Wallet cardano.Wallet `json:"wallet,omitempty"`
		Balance Balance `json:"balance,omitempty"` // Map of asset names to balances
	}

	Balance map[Asset]uint64
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

	if user.Balance == nil {
		user.Balance = make(Balance)
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

			if user.Balance == nil {
				user.Balance = make(Balance)
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

// func (u User) Send(to string, amount int) (string, error) {
// 	if u.Wallet.PaymentKey == "" {
// 		return "", cardano.ErrWalletDoesNotExist
// 	}

// 	signingPaymentKey, err := mongo.Decrypt(u.Wallet.SigningPaymentKey)

// 	tx, err := cardano.SendAll(u.Wallet.Address, to, signingPaymentKey)
// 	return tx.String(), err
// }