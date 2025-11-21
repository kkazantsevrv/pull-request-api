package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"pull-request-api.com/internal/api"
	database "pull-request-api.com/internal/repository"
	"pull-request-api.com/internal/service"
)

const (
	dbHost     = "postgres"
	dbPort     = "5432"
	dbUser     = "postgres"
	dbPassword = "postgres"
	dbName     = "prdb"
)

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	dbConn, err := database.Connect(psqlInfo)
	if err != nil {
		log.Fatalf("Infrastructure initialization failed: %v", err)
	}
	defer dbConn.Close()

	if err := database.Migrate(dbConn, dbName); err != nil {
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
