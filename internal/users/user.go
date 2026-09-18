package users

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	ErrDuplicate = errors.New("user already exists")
	ErrInvalid   = errors.New("invalid user")
	ErrNotFound  = errors.New("user not found")
)

type User struct {
	ID        bson.ObjectID   `bson:"_id,omitempty" json:"id"`
	Name      string          `bson:"name" json:"name"`
	Email     string          `bson:"email" json:"email"`
	Orders    []bson.ObjectID `bson:"orders" json:"orders"`
	IsAdmin   bool            `bson:"is_admin" json:"is_admin"`
	CreatedAt time.Time       `bson:"created_at" json:"created_at"`
}

type CreateInput struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
}

func normalize(input CreateInput) (CreateInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if input.Name == "" {
		return CreateInput{}, fmt.Errorf("%w: name is required", ErrInvalid)
	}
	address, err := mail.ParseAddress(input.Email)
	if err != nil || address.Address != input.Email {
		return CreateInput{}, fmt.Errorf("%w: valid email is required", ErrInvalid)
	}
	return input, nil
}
