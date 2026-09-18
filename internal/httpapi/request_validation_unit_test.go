//go:build unit

package httpapi

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Serpentes-DF/apostolepis/internal/inventory"
	"github.com/Serpentes-DF/apostolepis/internal/products"
	"github.com/Serpentes-DF/apostolepis/internal/users"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakeProducts struct{ createError error }

func (service fakeProducts) Create(context.Context, products.CreateInput) (products.Product, error) {
	return products.Product{}, service.createError
}
func (fakeProducts) Get(context.Context, bson.ObjectID) (products.Product, error) {
	return products.Product{}, nil
}
func (fakeProducts) List(context.Context) ([]products.Product, error) { return nil, nil }
func (fakeProducts) Update(context.Context, bson.ObjectID, products.UpdateInput) (products.Product, error) {
	return products.Product{}, nil
}
func (fakeProducts) Delete(context.Context, bson.ObjectID) error { return nil }

type unusedInventory struct{}

func (unusedInventory) Adjust(context.Context, bson.ObjectID, int64) (inventory.Inventory, error) {
	return inventory.Inventory{}, nil
}
func (unusedInventory) Get(context.Context, bson.ObjectID) (inventory.Inventory, error) {
	return inventory.Inventory{}, nil
}

type unusedUsers struct{}

func (unusedUsers) Create(context.Context, users.CreateInput) (users.User, error) {
	return users.User{}, nil
}
func (unusedUsers) Get(context.Context, bson.ObjectID) (users.User, error) { return users.User{}, nil }
func (unusedUsers) GetByEmail(context.Context, string) (users.User, error) {
	return users.User{}, users.ErrNotFound
}
func (unusedUsers) List(context.Context) ([]users.User, error) { return nil, nil }

type fakeUsers struct {
	getByEmail func(string) (users.User, error)
}

func (fakeUsers) Create(context.Context, users.CreateInput) (users.User, error) {
	return users.User{}, nil
}
func (fakeUsers) Get(context.Context, bson.ObjectID) (users.User, error) { return users.User{}, nil }
func (service fakeUsers) GetByEmail(_ context.Context, email string) (users.User, error) {
	return service.getByEmail(email)
}
func (fakeUsers) List(context.Context) ([]users.User, error) { return nil, nil }

func testRouter(productsService ProductService) http.Handler {
	return NewRouter(Dependencies{
		Products: productsService, Inventory: unusedInventory{}, Users: unusedUsers{},
		Ping: func(context.Context) error { return nil },
	})
}

func testRouterWithUsers(productsService ProductService, usersService UserService) http.Handler {
	return NewRouter(Dependencies{
		Products: productsService, Inventory: unusedInventory{}, Users: usersService,
		Ping: func(context.Context) error { return nil },
	})
}

func TestCreateProductRejectsMalformedJSON(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewBufferString("{"))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-User-Email", "admin@example.com")
	response := httptest.NewRecorder()
	testRouterWithUsers(fakeProducts{}, fakeUsers{getByEmail: func(email string) (users.User, error) {
		return users.User{Email: email, IsAdmin: true}, nil
	}}).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestCreateProductMapsDuplicate(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewBufferString(`{"name":"Guide","sku":"G-1","price":10}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-User-Email", "admin@example.com")
	response := httptest.NewRecorder()
	testRouterWithUsers(fakeProducts{createError: products.ErrDuplicate}, fakeUsers{getByEmail: func(email string) (users.User, error) {
		return users.User{Email: email, IsAdmin: true}, nil
	}}).ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
}

func TestGetProductRejectsInvalidID(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/products/not-an-id", nil)
	response := httptest.NewRecorder()
	testRouter(fakeProducts{}).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestCreateProductRequiresAdmin(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewBufferString(`{"name":"Guide","sku":"G-1","price":10}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	testRouter(fakeProducts{}).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestCreateProductRejectsNonAdmin(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewBufferString(`{"name":"Guide","sku":"G-1","price":10}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-User-Email", "viewer@example.com")
	response := httptest.NewRecorder()
	testRouterWithUsers(fakeProducts{}, fakeUsers{getByEmail: func(email string) (users.User, error) {
		return users.User{Email: email, IsAdmin: false}, nil
	}}).ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestCreateProductAllowsAdmin(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewBufferString(`{"name":"Guide","sku":"G-1","price":10}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-User-Email", "admin@example.com")
	response := httptest.NewRecorder()
	testRouterWithUsers(fakeProducts{}, fakeUsers{getByEmail: func(email string) (users.User, error) {
		return users.User{Email: email, IsAdmin: true}, nil
	}}).ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
}

func TestCreateProductAdminLookupFailure(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewBufferString(`{"name":"Guide","sku":"G-1","price":10}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-User-Email", "admin@example.com")
	response := httptest.NewRecorder()
	testRouterWithUsers(fakeProducts{}, fakeUsers{getByEmail: func(string) (users.User, error) {
		return users.User{}, errors.New("db unavailable")
	}}).ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}

func TestCreateUserRouteIsUnavailable(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(`{"name":"Ada","email":"ada@example.com","is_admin":true}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	testRouter(fakeProducts{}).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}
