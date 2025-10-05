package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthService struct {
	jwtSecret    string
	apiKeys      map[string]string
	userSessions map[string]string
}

func NewAuthService() *AuthService {
	return &AuthService{
		jwtSecret:    "your-secret-key-change-in-production",
		apiKeys:      make(map[string]string),
		userSessions: make(map[string]string),
	}
}

func (a *AuthService) GenerateAPIKey(userID string) string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	apiKey := hex.EncodeToString(bytes)
	a.apiKeys[apiKey] = userID
	return apiKey
}

func (a *AuthService) GenerateJWT(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.jwtSecret))
}

func (a *AuthService) ValidateJWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.jwtSecret), nil
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

func (a *AuthService) ValidateAPIKey(apiKey string) (string, bool) {
	userID, exists := a.apiKeys[apiKey]
	return userID, exists
}

func (a *AuthService) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			if strings.HasPrefix(auth, "Bearer ") {
				token := strings.TrimPrefix(auth, "Bearer ")
				if userID, err := a.ValidateJWT(token); err == nil {
					r.Header.Set("X-User-ID", userID)
					next.ServeHTTP(w, r)
					return
				}
				if userID, exists := a.ValidateAPIKey(token); exists {
					r.Header.Set("X-User-ID", userID)
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
	apiKey := a.GenerateAPIKey(userID)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    jwtToken,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400,
	})
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"api_key": "%s", "access_token": "%s"}`, apiKey, jwtToken)
}
