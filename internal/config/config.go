package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAddress        = ":8080"
	databaseName          = "apostolepis"
	defaultConnectTimeout = 20 * time.Second
	defaultTokenTTL       = 24 * time.Hour
)

type Config struct {
	Address         string
	MongoURI        string
	MongoDatabase   string
	ConnectTimeout  time.Duration
	AuthTokenSecret string
	AuthTokenTTL    time.Duration
	GoogleClientID  string
}

func Load() (Config, error) {
	timeout, err := durationFromEnv("MONGO_CONNECT_TIMEOUT", defaultConnectTimeout)
	if err != nil {
		return Config{}, err
	}
	tokenTTL, err := durationFromEnv("AUTH_TOKEN_TTL", defaultTokenTTL)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		Address:         resolveAddress(),
		MongoURI:        os.Getenv("MONGO_URI"),
		MongoDatabase:   databaseName,
		ConnectTimeout:  timeout,
		AuthTokenSecret: strings.TrimSpace(os.Getenv("AUTH_TOKEN_SECRET")),
		AuthTokenTTL:    tokenTTL,
		GoogleClientID:  strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
	}

	if config.MongoURI == "" {
		return Config{}, fmt.Errorf("MONGO_URI is required")
	}
	if config.AuthTokenSecret == "" {
		return Config{}, fmt.Errorf("AUTH_TOKEN_SECRET is required")
	}
	if config.GoogleClientID == "" {
		return Config{}, fmt.Errorf("GOOGLE_CLIENT_ID is required")
	}

	return config, nil
}

func resolveAddress() string {
	if address := os.Getenv("API_ADDRESS"); address != "" {
		return address
	}

	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		return ":" + port
	}

	return defaultAddress
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return 0, fmt.Errorf("%s must be a positive number of seconds", key)
	}
	return time.Duration(seconds) * time.Second, nil
}
