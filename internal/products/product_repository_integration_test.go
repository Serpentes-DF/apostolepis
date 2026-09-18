//go:build integration

package products

import (
	"context"
	"errors"
	"os"
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
	database := client.Database("products_test_" + bson.NewObjectID().Hex())
	t.Cleanup(func() {
		_ = database.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})
	return database
}

func TestProductRepositoryPersistence(t *testing.T) {
	repository := NewMongoRepository(integrationDatabase(t))
	ctx := context.Background()
	if err := repository.EnsureIndexes(ctx); err != nil {
		t.Fatalf("EnsureIndexes() error = %v", err)
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	created, err := repository.Create(ctx, Product{Name: "Guide", SKU: "G-1", Price: 10, CreatedAt: now, UpdatedAt: now})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	loaded, err := repository.Get(ctx, created.ID)
	if err != nil || loaded.SKU != "G-1" {
		t.Fatalf("Get() = %#v, %v", loaded, err)
	}
	loaded.Price = 12
	if _, err := repository.Update(ctx, loaded); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := repository.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repository.Get(ctx, created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() after delete error = %v, want %v", err, ErrNotFound)
	}
}

func TestProductRepositoryEnforcesUniqueSKU(t *testing.T) {
	repository := NewMongoRepository(integrationDatabase(t))
	ctx := context.Background()
	if err := repository.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	product := Product{Name: "Guide", SKU: "G-1", Price: 10, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if _, err := repository.Create(ctx, product); err != nil {
		t.Fatal(err)
	}
	product.ID = bson.NilObjectID
	if _, err := repository.Create(ctx, product); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate Create() error = %v, want %v", err, ErrDuplicate)
	}
}
