package inventory

import (
	"context"

	"github.com/Serpentes-DF/apostolepis/internal/products"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Repository interface {
	Adjust(context.Context, bson.ObjectID, int64) (Inventory, bool, error)
	Get(context.Context, bson.ObjectID) (Inventory, error)
}

type ProductLookup interface {
	Exists(context.Context, bson.ObjectID) (bool, error)
}

type Service struct {
	repository Repository
	products   ProductLookup
}

func NewService(repository Repository, products ProductLookup) *Service {
	return &Service{repository: repository, products: products}
}

func (service *Service) Adjust(ctx context.Context, productID bson.ObjectID, delta int64) (Inventory, error) {
	if delta == 0 {
		return Inventory{}, ErrInvalidDelta
	}
	exists, err := service.products.Exists(ctx, productID)
	if err != nil {
		return Inventory{}, err
	}
	if !exists {
		return Inventory{}, products.ErrNotFound
	}
	inventory, adjusted, err := service.repository.Adjust(ctx, productID, delta)
	if err != nil {
		return Inventory{}, err
	}
	if !adjusted {
		return Inventory{}, ErrInsufficientStock
	}
	return inventory, nil
}

func (service *Service) Get(ctx context.Context, productID bson.ObjectID) (Inventory, error) {
	return service.repository.Get(ctx, productID)
}
