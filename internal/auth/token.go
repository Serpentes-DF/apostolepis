package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
)

type Identity struct {
	UserID bson.ObjectID
	Email  string
}

type TokenService struct {
	secret []byte
	now    func() time.Time
	ttl    time.Duration
}

type tokenHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type tokenClaims struct {
	Subject string `json:"sub"`
	Email   string `json:"email"`
	Expires int64  `json:"exp"`
}

func NewTokenService(secret string, ttl time.Duration) (*TokenService, error) {
	trimmed := strings.TrimSpace(secret)
	if trimmed == "" {
		return nil, errors.New("token secret is required")
	}
	if ttl <= 0 {
		return nil, errors.New("token ttl must be positive")
	}
	return &TokenService{secret: []byte(trimmed), now: time.Now, ttl: ttl}, nil
}

func (service *TokenService) Issue(userID bson.ObjectID, email string) (string, error) {
	if userID.IsZero() {
		return "", ErrInvalidToken
	}
	headerJSON, err := json.Marshal(tokenHeader{Algorithm: "HS256", Type: "JWT"})
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(tokenClaims{
		Subject: userID.Hex(),
		Email:   strings.ToLower(strings.TrimSpace(email)),
		Expires: service.now().Add(service.ttl).Unix(),
	})
	if err != nil {
		return "", err
	}
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsJSON)
	message := encodedHeader + "." + encodedClaims
	signature := service.sign(message)
	return message + "." + signature, nil
}

func (service *TokenService) Parse(token string) (Identity, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Identity{}, ErrInvalidToken
	}
	message := parts[0] + "." + parts[1]
	expected := service.sign(message)
	if !hmac.Equal([]byte(parts[2]), []byte(expected)) {
		return Identity{}, ErrInvalidToken
	}
	claimsPayload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Identity{}, ErrInvalidToken
	}
	var claims tokenClaims
	if err := json.Unmarshal(claimsPayload, &claims); err != nil {
		return Identity{}, ErrInvalidToken
	}
	if claims.Expires <= service.now().Unix() {
		return Identity{}, ErrExpiredToken
	}
	userID, err := bson.ObjectIDFromHex(claims.Subject)
	if err != nil {
		return Identity{}, ErrInvalidToken
	}
	if claims.Email == "" {
		return Identity{}, ErrInvalidToken
	}
	return Identity{UserID: userID, Email: claims.Email}, nil
}

func (service *TokenService) sign(message string) string {
	mac := hmac.New(sha256.New, service.secret)
	_, _ = mac.Write([]byte(message))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func BearerToken(header string) (string, error) {
	trimmed := strings.TrimSpace(header)
	if trimmed == "" {
		return "", fmt.Errorf("%w: missing Authorization header", ErrInvalidToken)
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(trimmed, prefix) {
		return "", ErrInvalidToken
	}
	token := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
	if token == "" {
		return "", ErrInvalidToken
	}
	return token, nil
}
