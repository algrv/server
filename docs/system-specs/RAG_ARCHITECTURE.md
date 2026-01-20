# RAG Architecture

This document provides an overview of the Retrieval-Augmented Generation (RAG) system powering Algojams's intelligent code generation.

## Overview

Algojams uses a hybrid retrieval system that combines vector similarity search with keyword-based BM25 search to find the most relevant Strudel documentation and examples for user queries.

## Architecture Components

### 1. Document Processing Pipeline

**Location**: `cmd/ingester/`

The ingestion pipeline processes Strudel documentation:
- Fetches documentation from official Strudel sources
- Chunks documents using semantic splitting
- Generates embeddings using OpenAI's `text-embedding-3-small` model
- Stores in PostgreSQL with pgvector extension

### 2. Retrieval System

**Location**: `internal/retriever/`

Implements hybrid search combining:
- **Vector Search (70% weight)**: Semantic similarity using cosine distance
- **BM25 Search (30% weight)**: Keyword-based full-text search

### 3. Agent System

**Location**: `internal/agent/`

The agent orchestrates:
- Query actionability detection
- Context-aware retrieval
- LLM-based code generation
- Response formatting

## Detailed Implementation

For in-depth technical details on the hybrid retrieval system, see:

**[Hybrid Retrieval Guide](./HYBRID_RETRIEVAL_GUIDE.md)**

This guide covers:
- Vector search implementation
- BM25 integration
- Score fusion algorithms
- Query transformation
- Performance optimizations

## Key Features

### Query Transformation
Transforms natural language queries into technical keywords for better vector search:
```
"make it sound like drums" → "drums percussion rhythm pattern sound synthesis"
```

### Hybrid Scoring
Combines vector and BM25 results with weighted scoring:
```go
finalScore = (vectorSimilarity * 0.7) + (bm25Rank * 0.3)
```

### Special Chunk Handling
Automatically includes page summaries and examples for context.

## Database Schema

### doc_embeddings
```sql
- id: UUID
- page_name: TEXT
- page_url: TEXT
- section_title: TEXT
- content: TEXT
- embedding: vector(1536)
- content_tsvector: tsvector (for BM25)
- metadata: JSONB
```

### example_strudels
```sql
- id: UUID
- title: TEXT
- description: TEXT
- code: TEXT
- tags: TEXT[]
- embedding: vector(1536)
- searchable_tsvector: tsvector
- url: TEXT
```

## Search Flow

1. **User Query** → Agent receives query + editor state
2. **Actionability Check** → Determines if query is code-related
3. **Query Transformation** → Enhances query with technical keywords
4. **Parallel Retrieval**:
   - Vector search on transformed query
   - BM25 search on original query
5. **Score Fusion** → Combines results with 70/30 weighting
6. **Organization** → Groups by page, adds summaries/examples
7. **LLM Generation** → Uses retrieved context to generate code
8. **Response** → Returns Strudel code or explanation

## Configuration

Key parameters in `internal/retriever/`:

```go
defaultTopK = 5           // Number of results to retrieve
vectorWeight = 0.7        // Vector search weight
bm25Weight = 0.3          // BM25 search weight
```

## Related Documentation

- [Hybrid Retrieval Guide](./HYBRID_RETRIEVAL_GUIDE.md) - Detailed implementation
- [Strudel Code Analysis](./STRUDEL_CODE_ANALYSIS.md) - Code pattern analysis
- [Product Architecture](./PRODUCT_ARCHITECTURE.md) - User-facing features
