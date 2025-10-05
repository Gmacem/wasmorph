package tests

import (
	"net/http"
	"testing"
)

func TestIntegration(t *testing.T) {
	// Test against running server
	baseURL := "http://localhost:8080"

	t.Run("Health check", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/health")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}
