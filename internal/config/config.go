package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultAddress        = ":8080"
	defaultDatabase       = "apostolepis"
	defaultConnectTimeout = 10 * time.Second
)

type Config struct {
	Address        string
	MongoURI       string
	MongoDatabase  string
	ConnectTimeout time.Duration
}

func Load() (Config, error) {
	timeout, err := durationFromEnv("MONGO_CONNECT_TIMEOUT", defaultConnectTimeout)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		Address:        valueOrDefault("API_ADDRESS", defaultAddress),
		MongoURI:       os.Getenv("MONGO_URI"),
		MongoDatabase:  valueOrDefault("MONGO_DATABASE", defaultDatabase),
		ConnectTimeout: timeout,
	}

	if config.MongoURI == "" {
		return Config{}, fmt.Errorf("MONGO_URI is required")
	}

	return config, nil
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
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
