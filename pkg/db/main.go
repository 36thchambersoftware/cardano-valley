package db

import (
	"cardano-valley/pkg/logger"
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var DB *mongo.Client

func Close(client *mongo.Client, ctx context.Context, cancel context.CancelFunc){
	defer cancel()

	defer func() {
	if err := client.Disconnect(ctx); err != nil {
		panic(err)
	}
	}()
}


func Connect() (*mongo.Client, context.Context, context.CancelFunc, error) {
	// Use the SetServerAPIOptions() method to set the version of the Stable API on the client
	CARDANO_VALLEY_MONGODB_PASSWORD, ok := os.LookupEnv("CARDANO_VALLEY_MONGODB_PASSWORD")
	if !ok {
		logger.Record.Error("MONGO", "ERROR", "Could not get mongo db password")
	}

	CARDANO_VALLEY_MONGODB_INSTANCE, ok := os.LookupEnv("CARDANO_VALLEY_MONGODB_INSTANCE")
	if !ok {
		logger.Record.Error("MONGO", "ERROR", "Could not get mongo db instance")
	}
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	connectionString := fmt.Sprintf("mongodb+srv://preeb:%s@%s", CARDANO_VALLEY_MONGODB_PASSWORD, CARDANO_VALLEY_MONGODB_INSTANCE)
	opts := options.Client().ApplyURI(connectionString).SetServerAPIOptions(serverAPI)

	// Create a new DB and connect to the server
	ctx, cancel := context.WithTimeout(context.Background(), 30 * time.Second)
	DB, err := mongo.Connect(ctx, opts)
	if err != nil {
		panic(err)
	}

	// Send a ping to confirm a successful connection
	err = DB.Ping(ctx, nil)

	if err != nil {
		fmt.Println("There was a problem connecting to your Atlas cluster. Check that the URI includes a valid username and password, and that your IP address has been added to the access list. Error: ")
		panic(err)
	}

	fmt.Println("Connected to MongoDB!")
	return DB, ctx, cancel, err
}

