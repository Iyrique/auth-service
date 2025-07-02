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

	// Подключение к БД
	db := repository.NewPostgresDB(
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	defer db.Close()

	// Инициализация репозитория, сервиса и хендлеров
	repo := repository.NewPostgresRepository(db)
	tokenService := service.NewTokenService(repo, cfg.JWTSecret, cfg.WebhookURL)
	handler := handler.NewHandler(tokenService)

	// Регистрация маршрутов
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Запуск HTTP-сервера
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server running on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %s", err)
	}
}
