//go:build unit

package users

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakeRepository struct {
	create     func(User) (User, error)
	getByEmail func(string) (User, error)
}

func (repository fakeRepository) Create(_ context.Context, user User) (User, error) {
	return repository.create(user)
}
func (fakeRepository) Get(context.Context, bson.ObjectID) (User, error) { return User{}, nil }
func (repository fakeRepository) GetByEmail(_ context.Context, email string) (User, error) {
	if repository.getByEmail == nil {
		return User{}, ErrNotFound
	}
	return repository.getByEmail(email)
}
func (fakeRepository) List(context.Context) ([]User, error) { return nil, nil }

func TestCreateUser(t *testing.T) {
	service := NewService(fakeRepository{create: func(user User) (User, error) { return user, nil }})

	user, err := service.Create(context.Background(), CreateInput{Name: "  Ada ", Email: " ADA@EXAMPLE.COM ", IsAdmin: true})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if user.Name != "Ada" || user.Email != "ada@example.com" || !user.IsAdmin {
		t.Fatalf("Create() user = %#v, want normalized values", user)
	}
	if user.Orders == nil || len(user.Orders) != 0 {
		t.Fatalf("Create() orders = %#v, want an empty orders array", user.Orders)
	}
}

func TestCreateUserRejectsInvalidInput(t *testing.T) {
	service := NewService(fakeRepository{create: func(user User) (User, error) {
		t.Fatal("repository called for invalid input")
		return user, nil
	}})

	for name, input := range map[string]CreateInput{
		"missing name":  {Email: "ada@example.com"},
		"invalid email": {Name: "Ada", Email: "not-an-email"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := service.Create(context.Background(), input); err == nil {
				t.Fatal("Create() error = nil, want an error")
			}
		})
	}
}

func TestGetUserByEmailAppliesAdminRole(t *testing.T) {
	service := NewService(fakeRepository{getByEmail: func(email string) (User, error) {
		return User{Name: "Ada", Email: email, IsAdmin: true}, nil
	}})

	user, err := service.GetByEmail(context.Background(), " ADA@example.com ")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}
	if user.Email != "ada@example.com" || !user.IsAdmin {
		t.Fatalf("GetByEmail() user = %#v, want persisted admin user", user)
	}
}
