package users

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Repository interface {
	Create(context.Context, User) (User, error)
	Get(context.Context, bson.ObjectID) (User, error)
	GetByEmail(context.Context, string) (User, error)
	List(context.Context) ([]User, error)
}

type Service struct {
	repository Repository
	now        func() time.Time
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, now: time.Now}
}

func withOrders(user User) User {
	if user.Orders == nil {
		user.Orders = []bson.ObjectID{}
	}
	return user
}

func (service *Service) Create(ctx context.Context, input CreateInput) (User, error) {
	input, err := normalize(input)
	if err != nil {
		return User{}, err
	}
	user, err := service.repository.Create(ctx, User{
		Name:      input.Name,
		Email:     input.Email,
		Orders:    []bson.ObjectID{},
		IsAdmin:   input.IsAdmin,
		CreatedAt: service.now().UTC(),
	})
	if err != nil {
		return User{}, err
	}
	return withOrders(user), nil
}

func (service *Service) Get(ctx context.Context, id bson.ObjectID) (User, error) {
	user, err := service.repository.Get(ctx, id)
	if err != nil {
		return User{}, err
	}
	return withOrders(user), nil
}

func (service *Service) GetByEmail(ctx context.Context, email string) (User, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return User{}, ErrNotFound
	}
	user, err := service.repository.GetByEmail(ctx, normalized)
	if err != nil {
		return User{}, err
	}
	return withOrders(user), nil
}

func (service *Service) List(ctx context.Context) ([]User, error) {
	users, err := service.repository.List(ctx)
	if err != nil {
		return nil, err
	}
	for index := range users {
		users[index] = withOrders(users[index])
	}
	return users, nil
}
