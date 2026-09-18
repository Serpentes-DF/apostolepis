package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/Serpentes-DF/apostolepis/internal/users"
	googleidtoken "google.golang.org/api/idtoken"
)

var ErrInvalidIdentityToken = errors.New("invalid identity token")

type GoogleIdentityTokenValidator struct {
	audience string
	validate func(context.Context, string, string) (*googleidtoken.Payload, error)
}

func NewGoogleIdentityTokenValidator(audience string) (*GoogleIdentityTokenValidator, error) {
	trimmed := strings.TrimSpace(audience)
	if trimmed == "" {
		return nil, errors.New("google client id is required")
	}
	return &GoogleIdentityTokenValidator{audience: trimmed, validate: googleidtoken.Validate}, nil
}

func (validator *GoogleIdentityTokenValidator) Validate(ctx context.Context, token string) (users.User, error) {
	payload, err := validator.validate(ctx, strings.TrimSpace(token), validator.audience)
	if err != nil {
		return users.User{}, ErrInvalidIdentityToken
	}
	email, _ := payload.Claims["email"].(string)
	if email == "" || !verifiedEmail(payload.Claims["email_verified"]) {
		return users.User{}, ErrInvalidIdentityToken
	}
	name := strings.TrimSpace(claimString(payload.Claims["name"]))
	if name == "" {
		name = defaultName(email)
	}
	return users.User{Name: name, Email: strings.ToLower(email)}, nil
}

func claimString(value any) string {
	text, _ := value.(string)
	return text
}

func defaultName(email string) string {
	localPart, _, found := strings.Cut(strings.TrimSpace(email), "@")
	if !found || localPart == "" {
		return strings.TrimSpace(email)
	}
	return localPart
}

func verifiedEmail(value any) bool {
	verified, ok := value.(bool)
	if ok {
		return verified
	}
	verifiedString, ok := value.(string)
	return ok && strings.EqualFold(verifiedString, "true")
}
