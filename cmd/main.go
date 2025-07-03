package main

import (
	"auth-service/internal/config"
	"auth-service/internal/handler"
	"auth-service/internal/repository"
	"auth-service/internal/service"
	"fmt"
	"log"
	"net/http"
)

func main() {
	cfg := config.Load()

	// DB connection
	db := repository.NewPostgresDB(
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	defer db.Close()

	// Init repository, service, handler
	repo := repository.NewPostgresRepository(db)
	tokenService := service.NewTokenService(repo, cfg.JWTSecret, cfg.WebhookURL)
	handler := handler.NewHandler(tokenService)

	// Register routes
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server running on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %s", err)
	}
}
