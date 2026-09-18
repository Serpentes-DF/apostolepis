//go:build unit

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Serpentes-DF/apostolepis/internal/auth"
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
func (unusedUsers) Get(context.Context, bson.ObjectID) (users.User, error) {
	return users.User{}, nil
}
func (unusedUsers) GetByEmail(context.Context, string) (users.User, error) {
	return users.User{}, users.ErrNotFound
}
func (unusedUsers) List(context.Context) ([]users.User, error) { return nil, nil }

type fakeUsers struct {
	create     func(users.CreateInput) (users.User, error)
	get        func(bson.ObjectID) (users.User, error)
	getByEmail func(string) (users.User, error)
}

type fakeGoogleTokens struct {
	validate func(string) (users.User, error)
}

func (service fakeGoogleTokens) Validate(_ context.Context, token string) (users.User, error) {
	if service.validate == nil {
		return users.User{}, auth.ErrInvalidIdentityToken
	}
	return service.validate(token)
}

func (service fakeUsers) Create(_ context.Context, input users.CreateInput) (users.User, error) {
	if service.create == nil {
		return users.User{}, nil
	}
	return service.create(input)
}
func (service fakeUsers) Get(_ context.Context, id bson.ObjectID) (users.User, error) {
	if service.get == nil {
		return users.User{}, nil
	}
	return service.get(id)
}
func (service fakeUsers) GetByEmail(_ context.Context, email string) (users.User, error) {
	return service.getByEmail(email)
}
func (fakeUsers) List(context.Context) ([]users.User, error) { return nil, nil }

func testRouter(productsService ProductService) http.Handler {
	tokens, _ := auth.NewTokenService("test-secret", time.Hour)
	return NewRouter(Dependencies{
		Products: productsService, Inventory: unusedInventory{}, Users: unusedUsers{}, Tokens: tokens, GoogleTokens: fakeGoogleTokens{},
		Ping: func(context.Context) error { return nil },
	})
}

func testRouterWithUsers(productsService ProductService, usersService UserService) http.Handler {
	return testRouterWithAuth(productsService, usersService, fakeGoogleTokens{})
}

func testRouterWithAuth(productsService ProductService, usersService UserService, googleTokens GoogleIdentityTokenValidator) http.Handler {
	tokens, _ := auth.NewTokenService("test-secret", time.Hour)
	return NewRouter(Dependencies{
		Products: productsService, Inventory: unusedInventory{}, Users: usersService, Tokens: tokens, GoogleTokens: googleTokens,
		Ping: func(context.Context) error { return nil },
	})
}

func issueTestToken(t *testing.T, user users.User) string {
	t.Helper()
	tokens, err := auth.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	token, err := tokens.Issue(user.ID, user.Email)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	return token
}

