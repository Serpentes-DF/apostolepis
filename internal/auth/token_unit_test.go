//go:build unit

package auth

import (
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestTokenServiceIssueAndParse(t *testing.T) {
	service, err := NewTokenService("secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	userID := bson.NewObjectID()
	token, err := service.Issue(userID, "Ada@Example.com")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	identity, err := service.Parse(token)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if identity.UserID != userID || identity.Email != "ada@example.com" {
		t.Fatalf("Parse() identity = %#v, want normalized email and same user ID", identity)
	}
}

func TestTokenServiceRejectsExpiredToken(t *testing.T) {
	service, err := NewTokenService("secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	token, err := service.Issue(bson.NewObjectID(), "ada@example.com")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	service.now = func() time.Time { return now.Add(2 * time.Hour) }
	if _, err := service.Parse(token); !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("Parse() error = %v, want %v", err, ErrExpiredToken)
	}
}
