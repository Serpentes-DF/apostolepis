//go:build unit

package config

import (
	"testing"
)

func TestLoad(t *testing.T) {
	t.Run("loads defaults", func(t *testing.T) {
		t.Setenv("MONGO_URI", "mongodb://mongodb:27017")
		t.Setenv("AUTH_TOKEN_SECRET", "test-secret")
		t.Setenv("GOOGLE_CLIENT_ID", "test-client-id")
		t.Setenv("API_ADDRESS", "")
		t.Setenv("PORT", "")
		t.Setenv("MONGO_DATABASE", "")
		t.Setenv("MONGO_CONNECT_TIMEOUT", "")
		t.Setenv("AUTH_TOKEN_TTL", "")

		config, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if config.Address != ":8080" {
			t.Errorf("Address = %q, want %q", config.Address, ":8080")
		}
		if config.MongoDatabase != "apostolepis" {
			t.Errorf("MongoDatabase = %q, want %q", config.MongoDatabase, "apostolepis")
		}
		if config.ConnectTimeout != defaultConnectTimeout {
			t.Errorf("ConnectTimeout = %v, want %v", config.ConnectTimeout, defaultConnectTimeout)
		}
		if config.AuthTokenTTL != defaultTokenTTL {
			t.Errorf("AuthTokenTTL = %v, want %v", config.AuthTokenTTL, defaultTokenTTL)
		}
	})

	t.Run("uses platform port when address is unset", func(t *testing.T) {
		t.Setenv("MONGO_URI", "mongodb://mongodb:27017")
		t.Setenv("AUTH_TOKEN_SECRET", "test-secret")
		t.Setenv("GOOGLE_CLIENT_ID", "test-client-id")
		t.Setenv("API_ADDRESS", "")
		t.Setenv("PORT", "9090")

		config, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if config.Address != ":9090" {
			t.Errorf("Address = %q, want %q", config.Address, ":9090")
		}
	})

	t.Run("prefers explicit API address over platform port", func(t *testing.T) {
		t.Setenv("MONGO_URI", "mongodb://mongodb:27017")
		t.Setenv("AUTH_TOKEN_SECRET", "test-secret")
		t.Setenv("GOOGLE_CLIENT_ID", "test-client-id")
		t.Setenv("API_ADDRESS", ":7000")
		t.Setenv("PORT", "9090")

		config, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if config.Address != ":7000" {
			t.Errorf("Address = %q, want %q", config.Address, ":7000")
		}
	})

	t.Run("requires Mongo URI", func(t *testing.T) {
		t.Setenv("MONGO_URI", "")
		t.Setenv("AUTH_TOKEN_SECRET", "test-secret")
		t.Setenv("GOOGLE_CLIENT_ID", "test-client-id")

		if _, err := Load(); err == nil {
			t.Fatal("Load() error = nil, want an error")
		}
	})

	t.Run("requires auth token secret", func(t *testing.T) {
		t.Setenv("MONGO_URI", "mongodb://mongodb:27017")
		t.Setenv("AUTH_TOKEN_SECRET", "")
		t.Setenv("GOOGLE_CLIENT_ID", "test-client-id")

		if _, err := Load(); err == nil {
			t.Fatal("Load() error = nil, want an error")
		}
	})

	t.Run("requires google client id", func(t *testing.T) {
		t.Setenv("MONGO_URI", "mongodb://mongodb:27017")
		t.Setenv("AUTH_TOKEN_SECRET", "test-secret")
		t.Setenv("GOOGLE_CLIENT_ID", "")

		if _, err := Load(); err == nil {
			t.Fatal("Load() error = nil, want an error")
		}
	})

	t.Run("rejects invalid timeout", func(t *testing.T) {
		t.Setenv("MONGO_URI", "mongodb://mongodb:27017")
		t.Setenv("AUTH_TOKEN_SECRET", "test-secret")
		t.Setenv("GOOGLE_CLIENT_ID", "test-client-id")
		t.Setenv("MONGO_CONNECT_TIMEOUT", "0")

		if _, err := Load(); err == nil {
			t.Fatal("Load() error = nil, want an error")
		}
	})

	t.Run("rejects invalid token ttl", func(t *testing.T) {
		t.Setenv("MONGO_URI", "mongodb://mongodb:27017")
		t.Setenv("AUTH_TOKEN_SECRET", "test-secret")
		t.Setenv("GOOGLE_CLIENT_ID", "test-client-id")
		t.Setenv("AUTH_TOKEN_TTL", "0")

		if _, err := Load(); err == nil {
			t.Fatal("Load() error = nil, want an error")
		}
	})
}
