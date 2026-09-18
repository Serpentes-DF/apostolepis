//go:build integration

package users

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
	database := client.Database("users_test_" + bson.NewObjectID().Hex())
	t.Cleanup(func() {
		_ = database.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})
	return database
}

func TestUserRepositoryPersistenceAndUniqueEmail(t *testing.T) {
	repository := NewMongoRepository(integrationDatabase(t))
	ctx := context.Background()
	if err := repository.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	user := User{Name: "Ada", Email: "ada@example.com", IsAdmin: true, CreatedAt: time.Now().UTC()}
	created, err := repository.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	loaded, err := repository.Get(ctx, created.ID)
	if err != nil || loaded.Email != user.Email || !loaded.IsAdmin {
		t.Fatalf("Get() = %#v, %v", loaded, err)
	}
	byEmail, err := repository.GetByEmail(ctx, created.Email)
	if err != nil || byEmail.ID != created.ID {
		t.Fatalf("GetByEmail() = %#v, %v", byEmail, err)
	}
	listed, err := repository.List(ctx)
	if err != nil || len(listed) != 1 {
		t.Fatalf("List() = %#v, %v", listed, err)
	}
	user.ID = bson.NilObjectID
	if _, err := repository.Create(ctx, user); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate Create() error = %v, want %v", err, ErrDuplicate)
	}
}
