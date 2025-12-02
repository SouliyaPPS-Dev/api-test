package main

import (
	"context"
	"log"

	"backoffice/backend/internal/config"
	"backoffice/backend/internal/infrastructure/postgres"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := postgres.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(context.Background()); err != nil {
		log.Fatalf("failed to run database migrations: %v", err)
	}

	log.Println("database migrations completed successfully")
}
