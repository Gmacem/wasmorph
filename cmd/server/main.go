package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/Gmacem/wasmorph/internal/auth"
	"github.com/Gmacem/wasmorph/internal/handlers"
	"github.com/Gmacem/wasmorph/internal/wasm"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	config := auth.Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}
	if config.DatabaseURL == "" {
		logger.Error("DATABASE_URL is not set")
		os.Exit(1)
	}
	if config.JWTSecret == "" {
		logger.Error("JWT_SECRET is not set")
		os.Exit(1)
	}

	// Create connection pool instead of single connection
	pool, err := pgxpool.New(context.Background(), config.DatabaseURL)
	if err != nil {
		logger.Error("Failed to create connection pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Create services with shared connection pool
	authService := auth.NewAuthService(pool, config)
	wasmService := wasm.NewService(pool)
	rulesHandler := handlers.NewRulesHandler(wasmService)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Wasmorph API"))
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	r.Post("/api/v1/auth/login", authService.LoginHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authService.AuthMiddleware)

		r.Post("/rules", rulesHandler.CreateRule)
		r.Get("/rules", rulesHandler.ListRules)
		r.Post("/rules/{name}/execute", rulesHandler.ExecuteRule)
		r.Delete("/rules/{name}", rulesHandler.DeleteRule)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("Starting server", "port", port)
	slog.Error("Server stopped", "error", http.ListenAndServe(":"+port, r))
}
