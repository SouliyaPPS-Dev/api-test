package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"backoffice/backend/internal/config"
	"backoffice/backend/internal/httpserver"
	"backoffice/backend/internal/infrastructure/postgres"
	"backoffice/backend/internal/infrastructure/token"
	authusecase "backoffice/backend/internal/usecase/auth"
	productusecase "backoffice/backend/internal/usecase/product"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	rootCtx := context.Background()
	db, err := postgres.New(rootCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()
	if err := db.Migrate(rootCtx); err != nil {
		log.Fatalf("failed to run database migrations: %v", err)
	}

	tokenManager := token.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiry, cfg.JWTIssuer)

	authService := authusecase.NewService(postgres.NewUserRepository(db.Pool), tokenManager)
	productService := productusecase.NewService(postgres.NewProductRepository(db.Pool))

	server := httpserver.NewServer(cfg, authService, productService)
	log.Printf("HTTP server listening on %s", server.Addr())

	go func() {
		if err := server.Start(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				log.Printf("HTTP server closed: %v", err)
				return
			}
			log.Fatalf("server error: %v", err)
		}
		log.Printf("HTTP server stopped accepting new connections")
	}()

	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-shutdownCtx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v\n", err)
	} else {
		log.Printf("graceful shutdown completed")
	}
}