func TestCreateProductRejectsMalformedJSON(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/products", bytes.NewBufferString("{"))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueTestToken(t, users.User{ID: bson.NewObjectID(), Email: "admin@example.com"}))
	response := httptest.NewRecorder()
	testRouterWithUsers(fakeProducts{}, fakeUsers{
		get: func(id bson.ObjectID) (users.User, error) {
			return users.User{ID: id, Email: "admin@example.com", IsAdmin: true}, nil
		},
		getByEmail: func(email string) (users.User, error) {
			return users.User{ID: bson.NewObjectID(), Email: email, IsAdmin: true}, nil
		},
	}).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestCreateProductMapsDuplicate(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/products", bytes.NewBufferString(`{"name":"Guide","sku":"G-1","price":10}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueTestToken(t, users.User{ID: bson.NewObjectID(), Email: "admin@example.com"}))
	response := httptest.NewRecorder()
	testRouterWithUsers(fakeProducts{createError: products.ErrDuplicate}, fakeUsers{
		get: func(id bson.ObjectID) (users.User, error) {
			return users.User{ID: id, Email: "admin@example.com", IsAdmin: true}, nil
		},
		getByEmail: func(email string) (users.User, error) {
			return users.User{ID: bson.NewObjectID(), Email: email, IsAdmin: true}, nil
		},
	}).ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
}

func TestGetProductRejectsInvalidID(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/products/not-an-id", nil)
	response := httptest.NewRecorder()
	testRouter(fakeProducts{}).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestCreateProductRequiresAdmin(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/products", bytes.NewBufferString(`{"name":"Guide","sku":"G-1","price":10}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	testRouter(fakeProducts{}).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestCreateProductRejectsNonAdmin(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/products", bytes.NewBufferString(`{"name":"Guide","sku":"G-1","price":10}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueTestToken(t, users.User{ID: bson.NewObjectID(), Email: "viewer@example.com"}))
	response := httptest.NewRecorder()
	testRouterWithUsers(fakeProducts{}, fakeUsers{
		get: func(id bson.ObjectID) (users.User, error) {
			return users.User{ID: id, Email: "viewer@example.com", IsAdmin: false}, nil
		},
		getByEmail: func(email string) (users.User, error) {
			return users.User{ID: bson.NewObjectID(), Email: email, IsAdmin: false}, nil
		},
	}).ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestCreateProductAllowsAdmin(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/products", bytes.NewBufferString(`{"name":"Guide","sku":"G-1","price":10}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueTestToken(t, users.User{ID: bson.NewObjectID(), Email: "admin@example.com"}))
	response := httptest.NewRecorder()
	testRouterWithUsers(fakeProducts{}, fakeUsers{
		get: func(id bson.ObjectID) (users.User, error) {
			return users.User{ID: id, Email: "admin@example.com", IsAdmin: true}, nil
		},
		getByEmail: func(email string) (users.User, error) {
			return users.User{ID: bson.NewObjectID(), Email: email, IsAdmin: true}, nil
		},
	}).ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
}

func TestCreateProductAdminLookupFailure(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/products", bytes.NewBufferString(`{"name":"Guide","sku":"G-1","price":10}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+issueTestToken(t, users.User{ID: bson.NewObjectID(), Email: "admin@example.com"}))
	response := httptest.NewRecorder()
	testRouterWithUsers(fakeProducts{}, fakeUsers{get: func(bson.ObjectID) (users.User, error) {
		return users.User{}, errors.New("db unavailable")
	}}).ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}

func TestCreateUserRouteIsUnavailable(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/users", bytes.NewBufferString(`{"name":"Ada","email":"ada@example.com","is_admin":true}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	testRouter(fakeProducts{}).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestDocsServesEmbeddedHTML(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/docs", nil)
	response := httptest.NewRecorder()
	testRouter(fakeProducts{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("content type = %q, want HTML", response.Header().Get("Content-Type"))
	}
	if !strings.Contains(response.Body.String(), "/docs/openapi.yaml") {
		t.Fatalf("body does not reference embedded OpenAPI spec")
	}
}

func TestDocsServesEmbeddedOpenAPISpec(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/docs/openapi.yaml", nil)
	response := httptest.NewRecorder()
	testRouter(fakeProducts{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Header().Get("Content-Type"), "application/yaml") {
		t.Fatalf("content type = %q, want YAML", response.Header().Get("Content-Type"))
	}
	if !strings.Contains(response.Body.String(), "openapi: 3.0.3") {
		t.Fatalf("body = %q, want embedded OpenAPI spec", response.Body.String())
	}
}

func TestDocsServesEmbeddedSwaggerUIAssets(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/docs/swagger-ui.css", nil)
	response := httptest.NewRecorder()
	testRouter(fakeProducts{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Header().Get("Content-Type"), "text/css") {
		t.Fatalf("content type = %q, want CSS", response.Header().Get("Content-Type"))
	}
	if response.Body.Len() == 0 {
		t.Fatal("body is empty, want embedded Swagger UI asset")
	}
}

func TestGetUserOrdersRequiresAuthentication(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/users/0123456789abcdef01234567/orders", nil)
	response := httptest.NewRecorder()
	testRouter(fakeProducts{}).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestGetUserOrdersRejectsDifferentNonAdminUser(t *testing.T) {
	requestedID := bson.NewObjectID()
	authenticatedID := bson.NewObjectID()
	request := httptest.NewRequest(http.MethodGet, "/v1/users/"+requestedID.Hex()+"/orders", nil)
	request.Header.Set("Authorization", "Bearer "+issueTestToken(t, users.User{ID: authenticatedID, Email: "viewer@example.com"}))
	response := httptest.NewRecorder()
	testRouterWithUsers(fakeProducts{}, fakeUsers{get: func(id bson.ObjectID) (users.User, error) {
		return users.User{ID: authenticatedID, Email: "viewer@example.com", IsAdmin: false}, nil
	}}).ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestGetUserOrdersAllowsSameUser(t *testing.T) {
	userID := bson.NewObjectID()
	firstOrderID := bson.NewObjectID()
	secondOrderID := bson.NewObjectID()
	request := httptest.NewRequest(http.MethodGet, "/v1/users/"+userID.Hex()+"/orders", nil)
	request.Header.Set("Authorization", "Bearer "+issueTestToken(t, users.User{ID: userID, Email: "ada@example.com"}))
	response := httptest.NewRecorder()
	testRouterWithUsers(fakeProducts{}, fakeUsers{
		get: func(id bson.ObjectID) (users.User, error) {
			if id == userID {
				return users.User{ID: id, Email: "ada@example.com", IsAdmin: false, Orders: []bson.ObjectID{firstOrderID, secondOrderID}}, nil
			}
			return users.User{ID: id, Orders: []bson.ObjectID{firstOrderID, secondOrderID}}, nil
		},
	}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if body := response.Body.String(); !strings.Contains(body, firstOrderID.Hex()) || !strings.Contains(body, secondOrderID.Hex()) {
		t.Fatalf("body = %q, want both order IDs", body)
	}
}

func TestGetUserOrdersAllowsAdmin(t *testing.T) {
	adminID := bson.NewObjectID()
	requestedID := bson.NewObjectID()
	orderID := bson.NewObjectID()
	request := httptest.NewRequest(http.MethodGet, "/v1/users/"+requestedID.Hex()+"/orders", nil)
	request.Header.Set("Authorization", "Bearer "+issueTestToken(t, users.User{ID: adminID, Email: "admin@example.com"}))
	response := httptest.NewRecorder()
	testRouterWithUsers(fakeProducts{}, fakeUsers{
		get: func(id bson.ObjectID) (users.User, error) {
			if id == adminID {
				return users.User{ID: id, Email: "admin@example.com", IsAdmin: true}, nil
			}
			if id == requestedID {
				return users.User{ID: id, Email: "other@example.com", Orders: []bson.ObjectID{orderID}}, nil
			}
			return users.User{ID: id, Orders: []bson.ObjectID{orderID}}, nil
		},
	}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), orderID.Hex()) {
		t.Fatalf("body = %q, want order ID", response.Body.String())
	}
}

func TestLoginReturnsToken(t *testing.T) {
	userID := bson.NewObjectID()
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"identity_token":"google-identity-token"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	testRouterWithAuth(fakeProducts{}, fakeUsers{getByEmail: func(email string) (users.User, error) {
		return users.User{ID: userID, Name: "Ada", Email: email, Orders: []bson.ObjectID{}, IsAdmin: true}, nil
	}}, fakeGoogleTokens{validate: func(token string) (users.User, error) {
		if token != "google-identity-token" {
			return users.User{}, auth.ErrInvalidIdentityToken
		}
		return users.User{Email: "ada@example.com"}, nil
	}}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var payload struct {
		Token string `json:"token"`
		User  struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if payload.Token == "" || payload.User.ID != userID.Hex() {
		t.Fatalf("login payload = %#v, want token and user", payload)
	}
}

func TestLoginCreatesUserWhenMissing(t *testing.T) {
	createdUserID := bson.NewObjectID()
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"identity_token":"google-identity-token"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	testRouterWithAuth(fakeProducts{}, fakeUsers{
		getByEmail: func(string) (users.User, error) {
			return users.User{}, users.ErrNotFound
		},
		create: func(input users.CreateInput) (users.User, error) {
			if input.Name != "Ada Lovelace" || input.Email != "ada@example.com" || input.IsAdmin {
				t.Fatalf("Create() input = %#v, want Google profile data and non-admin", input)
			}
			return users.User{ID: createdUserID, Name: input.Name, Email: input.Email, Orders: []bson.ObjectID{}}, nil
		},
	}, fakeGoogleTokens{validate: func(string) (users.User, error) {
		return users.User{Name: "Ada Lovelace", Email: "ada@example.com"}, nil
	}}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var payload struct {
		Token string `json:"token"`
		User  struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if payload.Token == "" || payload.User.ID != createdUserID.Hex() {
		t.Fatalf("login payload = %#v, want token and created user", payload)
	}
}

func TestLoginRejectsUnknownUserWhenCreateFails(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"identity_token":"google-identity-token"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	testRouterWithAuth(fakeProducts{}, fakeUsers{
		getByEmail: func(string) (users.User, error) {
			return users.User{}, users.ErrNotFound
		},
		create: func(users.CreateInput) (users.User, error) {
			return users.User{}, users.ErrInvalid
		},
	}, fakeGoogleTokens{validate: func(string) (users.User, error) {
		return users.User{Name: "Ada Lovelace", Email: "missing@example.com"}, nil
	}}).ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnprocessableEntity)
	}
}

func TestLoginRejectsInvalidGoogleIdentityToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"identity_token":"bad-token"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	testRouterWithAuth(fakeProducts{}, fakeUsers{}, fakeGoogleTokens{validate: func(string) (users.User, error) {
		return users.User{}, auth.ErrInvalidIdentityToken
	}}).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}
