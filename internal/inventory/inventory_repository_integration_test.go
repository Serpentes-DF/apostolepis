//go:build integration

package inventory

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	platformmongo "github.com/Serpentes-DF/apostolepis/internal/platform/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func integrationDatabase(t *testing.T) *mongo.Database {
	t.Helper()
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		t.Fatal("MONGO_URI is required for integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	client, err := platformmongo.Connect(ctx, uri)
	if err != nil {
		t.Fatalf("connect to MongoDB: %v", err)
	}
	database := client.Database("inventory_test_" + bson.NewObjectID().Hex())
	t.Cleanup(func() {
		_ = database.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})
	return database
}

func TestInventoryRepositoryAdjustsAtomically(t *testing.T) {
	repository := NewMongoRepository(integrationDatabase(t))
	ctx := context.Background()
	if err := repository.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	productID := bson.NewObjectID()
	if _, adjusted, err := repository.Adjust(ctx, productID, 10); err != nil || !adjusted {
		t.Fatalf("initial Adjust() adjusted = %v, error = %v", adjusted, err)
	}

	var wait sync.WaitGroup
	results := make(chan bool, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, adjusted, err := repository.Adjust(ctx, productID, -7)
			results <- err == nil && adjusted
		}()
	}
	wait.Wait()
	close(results)
	successes := 0
	for adjusted := range results {
		if adjusted {
			successes++
		}
	}
	stock, err := repository.Get(ctx, productID)
	if err != nil {
		t.Fatal(err)
	}
	if successes != 1 || stock.Quantity != 3 {
		t.Fatalf("successful decrements = %d, quantity = %d; want 1 and 3", successes, stock.Quantity)
	}
}
