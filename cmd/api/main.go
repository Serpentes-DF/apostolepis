package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Serpentes-DF/apostolepis/internal/auth"
	"github.com/Serpentes-DF/apostolepis/internal/config"
	httpapi "github.com/Serpentes-DF/apostolepis/internal/httpapi"
	"github.com/Serpentes-DF/apostolepis/internal/inventory"
	platformmongo "github.com/Serpentes-DF/apostolepis/internal/platform/mongodb"
	"github.com/Serpentes-DF/apostolepis/internal/products"
	"github.com/Serpentes-DF/apostolepis/internal/users"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	configuration, err := config.Load()
	if err != nil {
		return err
	}
	connectContext, cancelConnect := context.WithTimeout(context.Background(), configuration.ConnectTimeout)
	defer cancelConnect()
	client, err := platformmongo.Connect(connectContext, configuration.MongoURI)
	if err != nil {
		return err
	}
	defer client.Disconnect(context.Background())

	database := client.Database(configuration.MongoDatabase)
	productRepository := products.NewMongoRepository(database)
	inventoryRepository := inventory.NewMongoRepository(database)
	userRepository := users.NewMongoRepository(database)
	if err := productRepository.EnsureIndexes(connectContext); err != nil {
		return err
	}
	if err := inventoryRepository.EnsureIndexes(connectContext); err != nil {
		return err
	}
	if err := userRepository.EnsureIndexes(connectContext); err != nil {
		return err
	}
	authTokens, err := auth.NewTokenService(configuration.AuthTokenSecret, configuration.AuthTokenTTL)
	if err != nil {
		return err
	}
	googleIdentityTokens, err := auth.NewGoogleIdentityTokenValidator(configuration.GoogleClientID)
	if err != nil {
		return err
	}

	router := httpapi.NewRouter(httpapi.Dependencies{
		Products:     products.NewService(productRepository),
		Inventory:    inventory.NewService(inventoryRepository, productRepository),
		Users:        users.NewService(userRepository),
		Tokens:       authTokens,
		GoogleTokens: googleIdentityTokens,
		Ping:         func(ctx context.Context) error { return client.Ping(ctx, nil) },
	})
	server := &http.Server{Addr: configuration.Address, Handler: router, ReadHeaderTimeout: 5 * time.Second}
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- server.ListenAndServe() }()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-signals:
		shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelShutdown()
		return server.Shutdown(shutdownContext)
	}
	return nil
}
