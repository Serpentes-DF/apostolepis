//go:build unit

package inventory

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakeInventoryRepository struct {
	adjusted bool
}

func (repository fakeInventoryRepository) Adjust(_ context.Context, id bson.ObjectID, delta int64) (Inventory, bool, error) {
	return Inventory{ProductID: id, Quantity: delta}, repository.adjusted, nil
}
func (fakeInventoryRepository) Get(context.Context, bson.ObjectID) (Inventory, error) {
	return Inventory{}, nil
}

type fakeProductLookup struct{ exists bool }

func (lookup fakeProductLookup) Exists(context.Context, bson.ObjectID) (bool, error) {
	return lookup.exists, nil
}

func TestAdjustStock(t *testing.T) {
	productID := bson.NewObjectID()
	service := NewService(fakeInventoryRepository{adjusted: true}, fakeProductLookup{exists: true})

	stock, err := service.Adjust(context.Background(), productID, 5)
	if err != nil {
		t.Fatalf("Adjust() error = %v", err)
	}
	if stock.Quantity != 5 {
		t.Fatalf("Adjust() quantity = %d, want 5", stock.Quantity)
	}
}

func TestAdjustStockRejectsInvalidOperations(t *testing.T) {
	productID := bson.NewObjectID()
	tests := map[string]struct {
		service *Service
		delta   int64
		want    error
	}{
		"zero delta":         {NewService(fakeInventoryRepository{}, fakeProductLookup{exists: true}), 0, ErrInvalidDelta},
		"missing product":    {NewService(fakeInventoryRepository{}, fakeProductLookup{}), 1, errors.New("product not found")},
		"insufficient stock": {NewService(fakeInventoryRepository{}, fakeProductLookup{exists: true}), -1, ErrInsufficientStock},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := test.service.Adjust(context.Background(), productID, test.delta)
			if err == nil || err.Error() != test.want.Error() {
				t.Fatalf("Adjust() error = %v, want %v", err, test.want)
			}
		})
	}
}
