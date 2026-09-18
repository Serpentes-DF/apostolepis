package products

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Repository interface {
	Create(context.Context, Product) (Product, error)
	Get(context.Context, bson.ObjectID) (Product, error)
	List(context.Context) ([]Product, error)
	Update(context.Context, Product) (Product, error)
	Delete(context.Context, bson.ObjectID) error
}

type Service struct {
	repository Repository
	now        func() time.Time
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, now: time.Now}
}

func (service *Service) Create(ctx context.Context, input CreateInput) (Product, error) {
	name, sku, err := normalize(input.Name, input.SKU, input.Price)
	if err != nil {
		return Product{}, err
	}
	now := service.now().UTC()
	return service.repository.Create(ctx, Product{
		Name: name, SKU: sku, Price: input.Price, CreatedAt: now, UpdatedAt: now,
	})
}

func (service *Service) Get(ctx context.Context, id bson.ObjectID) (Product, error) {
	return service.repository.Get(ctx, id)
}

func (service *Service) List(ctx context.Context) ([]Product, error) {
	return service.repository.List(ctx)
}

func (service *Service) Update(ctx context.Context, id bson.ObjectID, input UpdateInput) (Product, error) {
	name, sku, err := normalize(input.Name, input.SKU, input.Price)
	if err != nil {
		return Product{}, err
	}
	product, err := service.repository.Get(ctx, id)
	if err != nil {
		return Product{}, err
	}
	product.Name, product.SKU, product.Price = name, sku, input.Price
	product.UpdatedAt = service.now().UTC()
	return service.repository.Update(ctx, product)
}

func (service *Service) Delete(ctx context.Context, id bson.ObjectID) error {
	return service.repository.Delete(ctx, id)
}
