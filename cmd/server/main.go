package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/Gmacem/wasmorph/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	authService := auth.NewAuthService()
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

		r.Post("/rules", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Header.Get("X-User-ID")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"message": "Rule created", "user": "` + userID + `"}`))
		})

		r.Post("/rules/{name}/execute", func(w http.ResponseWriter, r *http.Request) {
			name := chi.URLParam(r, "name")
			userID := r.Header.Get("X-User-ID")
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result": "echo: ` + name + `", "user": "` + userID + `"}`))
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("Starting server", "port", port)
	slog.Error("Server stopped", "error", http.ListenAndServe(":"+port, r))
}
