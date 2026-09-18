//go:build unit

package auth

import (
	"context"
	"errors"
	"testing"

	googleidtoken "google.golang.org/api/idtoken"
)

func TestGoogleIdentityTokenValidatorValidate(t *testing.T) {
	validator, err := NewGoogleIdentityTokenValidator("client-id")
	if err != nil {
		t.Fatalf("NewGoogleIdentityTokenValidator() error = %v", err)
	}
	validator.validate = func(context.Context, string, string) (*googleidtoken.Payload, error) {
		return &googleidtoken.Payload{Claims: map[string]any{"email": "Ada@Example.com", "email_verified": true, "name": "Ada Lovelace"}}, nil
	}

	identity, err := validator.Validate(context.Background(), "google-token")
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if identity.Email != "ada@example.com" || identity.Name != "Ada Lovelace" {
		t.Fatalf("Validate() user = %#v, want normalized email and Google name", identity)
	}
}

func TestGoogleIdentityTokenValidatorFallsBackToEmailLocalPartForName(t *testing.T) {
	validator, err := NewGoogleIdentityTokenValidator("client-id")
	if err != nil {
		t.Fatalf("NewGoogleIdentityTokenValidator() error = %v", err)
	}
	validator.validate = func(context.Context, string, string) (*googleidtoken.Payload, error) {
		return &googleidtoken.Payload{Claims: map[string]any{"email": "ada@example.com", "email_verified": true}}, nil
	}

	identity, err := validator.Validate(context.Background(), "google-token")
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if identity.Name != "ada" {
		t.Fatalf("Validate() name = %q, want %q", identity.Name, "ada")
	}
}

func TestGoogleIdentityTokenValidatorRejectsUnverifiedEmail(t *testing.T) {
	validator, err := NewGoogleIdentityTokenValidator("client-id")
	if err != nil {
		t.Fatalf("NewGoogleIdentityTokenValidator() error = %v", err)
	}
	validator.validate = func(context.Context, string, string) (*googleidtoken.Payload, error) {
		return &googleidtoken.Payload{Claims: map[string]any{"email": "ada@example.com", "email_verified": false}}, nil
	}

	if _, err := validator.Validate(context.Background(), "google-token"); !errors.Is(err, ErrInvalidIdentityToken) {
		t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidIdentityToken)
	}
}
