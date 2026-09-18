//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Serpentes-DF/apostolepis/internal/inventory"
	platformmongo "github.com/Serpentes-DF/apostolepis/internal/platform/mongodb"
	"github.com/Serpentes-DF/apostolepis/internal/products"
	"github.com/Serpentes-DF/apostolepis/internal/users"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestProductInventoryAndUserHTTPFlow(t *testing.T) {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		t.Fatal("MONGO_URI is required for integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	client, err := platformmongo.Connect(ctx, uri)
	if err != nil {
		t.Fatal(err)
	}
	database := client.Database("http_test_" + bson.NewObjectID().Hex())
	t.Cleanup(func() {
		_ = database.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})
	productRepository := products.NewMongoRepository(database)
	inventoryRepository := inventory.NewMongoRepository(database)
	userRepository := users.NewMongoRepository(database)
	for _, ensure := range []func(context.Context) error{
		productRepository.EnsureIndexes, inventoryRepository.EnsureIndexes, userRepository.EnsureIndexes,
	} {
		if err := ensure(ctx); err != nil {
			t.Fatal(err)
		}
	}
	router := NewRouter(Dependencies{
		Products:  products.NewService(productRepository),
		Inventory: inventory.NewService(inventoryRepository, productRepository),
		Users:     users.NewService(userRepository),
		Ping:      func(ctx context.Context) error { return client.Ping(ctx, nil) },
	})

	if _, err := userRepository.Create(ctx, users.User{
		Name:      "Ada",
		Email:     "ada@example.com",
		IsAdmin:   true,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed admin user: %v", err)
	}
	productResponse := performJSON(router, http.MethodPost, "/api/v1/products", `{"name":"Guide","sku":"G-1","price":10}`, map[string]string{"X-User-Email": "ada@example.com"})
	if productResponse.Code != http.StatusCreated {
		t.Fatalf("create product status = %d, body = %s", productResponse.Code, productResponse.Body.String())
	}
	var created products.Product
	if err := json.Unmarshal(productResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode product: %v", err)
	}
	stockResponse := performJSON(router, http.MethodPost, "/api/v1/products/"+created.ID.Hex()+"/stock-adjustments", `{"delta":5}`, map[string]string{"X-User-Email": "ada@example.com"})
	if stockResponse.Code != http.StatusOK {
		t.Fatalf("adjust stock status = %d, body = %s", stockResponse.Code, stockResponse.Body.String())
	}
	healthResponse := performJSON(router, http.MethodGet, "/health", "", nil)
	if healthResponse.Code != http.StatusOK {
		t.Fatalf("health status = %d", healthResponse.Code)
	}
}

func performJSON(handler http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
