//go:build unit

package products

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakeRepository struct {
	create func(Product) (Product, error)
	get    func(bson.ObjectID) (Product, error)
	update func(Product) (Product, error)
}

func (repository fakeRepository) Create(_ context.Context, product Product) (Product, error) {
	return repository.create(product)
}
func (repository fakeRepository) Get(_ context.Context, id bson.ObjectID) (Product, error) {
	return repository.get(id)
}
func (repository fakeRepository) List(context.Context) ([]Product, error) { return nil, nil }
func (repository fakeRepository) Update(_ context.Context, product Product) (Product, error) {
	return repository.update(product)
}
func (repository fakeRepository) Delete(context.Context, bson.ObjectID) error { return nil }

func TestCreateProduct(t *testing.T) {
	fixedTime := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	repository := fakeRepository{create: func(product Product) (Product, error) {
		product.ID = bson.NewObjectID()
		return product, nil
	}}
	service := NewService(repository)
	service.now = func() time.Time { return fixedTime }

	product, err := service.Create(context.Background(), CreateInput{Name: "  Field Guide ", SKU: " fg-1 ", Price: 25})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if product.Name != "Field Guide" || product.SKU != "FG-1" {
		t.Fatalf("Create() product = %#v, want normalized name and SKU", product)
	}
	if product.CreatedAt != fixedTime || product.UpdatedAt != fixedTime {
		t.Fatalf("Create() timestamps = %v/%v, want %v", product.CreatedAt, product.UpdatedAt, fixedTime)
	}
}

func TestCreateProductRejectsInvalidInput(t *testing.T) {
	service := NewService(fakeRepository{create: func(product Product) (Product, error) {
		t.Fatal("repository called for invalid input")
		return product, nil
	}})

	for name, input := range map[string]CreateInput{
		"missing name":  {SKU: "SKU", Price: 1},
		"missing SKU":   {Name: "Product", Price: 1},
		"invalid price": {Name: "Product", SKU: "SKU", Price: 0},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := service.Create(context.Background(), input); err == nil {
				t.Fatal("Create() error = nil, want an error")
			}
		})
	}
}
