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
	cache := wasm.NewRistrettoCache(&wasm.RuntimeCacheConfig{
		MaxCost:     100 << 20,
		NumCounters: 1000,
		BufferItems: 64,
	})
	wasmService := wasm.NewService(pool, cache)
	rulesHandler := handlers.NewRulesHandler(wasmService)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	r.Post("/api/v1/auth/login", authService.LoginHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authService.AuthMiddleware)

		r.Post("/rules", rulesHandler.CreateRule)
		r.Get("/rules", rulesHandler.ListRules)
		r.Get("/rules/{name}", rulesHandler.GetRule)
		r.Post("/rules/{name}/execute", rulesHandler.ExecuteRule)
		r.Delete("/rules/{name}", rulesHandler.DeleteRule)
	})

	fileServer := http.FileServer(http.Dir("web/static"))
	r.Handle("/*", http.StripPrefix("/", fileServer))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("Starting server", "port", port)
	slog.Error("Server stopped", "error", http.ListenAndServe(":"+port, r))
}
