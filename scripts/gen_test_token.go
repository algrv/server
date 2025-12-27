package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/algorave/server/algorave/users"
	"github.com/algorave/server/internal/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// load environment
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// connect to database
	dbConnString := os.Getenv("SUPABASE_CONNECTION_STRING")
	if dbConnString == "" {
		log.Fatal("SUPABASE_CONNECTION_STRING not set")
	}

	dbPool, err := pgxpool.New(context.Background(), dbConnString)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	ctx := context.Background()
	_ = users.NewRepository(dbPool)

	// create or find test user
	testEmail := "test@algorave.dev"
	testProvider := "test"
	testProviderID := "test-user-123"
	var userID string

	// check if user exists
	err = dbPool.QueryRow(ctx, "SELECT id FROM users WHERE provider = $1 AND provider_id = $2", testProvider, testProviderID).Scan(&userID)

	if err != nil {
		// create test user
		userID = uuid.New().String()
		_, err = dbPool.Exec(ctx, `
			INSERT INTO users (id, email, provider, provider_id, name, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		`, userID, testEmail, testProvider, testProviderID, "Test User")

		if err != nil {
			log.Fatalf("Failed to create test user: %v", err)
		}
		fmt.Printf("✅ Created test user: %s (ID: %s)\n", testEmail, userID)
	} else {
		fmt.Printf("✅ Using existing test user (ID: %s)\n", userID)
	}

	// generate JWT token
	token, err := auth.GenerateJWT(userID, testEmail)
	if err != nil {
		log.Fatalf("Failed to generate JWT: %v", err)
	}

	fmt.Printf("\n🔑 Test JWT Token:\n%s\n\n", token)
	fmt.Printf("Export this token for testing:\nexport TEST_TOKEN=\"%s\"\n", token)
}
