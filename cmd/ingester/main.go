package main

import (
	"context"
	"fmt"
	"os"

	"codeberg.org/algorave/server/internal/config"
	"codeberg.org/algorave/server/internal/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ingester <command> [options]")
		fmt.Println("Commands:")
		fmt.Println("  docs      - ingest documentation from markdown files")
		fmt.Println("  concepts  - ingest teaching concepts from MDX files")
		fmt.Println("  all       - ingest everything (docs, concepts)")
		fmt.Println("\nOptions:")
		fmt.Println("  --path <path>  - Custom path to ingest from")
		fmt.Println("  --clear        - Clear existing data before ingesting")
		os.Exit(1)
	}

	command := os.Args[1]

	// load environment variables
	cfg, err := config.LoadEnvironmentVariables()
	if err != nil {
		logger.Fatal("failed to load configuration", "error", err)
	}

	// connect to database
	ctx := context.Background()
	db, err := pgxpool.New(ctx, cfg.SupabaseConnString)
	if err != nil {
		logger.Fatal("failed to create database pool", "error", err)
	}

	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		logger.Fatal("failed to ping database", "error", err)
	}

	logger.Info("connected to database")

	// route to appropriate command
	switch command {
	case "docs":
		flags := config.ParseDocsFlags()
		if err := IngestDocs(cfg, db, flags); err != nil {
			logger.Fatal("failed to ingest docs", "error", err)
		}

	case "concepts":
		flags := config.ParseConceptsFlags()
		if err := IngestConcepts(cfg, db, flags); err != nil {
			logger.Fatal("failed to ingest concepts", "error", err)
		}

	case "all":
		// use default flags for all subcommands
		docsFlags := config.DefaultDocsFlags()
		conceptsFlags := config.DefaultConceptsFlags()

		// check for --clear flag
		for _, arg := range os.Args[2:] {
			if arg == "--clear" {
				docsFlags.Clear = true
				conceptsFlags.Clear = true
			}
		}

		logger.Info("ingesting all data (docs, concepts)")

		if err := IngestDocs(cfg, db, docsFlags); err != nil {
			logger.Fatal("failed to ingest docs", "error", err)
		}

		if err := IngestConcepts(cfg, db, conceptsFlags); err != nil {
			logger.Fatal("failed to ingest concepts", "error", err)
		}

		logger.Info("successfully ingested all data")

	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}
