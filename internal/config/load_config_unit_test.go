//go:build unit

package config

import (
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Run("loads defaults", func(t *testing.T) {
		t.Setenv("MONGO_URI", "mongodb://mongodb:27017")
		t.Setenv("API_ADDRESS", "")
		t.Setenv("MONGO_DATABASE", "")
		t.Setenv("MONGO_CONNECT_TIMEOUT", "")

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
		if config.ConnectTimeout != 10*time.Second {
			t.Errorf("ConnectTimeout = %v, want %v", config.ConnectTimeout, 10*time.Second)
		}
	})

	t.Run("requires Mongo URI", func(t *testing.T) {
		t.Setenv("MONGO_URI", "")

		if _, err := Load(); err == nil {
			t.Fatal("Load() error = nil, want an error")
		}
	})

	t.Run("rejects invalid timeout", func(t *testing.T) {
		t.Setenv("MONGO_URI", "mongodb://mongodb:27017")
		t.Setenv("MONGO_CONNECT_TIMEOUT", "0")

		if _, err := Load(); err == nil {
			t.Fatal("Load() error = nil, want an error")
		}
	})
}
