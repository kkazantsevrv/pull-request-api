package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"pull-request-api.com/internal/api"
	database "pull-request-api.com/internal/database"
	"pull-request-api.com/internal/service"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "prdb")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	dbConn, err := database.Connect(psqlInfo)
	if err != nil {
		log.Fatalf("Infrastructure initialization failed: %v", err)
	}
	defer dbConn.Close()

	if err := database.Migrate(dbConn, dbName, "file://migrations"); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	ser := service.NewService(dbConn)
	server := api.NewServer(ser)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	api.HandlerFromMux(server, r)
	slog.Info("Server starting on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
