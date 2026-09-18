package inventory

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	ErrInvalidDelta      = errors.New("stock delta must not be zero")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrNotFound          = errors.New("inventory not found")
)

type Inventory struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	ProductID bson.ObjectID `bson:"product_id" json:"product_id"`
	Quantity  int64         `bson:"quantity" json:"quantity"`
	UpdatedAt time.Time     `bson:"updated_at" json:"updated_at"`
}
