# Algorave

Create music using human language.

## Quick Start

### Prerequisites

- Go 1.21+
- Supabase account
- OpenAI API key
- Anthropic API key (for server phase)

### Setup

1. **Clone and enter the project:**

   ```bash
   cd algorave
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

   This will create:
   - `doc_embeddings` table with pgvector support
   - Vector similarity search index (ivfflat)
   - Indexes on `page_name` and `created_at`

   d. Get your connection string:
   - Go to Supabase Dashboard → Project Settings → Database
   - Copy the Connection String (URI format)

4. **Configure environment:**

   ```bash
   cp .env.example .env
   # Edit .env with your actual API keys
   ```

   Required variables:
   - `OPENAI_API_KEY` - For generating embeddings
   - `SUPABASE_CONNECTION_STRING` - Database connection
   - `ANTHROPIC_API_KEY` - For code generation (server phase)

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

### Database Schema

The migration in `supabase/migrations/20251224013556_init_schema.sql` creates:

**Table: `doc_embeddings`**
- `id` - UUID primary key
- `page_name` - Source file path (e.g., "learn/notes.mdx")
- `page_url` - Generated URL for the page
- `section_title` - Section header within the document
- `content` - The actual chunk text
- `embedding` - Vector embedding (1536 dimensions, OpenAI text-embedding-3-small)
- `metadata` - JSONB for frontmatter and custom metadata
- `created_at` - Timestamp

**Indexes:**
- IVFFlat index on `embedding` for fast similarity search
- B-tree index on `page_name` for lookups
- B-tree index on `created_at` for maintenance

### Automated Ingestion

The project includes a GitHub Actions workflow (`.github/workflows/ingest-docs.yml`) that:
- Runs every 6 hours (configurable cron)
- Clones the Strudel documentation from Codeberg
- Runs the ingestion pipeline
- Updates the vector database with latest docs

You can also trigger it manually via GitHub UI → Actions → "Sync & Ingest Strudel Docs" → Run workflow

### Development

See `agent.md` for full architectural details.

For detailed coding standards, see `.clinerules`.

## Architecture

```
User Query → Query Transformation → Vector Search → Retrieved Docs
                                                            ↓
                                    Code Generation ← Docs + History
```

See `agent.md` for complete architecture documentation.
