# algorave - RAG-Powered Code Generation Agent

## Project Context
Building a RAG system that helps users create music via strudel with text commands by retrieving relevant documentation and feeding it to Claude API. The system transforms user queries for better retrieval and maintains conversation context for coherent multi-turn interactions.

## Architecture Overview

### Two-Binary Approach
1. **Ingestion CLI** (`cmd/ingest/main.go`) - Standalone tool to process docs and create embeddings
2. **API Server** (`cmd/server/main.go`) - Runtime service for code generation with RAG

### Current Focus: Building Ingestion CLI First

## Tech Stack
- **Language:** Go 1.21+
- **Module:** algorave
- **Vector DB:** Supabase pgvector
- **Embeddings:** OpenAI text-embedding-3-small (1536 dimensions)
- **LLM:** Claude API (Anthropic)
- **Docs:** 30 markdown pages in `docs/project-docs/`

## Project Structure
```
algorave/
├── cmd/
│   ├── ingester/
│   │   └── main.go          # Ingestion CLI entry point
│   └── server/              # (Build later)
│       └── main.go          # API server entry point
├── internal/
│   ├── chunker/
│   │   └── chunker.go       # Markdown chunking logic
│   ├── embedder/
│   │   └── openai.go        # OpenAI embedding client
│   ├── storage/
│   │   └── supabase.go      # Supabase pgvector operations
│   ├── retriever/           # (For server phase)
│   │   └── retriever.go     # Vector search + query transformation
│   └── agent/               # (For server phase)
│       └── agent.go         # Code generation orchestration
├── docs/
│   └── project-docs/        # Documentation to be indexed (30 pages)
├── agent.md                 # This file
├── .env                     # Environment variables (gitignored)
├── .env.example             # Template
└── go.mod
```

## Key Design Decisions

