package helpers

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

type DatabaseClient struct {
	db *sql.DB
}

func NewDatabaseClient() (*DatabaseClient, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &DatabaseClient{db: db}, nil
}

func (dc *DatabaseClient) Close() error {
	return dc.db.Close()
}

func (dc *DatabaseClient) GetDatabaseURL() string {
	return os.Getenv("DATABASE_URL")
}

func (dc *DatabaseClient) AddUser(username, passwordHash string) error {
	_, err := dc.db.Exec(`
		INSERT INTO wasmorph.users (username, password_hash, is_active) 
		VALUES ($1, $2, $3) 
		ON CONFLICT (username) DO NOTHING`,
		username, passwordHash, true)
	return err
}

func (dc *DatabaseClient) AddAPIKey(apiKey, username string) error {
	var userID int32
	err := dc.db.QueryRow(`
		SELECT id FROM wasmorph.users 
		WHERE username = $1`,
		username).Scan(&userID)
	if err != nil {
		return fmt.Errorf("failed to find user %s: %w", username, err)
	}

	_, err = dc.db.Exec(`
		INSERT INTO wasmorph.api_keys (api_key, user_id, is_active) 
		VALUES ($1, $2, $3)
		ON CONFLICT (api_key) DO NOTHING`,
		apiKey, userID, true)
	return err
}

func (dc *DatabaseClient) GetUserID(username string) (int32, error) {
	var userID int32
	err := dc.db.QueryRow(`
		SELECT id FROM wasmorph.users 
		WHERE username = $1`,
		username).Scan(&userID)
	return userID, err
}

func (dc *DatabaseClient) Cleanup() error {
	_, err := dc.db.Exec("DELETE FROM wasmorph.rules")
	if err != nil {
		return err
	}
	_, err = dc.db.Exec("DELETE FROM wasmorph.api_keys")
	if err != nil {
		return err
	}
	_, err = dc.db.Exec("DELETE FROM wasmorph.users")
	return err
}

func (dc *DatabaseClient) CleanupAll() error {
	// Clean up all tables in the correct order (respecting foreign key constraints)
	tables := []string{
		"wasmorph.rules",
		"wasmorph.api_keys",
		"wasmorph.users",
	}

	for _, table := range tables {
		_, err := dc.db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			return fmt.Errorf("failed to clean table %s: %w", table, err)
		}
	}

	return nil
}

func (dc *DatabaseClient) CleanupRules() error {
	_, err := dc.db.Exec("DELETE FROM wasmorph.rules")
	return err
}

func (dc *DatabaseClient) GetUserByUsername(username string) (*User, error) {
	var user User
	err := dc.db.QueryRow(`
		SELECT id, username, COALESCE(email, '') as email, password_hash, created_at, updated_at, is_active 
		FROM wasmorph.users 
		WHERE username = $1`,
		username).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt, &user.IsActive)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (dc *DatabaseClient) VerifyUserAndAPIKey(username, apiKey string) error {
	var userCount int
	err := dc.db.QueryRow("SELECT COUNT(*) FROM wasmorph.users WHERE username = $1", username).Scan(&userCount)
	if err != nil {
		return err
	}
	if userCount != 1 {
		return fmt.Errorf("user %s not found in database", username)
	}

	var apiKeyCount int
	err = dc.db.QueryRow("SELECT COUNT(*) FROM wasmorph.api_keys WHERE api_key = $1", apiKey).Scan(&apiKeyCount)
	if err != nil {
		return err
	}
	if apiKeyCount != 1 {
		return fmt.Errorf("API key %s not found in database", apiKey)
	}

	return nil
}

type User struct {
	ID           int32
	Username     string
	Email        string
	PasswordHash string
	CreatedAt    interface{}
	UpdatedAt    interface{}
	IsActive     bool
}
