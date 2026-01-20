# Algopatterns

Create music using human language.

## Quick Start

### Prerequisites

- Go 1.24+
- Supabase account (PostgreSQL with pgvector)
- Redis (for session buffering)
- OpenAI API key (for embeddings)
- Anthropic API key (for code generation)
- Google OAuth credentials (required for authentication)
- GitHub OAuth credentials (optional)

### Setup

1. **Clone and enter the project:**

   ```bash
   cd algopatterns
   ```

2. **Install dependencies:**

   ```bash
   go mod download
   ```

3. **Set up Supabase:**

   a. Install Supabase CLI (if not already installed):

   ```bash
   # macOS
   brew install supabase/tap/supabase

   # Other platforms: https://supabase.com/docs/guides/cli
   ```

   b. Link your Supabase project:

   ```bash
   supabase link --project-ref your-project-ref
   ```

   c. Run migrations to set up the database schema:

   ```bash
   supabase db push
   ```

   This will create the database schema including:
   - `doc_embeddings` - Vector embeddings for RAG retrieval
   - `users` - User accounts and preferences
   - `user_strudels` - Saved Strudel patterns
   - `collaborative_sessions` - Real-time collaboration rooms
   - `session_messages` - Chat and code messages
   - `usage_tracking` - API usage metrics
   - And supporting indexes for vector similarity search

   d. Get your connection string:
   - Go to Supabase Dashboard → Project Settings → Database
   - Copy the Connection String (URI format)

4. **Configure environment:**

   ```bash
   cp .env.example .env
   # Edit .env with your actual values
   ```

   Required variables:
   - `SUPABASE_CONNECTION_STRING` - PostgreSQL connection string
   - `REDIS_URL` - Redis URL for session buffering
   - `OPENAI_API_KEY` - For generating embeddings
   - `ANTHROPIC_API_KEY` - For code generation
   - `JWT_SECRET` - Secret for signing JWT tokens
   - `SESSION_SECRET` - Secret for OAuth cookie signing
   - `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` - Google OAuth

   See `.env.example` for the full list of configuration options.

### Running Ingestion

```bash
go run cmd/ingester/main.go --docs ./docs/strudel --clear
```

Options:

- `--docs`: Path to documentation directory (default: `./docs/strudel`)
- `--clear`: Clear existing chunks before ingesting

The ingestion process will:
1. Discover all `.md` and `.mdx` files in the docs directory
2. Chunk documents intelligently (preserving section context)
3. Generate embeddings in batch via OpenAI API
4. Store chunks with embeddings in Supabase

**Note:** The `--clear` flag deletes all existing chunks from the database before ingesting. Use it when you want a fresh start.

### Running the Server

```bash
go run cmd/server/main.go
```

The server provides:
- REST API for Strudel code generation
- WebSocket support for real-time collaboration
- OAuth authentication (Google, GitHub)
- Anonymous session support

Default port is `8080`. Override with the `PORT` environment variable.

### Running the TUI Client

```bash
go run cmd/tui/main.go
```

A terminal-based interface for interacting with Algopatterns.

### Automated Ingestion

The project includes a GitHub Actions workflow (`.github/workflows/ingest.yml`) that:
- Runs every 6 hours (configurable cron)
- Clones the Strudel documentation from Codeberg
- Runs the ingestion pipeline
- Updates the vector database with latest docs

You can also trigger it manually via GitHub UI → Actions → "Sync & Ingest Strudel Docs" → Run workflow

### Development

See `AGENTS.md` for full architectural details.

For detailed coding standards, see `.clinerules`.

## Architecture

```
User Query → Query Transformation → Vector Search → Retrieved Docs
                                                            ↓
                                    Code Generation ← Docs + History
```

See `AGENTS.md` for complete architecture documentation.

## License

This project is licensed under the [GNU Affero General Public License v3.0](LICENSE).
