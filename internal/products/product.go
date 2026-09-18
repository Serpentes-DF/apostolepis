package products

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	ErrDuplicate = errors.New("product already exists")
	ErrInvalid   = errors.New("invalid product")
	ErrNotFound  = errors.New("product not found")
)

type Product struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string        `bson:"name" json:"name"`
	SKU       string        `bson:"sku" json:"sku"`
	Price     float64       `bson:"price" json:"price"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at" json:"updated_at"`
}

type CreateInput struct {
	Name  string  `json:"name"`
	SKU   string  `json:"sku"`
	Price float64 `json:"price"`
}

type UpdateInput struct {
	Name  string  `json:"name"`
	SKU   string  `json:"sku"`
	Price float64 `json:"price"`
}

func normalize(name, sku string, price float64) (string, string, error) {
	name = strings.TrimSpace(name)
	sku = strings.ToUpper(strings.TrimSpace(sku))
	if name == "" {
		return "", "", fmt.Errorf("%w: name is required", ErrInvalid)
	}
	if sku == "" {
		return "", "", fmt.Errorf("%w: SKU is required", ErrInvalid)
	}
	if price <= 0 {
		return "", "", fmt.Errorf("%w: price must be positive", ErrInvalid)
	}
	return name, sku, nil
}
