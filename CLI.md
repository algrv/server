# Algorave CLI Tools

This document describes the command-line interface tools for Algorave.

## Quick Start

### Build All Binaries
```bash
make build
```

This creates:
- `bin/algorave` - Local interactive CLI
- `bin/server` - HTTP API server
- `bin/ingester` - Documentation ingester

## Local CLI (`bin/algorave`)

Interactive terminal interface for local development.

### Usage
```bash
# Development mode (all commands available)
ALGORAVE_ENV=development ./bin/algorave

# Production mode (ingester hidden)
ALGORAVE_ENV=production ./bin/algorave
```

### Commands
- `start` - Start the Algorave HTTP server
- `ingest` - Run documentation ingester (dev mode only)
- `editor` - Interactive code editor with AI assistance
- `quit` - Exit the CLI

### Editor Mode
Press `Ctrl+S` to send code to AI for assistance
Press `Ctrl+L` to clear the editor
Press `Ctrl+C` to exit editor mode

## Architecture

### Local CLI Flow
```
User → algorave binary → TUI (Bubbletea) → Agent API
                       → Server (background)
                       → Ingester (dev mode)
```

### Production vs Development Mode

**Development Mode:**
- All commands available
- Can run ingester
- Full access to all features

**Production Mode:**
- Ingester hidden
- Safe for public/guest access
- Users can still use editor and start server

## Makefile Commands

```bash
make build      # Build all binaries
make cli        # Build local CLI only
make clean      # Remove all binaries
make help       # Show all available commands
```

## File Locations

```
algorave/
├── bin/                    # All compiled binaries (gitignored)
│   ├── algorave            # Local CLI
│   ├── server              # HTTP server
│   └── ingester            # Documentation ingester
│
├── cmd/
│   ├── tui/                # Local CLI source
│   ├── server/             # HTTP server source
│   └── ingester/           # Ingester source
│
└── internal/
    └── tui/                # Terminal UI components
```

## Examples

### Local Development Workflow
```bash
# Build the CLI
make cli

# Run in dev mode
ALGORAVE_ENV=development ./bin/algorave

# At the prompt:
> ingest          # Ingest documentation
> start           # Start HTTP server
> editor          # Open code editor
> quit            # Exit
```

### Using the Editor
```bash
./bin/algorave

> editor

# In editor:
# Type your musical idea:
"make a drum beat with reverb"

# Press Ctrl+S to send to AI
# AI generates Strudel code
# Continue editing or ask questions

# Press Ctrl+C to return to menu
```

## Troubleshooting

### Agent Connection Errors
```bash
# Make sure HTTP server is running
./bin/server

# Or configure custom endpoint
ALGORAVE_AGENT_ENDPOINT=http://your-server:8080/api/v1/generate ./bin/algorave
```

## Next Steps

See [CLI_ARCHITECTURE.md](docs/system-specs/CLI_ARCHITECTURE.md) for detailed technical architecture.

See [CLI_IMPLEMENTATION_PLAN.md](docs/CLI_IMPLEMENTATION_PLAN.md) for full implementation details.
