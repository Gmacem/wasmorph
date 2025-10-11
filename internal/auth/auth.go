package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Gmacem/wasmorph/internal/sql"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
}

type AuthService struct {
	config       Config
	userSessions map[string]string
	queries      *sql.Queries
}

func NewAuthService(pool *pgxpool.Pool, config Config) *AuthService {
	return &AuthService{
		config:       config,
		userSessions: make(map[string]string),
		queries:      sql.New(pool),
	}
}

func (a *AuthService) GenerateAPIKey(userID string) string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	apiKey := hex.EncodeToString(bytes)
	return apiKey
}

func (a *AuthService) GenerateJWT(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.config.JWTSecret))
}

func (a *AuthService) ValidateJWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.config.JWTSecret), nil
	})
	if err != nil {
		return "", err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if userID, ok := claims["user_id"].(string); ok {
			return userID, nil
		}
	}
	return "", fmt.Errorf("invalid token")
}

func (a *AuthService) ValidateAPIKey(apiKey string) (int32, bool) {
	userID, err := a.queries.ValidateAPIKey(context.Background(), apiKey)
	if err != nil {
		return 0, false
	}

	return userID, true
}

func (a *AuthService) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			if after, ok := strings.CutPrefix(auth, "Bearer "); ok {
				apiKey := after
				if userID, exists := a.ValidateAPIKey(apiKey); exists {
					r.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
					next.ServeHTTP(w, r)
					return
				}
			}
		}
		if cookie, err := r.Cookie("session"); err == nil {
			if userID, err := a.ValidateJWT(cookie.Value); err == nil {
				r.Header.Set("X-User-ID", userID)
				next.ServeHTTP(w, r)
				return
			}
		}
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

func (a *AuthService) LoginHandler(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		http.Error(w, "Username and password required", http.StatusBadRequest)
		return
	}
	userID := username
	jwtToken, err := a.GenerateJWT(userID)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    jwtToken,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400,
	})
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"access_token": "%s"}`, jwtToken)
}
