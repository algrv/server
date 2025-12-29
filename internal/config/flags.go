package config

import (
	"flag"
	"os"
)

// parses CLI flags for the docs subcommand
func ParseDocsFlags() Flags {
	args := os.Args[2:]

	fs := flag.NewFlagSet("docs", flag.ExitOnError)
	path := fs.String("path", "./docs/strudel", "path to documentation directory")
	clearFlag := fs.Bool("clear", false, "clear existing chunks before ingesting")
	fs.Parse(args) //nolint:errcheck,gosec // G104: ExitOnError flag set handles errors

	return Flags{Path: *path, Clear: *clearFlag}
}

// parses CLI flags for the concepts subcommand
func ParseConceptsFlags() Flags {
	args := os.Args[2:]

	fs := flag.NewFlagSet("concepts", flag.ExitOnError)
	path := fs.String("path", "./docs/concepts", "path to concepts directory")
	clearFlag := fs.Bool("clear", false, "clear existing concepts before ingesting")
	fs.Parse(args) //nolint:errcheck,gosec // G104: ExitOnError flag set handles errors

	return Flags{Path: *path, Clear: *clearFlag}
}

// parses CLI flags for the examples subcommand
func ParseExamplesFlags() Flags {
	args := os.Args[2:]

	fs := flag.NewFlagSet("examples", flag.ExitOnError)
	path := fs.String("path", "./resources/strudel_examples.json", "path to examples JSON file")
	clearFlag := fs.Bool("clear", false, "clear existing examples before ingesting")
	fs.Parse(args) //nolint:errcheck,gosec // G104: ExitOnError flag set handles errors

	return Flags{Path: *path, Clear: *clearFlag}
}

// returns default flags for docs ingestion
func DefaultDocsFlags() Flags {
	return Flags{Path: "./docs/strudel", Clear: false}
}

// returns default flags for concepts ingestion
func DefaultConceptsFlags() Flags {
	return Flags{Path: "./docs/concepts", Clear: false}
}

// returns default flags for examples ingestion
func DefaultExamplesFlags() Flags {
	return Flags{Path: "./resources/strudel_examples.json", Clear: false}
}
