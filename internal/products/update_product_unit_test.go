//go:build unit

package products

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestUpdateProduct(t *testing.T) {
	productID := bson.NewObjectID()
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	repository := fakeRepository{
		get: func(id bson.ObjectID) (Product, error) {
			return Product{ID: id, Name: "Old", SKU: "OLD", Price: 1, CreatedAt: createdAt}, nil
		},
		update: func(product Product) (Product, error) { return product, nil },
	}
	service := NewService(repository)
	service.now = func() time.Time { return updatedAt }

	product, err := service.Update(context.Background(), productID, UpdateInput{Name: " New ", SKU: " new-1 ", Price: 2})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if product.Name != "New" || product.SKU != "NEW-1" || product.Price != 2 {
		t.Fatalf("Update() product = %#v, want updated normalized values", product)
	}
	if product.CreatedAt != createdAt || product.UpdatedAt != updatedAt {
		t.Fatalf("Update() timestamps = %v/%v", product.CreatedAt, product.UpdatedAt)
	}
}

func TestUpdateProductPropagatesNotFound(t *testing.T) {
	service := NewService(fakeRepository{
		get: func(bson.ObjectID) (Product, error) { return Product{}, ErrNotFound },
	})

	_, err := service.Update(context.Background(), bson.NewObjectID(), UpdateInput{Name: "New", SKU: "NEW", Price: 2})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, ErrNotFound)
	}
}
