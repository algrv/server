package retriever

import (
	"fmt"
	"os"
	"strconv"
)

// loadRetrieverConfig loads configuration from environment variables
func loadRetrieverConfig() (*RetrieverConfig, error) {
	// database connection
	dbConnString := os.Getenv("SUPABASE_CONNECTION_STRING")
	if dbConnString == "" {
		return nil, fmt.Errorf("SUPABASE_CONNECTION_STRING environment variable is required")
	}

	// optional: top K for retrieval
	topK := defaultTopK // default from utils.go
	if topKStr := os.Getenv("RETRIEVAL_TOP_K"); topKStr != "" {
		if val, err := strconv.Atoi(topKStr); err == nil {
			topK = val
		}
	}

	return &RetrieverConfig{
		DBConnString: dbConnString,
		TopK:         topK,
	}, nil
}