### Chunking Strategy
- Split by markdown headers (##, ###, ####)
- Target: ~500 tokens per chunk
- Overlap: 50 tokens between consecutive chunks
- **Critical:** Keep code blocks intact, never split them across chunks
- Preserve context by including section hierarchy in chunk metadata

### Query Processing Pipeline

#### Stage 1: Query Transformation (Before Retrieval)
Transform user queries into technical search terms for better document retrieval.

**Example:**
```
User: "play a loud pitched sound on the second and fourth beats"
  ↓ Query Transformation (Claude)
Search Query: "audio playback, frequency, volume, amplitude, sound generation, beat count"
```

**Implementation:**
- Use lightweight Claude call to extract technical keywords
- Combine original query + expanded keywords for hybrid search
- Keep transformation fast (<100ms)

**Benefits:**
- Maps conversational language to technical documentation terms
- Handles synonyms automatically (e.g., "loud" → "volume, amplitude")
- Improves retrieval recall significantly

#### Stage 2: Vector Retrieval (Stateless)
- Embed the transformed query
- Search Supabase pgvector for top K most similar chunks
- **Important:** Retrieval does NOT use conversation history
- Keep retrieval stateless for simplicity and caching

#### Stage 3: Code Generation (Stateful)
- Combine retrieved docs + conversation history + current query
- Send to Claude API for code generation
- Claude uses history to understand references like "the sound you played before"

**Architecture:**
```
User Query
    ↓
[Query Transformation] ← Claude (fast, cheap)
    ↓
[Vector Search] ← Stateless, no history
    ↓
Retrieved Docs
    ↓
[Code Generation] ← Claude + Docs + Conversation History
    ↓
Generated Code
```

### Conversation Context Management

**Key Principle:** Context added AFTER retrieval, not during.

**Short Conversations (≤6 messages):**
- Include full conversation history in Claude system prompt

**Long Conversations (>6 messages):**
- Implement conversation summarization

**Example Flow:**
```
Turn 1:
User: "play a loud pitched sound"
→ Retrieval: Audio synthesis docs
→ Generation: Uses docs only (no history)
→ Output: play(880, 1.0, volume=0.8)

Turn 2:
User: "adjust the volume to match the kick sound you played before"
→ Retrieval: Volume control docs (stateless search)
→ Generation: Uses docs + Turn 1 history
→ Output: play(880, 1.0, volume=0.6) // References previous code
```

### Supabase Schema
```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE doc_embeddings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    page_name TEXT NOT NULL,
    page_url TEXT NOT NULL,
    section_title TEXT,
    content TEXT NOT NULL,
    embedding vector(1536),
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX doc_embeddings_embedding_idx ON doc_embeddings 
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);

CREATE INDEX doc_embeddings_page_name_idx ON doc_embeddings(page_name);
```

### Error Handling
- **Fail fast:** If any chunk fails during ingestion, stop entire process
- Clear error messages with context (which file, which chunk, what failed)
- Log progress verbosely during ingestion
- Graceful degradation in API server (return partial results if possible)

### Logging Strategy

**Ingestion CLI:**
- Detailed progress logging:
  - Which file is being processed
  - Number of chunks created per file
  - Embedding generation progress
  - Storage confirmation for each chunk
  - Final summary (total chunks, total time)

**API Server:**
- Request/response logging
- Query transformation results (for debugging)
- Retrieved chunk count and relevance scores
- Generation time metrics

## Environment Variables
```
# Required
OPENAI_API_KEY=sk-...
SUPABASE_CONNECTION_STRING=postgresql://postgres:[PASSWORD]@[PROJECT].supabase.co:5432/postgres
ANTHROPIC_API_KEY=sk-ant-...

# Optional (with defaults)
CHUNK_TARGET_TOKENS=500
CHUNK_OVERLAP_TOKENS=50
RETRIEVAL_TOP_K=5
```

## Dependencies
```go
require (
    github.com/jackc/pgx/v5 v5.5.0          // PostgreSQL driver
    github.com/pgvector/pgvector-go v0.1.1   // pgvector support
    github.com/anthropic-ai/anthropic-sdk-go // Claude API (for server)
)
```

## Implementation Phases

### Phase 1: Ingestion CLI (Current)
1. ✅ Setup: go.mod, .env.example, agent.md, Supabase schema
2. 🔄 internal/chunker - Markdown chunking logic
3. ⏳ internal/embedder - OpenAI embedding client
4. ⏳ internal/storage - Supabase pgvector operations
5. ⏳ cmd/ingest - CLI orchestration
6. ⏳ Test with real documentation

### Phase 2: API Server (Later)
1. internal/retriever - Vector search + query transformation
2. internal/agent - Code generation with context management
3. cmd/server - REST API endpoints
4. Frontend integration with Replit editor

## Coding Guidelines

### General
- Prefer stdlib over external dependencies when possible
- Use `context.Context` for all I/O operations
- Structured error wrapping: `fmt.Errorf("operation failed: %w", err)`
- Comments explain "why" not "what"
- Keep functions focused and testable (<50 lines ideal)

### Go Style
- Use short variable names in small scopes (i, err, ctx, db)
- Group imports: stdlib, external, internal
- Error messages: lowercase, no punctuation at end
- Prefer table-driven tests

### Package Organization
- internal/ for private packages only used by this project
- cmd/ for executable entry points (keep main.go minimal)
- Each package should have a clear, single responsibility

### Error Handling Patterns
```go
// Good: Context in errors
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to process document %s: %w", filename, err)
}

// Good: Early returns
if err != nil {
    return err
}

// Avoid: Silent failures, generic error messages
```

## Testing Approach

### Phase 1 (Ingestion)
- Unit tests for chunker logic (most complex part)
  - Test markdown header splitting
  - Test code block preservation
  - Test token counting and overlap
  - Test edge cases (no headers, very long sections)
- Integration tests can come later
- Manual end-to-end testing with real docs for MVP

### Phase 2 (Server)
- Unit tests for query transformation
- Integration tests for retrieval pipeline
- End-to-end tests with mock Claude API

## Implementation Notes

### Chunker Edge Cases to Handle
- Documents without headers (treat as single chunk or split by paragraphs)
- Very large sections (>1000 tokens) → split by paragraphs with overlap
- Code blocks with triple backticks → never split
- Nested headers (h2, h3, h4) → preserve hierarchy in metadata
- Empty sections → skip

### OpenAI API Considerations
- Rate limits: 3,500 requests/min (tier 1)
- Batch embedding when possible for efficiency
- Handle rate limit errors with exponential backoff
- Cost: ~$0.00002 per 1K tokens (very cheap)

### Supabase/pgvector Best Practices
- Use connection pooling (pgxpool) for efficiency
- Batch inserts during ingestion for speed
- Use prepared statements for repeated queries
- Vector index: ivfflat is good for <1M vectors (our case)

### Query Transformation Guidelines
- Keep transformation prompts short and focused
- Use Claude Haiku or GPT-3.5-turbo (fast, cheap models)
- Target: 3-5 technical keywords per query
- Fallback: If transformation fails, use original query
- Cache common transformations (optional optimization)

## Performance Targets

### Ingestion (30 pages)
- Total time: <5 minutes
- Per page: <10 seconds
- Chunking: <1 second per page
- Embedding: ~2-3 seconds per chunk (depends on API latency)
- Storage: <100ms per chunk

### API Server (per request)
- Query transformation: <100ms
- Vector search: <50ms
- Code generation: 1-3 seconds (depends on complexity)
- Total response time: <5 seconds

## Debugging Tips
- Use `log.Printf` liberally during development
- Print chunk boundaries during chunking (helpful for tuning)
- Log embedding dimensions to verify (should be 1536)
- Log similarity scores during retrieval (helps tune topK)
- Print retrieved chunks to verify relevance

## Future Enhancements (Not MVP)
- Conversation summarization for very long chats
- Query result caching
- Semantic caching for embeddings
- User feedback loop (thumbs up/down on generated code)
- A/B testing different chunking strategies
- Multi-language documentation support
- Incremental doc updates (only re-index changed files)

## Notes for Claude Code
When implementing, pay special attention to:
- Markdown parsing: Use regex carefully, test with varied headers
- Token estimation: 4 chars ≈ 1 token is rough, consider using tiktoken library
- Code block detection: Handle both ```language and ``` styles
- Error context: Always include which file/chunk failed
- Progress indicators: Users want to see what's happening during long operations
- Each suggestion/change/decision should be explained in layman terms. EG if you suggest a line of code or a function, indicate why.