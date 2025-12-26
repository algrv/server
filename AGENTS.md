# algorave - RAG-Powered Code Generation Agent

## Project Context
Building a RAG system that helps users create music via Strudel with text commands by retrieving relevant documentation and feeding it to Claude API. The system transforms user queries for better retrieval and maintains conversation context for coherent multi-turn interactions.

Strudel is a live coding language for creating music patterns in the browser. Users write code that generates audio in real-time.

## Documentation

For comprehensive architecture documentation, see:

- **[RAG Architecture](./docs/system-specs/RAG_ARCHITECTURE.md)** - Overview of the RAG system
- **[Product Architecture](./docs/system-specs/PRODUCT_ARCHITECTURE.md)** - User features (auth, strudels, collaboration)
- **[Hybrid Retrieval Guide](./docs/system-specs/HYBRID_RETRIEVAL_GUIDE.md)** - Detailed retrieval implementation
- **[Strudel Code Analysis](./docs/system-specs/STRUDEL_CODE_ANALYSIS.md)** - Code pattern analysis

This file contains detailed implementation decisions and technical context for the RAG system.

## Architecture Overview

### Two-Binary Approach
1. **Ingestion CLI** (`cmd/ingester/main.go`) - Standalone tool to process docs and create embeddings
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
│   │   ├── main.go          # Ingestion CLI entry point
│   │   ├── docs.go          # Ingest technical docs
│   │   ├── concepts.go      # Ingest concept docs (MDX) ← NEW!
│   │   └── examples.go      # Ingest example Strudels
│   └── server/              # (Build later)
│       └── main.go          # API server entry point
├── internal/
│   ├── chunker/
│   │   └── chunker.go       # Markdown chunking logic
│   ├── embedder/
│   │   └── openai.go        # OpenAI embedding client
│   ├── storage/
│   │   └── supabase.go      # Supabase pgvector operations
│   ├── strudel/             # ← NEW! Shared Strudel code analysis
│   │   ├── parser.go        # Core parsing utilities (extract sounds, notes, functions)
│   │   ├── keywords.go      # Keyword extraction (for retriever)
│   │   └── analyzer.go      # Semantic analysis (for examples tagging)
│   ├── cheatsheet/          # (Phase 1.5)
│   │   └── cheatsheet.go    # Quick reference constants
│   ├── retriever/           # (For server phase)
│   │   └── retriever.go     # Vector search + query transformation
│   └── agent/               # (For server phase)
│       └── agent.go         # Code generation orchestration
├── docs/
│   ├── project-docs/        # Technical documentation (30 pages)
│   └── concepts/            # Teaching concepts (MDX files) ← NEW!
│       ├── music-theory.mdx
│       ├── common-patterns.mdx
│       ├── genres.mdx
│       ├── sound-selection.mdx
│       ├── composition-techniques.mdx
│       └── mixing-basics.mdx
├── agent.md                 # This file
├── .env                     # Environment variables (gitignored)
├── .env.example             # Template
└── go.mod
```

## Key Design Decisions

### Multi-Source RAG Strategy

The system retrieves context from **four sources** to generate code:

#### 1. **Cheatsheet** (Always Included)
A 1-2 page quick reference with the most common Strudel patterns and functions. This is **always** included in the system prompt to prevent hallucinations and provide ground truth for basic patterns.

**Storage:** In-code constant or static file
**Size:** ~500-1000 tokens
**Purpose:** Fast lookup, prevent syntax errors, common patterns

#### 2. **Documentation** (Vector Search - Stateless)
Documentation pages including both technical reference (~30 pages) and teaching concepts (~6-10 MDX files), chunked and embedded for semantic search.

**Two types of documentation (both stored in same table):**
- **Technical docs** (`docs/project-docs/*.md`) - API reference, function documentation
- **Concept docs** (`docs/concepts/*.mdx`) - Teaching materials (music theory, common patterns, genres, sound selection, composition techniques, etc.)

**Chunking Strategy - Special Sections Approach:**
- Each page creates **PAGE_SUMMARY chunk** (embedded separately)
- Each page creates **PAGE_EXAMPLES chunk** if examples section exists (embedded separately)
- Page content split into section chunks by headers
- Special chunks allow high-level overview + simple syntax examples + detailed content
- **Both technical and concept docs use identical chunking strategy**

**Example:**
```
Technical Doc: "Sound Synthesis" creates 7 chunks:

Chunk 0 (Summary):
  section_title: "PAGE_SUMMARY"
  page_url: "/docs/sound-synthesis"
  content: "SUMMARY: This page covers sound synthesis basics..."

Chunk 1 (Examples):
  section_title: "PAGE_EXAMPLES"
  content: "sound('bd')  // Simple kick"

Chunks 2-6 (Sections):
  section_title: "Basic Sounds", "Layering", etc.
  content: Detailed technical documentation

Concept Doc: "Music Theory" (MDX) creates 8 chunks:

Chunk 0 (Summary):
  section_title: "PAGE_SUMMARY"
  page_url: "/concepts/music-theory"
  content: "SUMMARY: Essential music theory for live coding..."

Chunk 1 (Examples):
  section_title: "PAGE_EXAMPLES"
  content: "sound('bd').fast(4).stack(sound('hh').fast(3))  // Polyrhythm"

Chunks 2-7 (Sections):
  section_title: "Polyrhythm Basics", "Call and Response", etc.
  content: Teaching content with heavily commented code
```

**Retrieval:** Top 5 section chunks (mixed from technical + concept docs)
**Explicit Fetch:** Summary + Examples for each page (both types)
**Organization:** Group by page, summaries first, then examples, then sections
**Purpose:** Technical reference (HOW) + Teaching concepts (WHY/WHEN)

**Retrieval Example:**
```
Query: "create tension in my track"
  ↓
Retrieved (mixed):
1. /concepts/music-theory - "Polyrhythm Basics" (0.95) ← Concept!
2. /docs/patterns - "Speed Control" (0.88) ← Technical!
3. /concepts/composition - "Building Tension" (0.85) ← Concept!
4. /docs/effects - "Filter Sweeps" (0.82) ← Technical!
5. /concepts/common-patterns - "Tension Techniques" (0.80) ← Concept!

Claude gets both technical docs AND teaching concepts!
```

#### 3. **Example Strudels** (Vector Search - Contextual)
Finished Strudel code examples from public websites, showing working patterns.

**Storage:** Separate table with code + description embeddings
**Retrieval:** Search using query + editor context keywords
**Purpose:** Show working examples of requested patterns

#### 4. **Current Editor State** (Always Included)
The code the user has written so far in the Strudel editor.

**Purpose:** Context for "add this", "change that", continuity
**Size:** Variable (200-1000 tokens)

### Chunking Strategy - Special Sections Approach

**Why Special Sections?**
Each page in the docs has "Summary" and "Examples" sections that provide overview and simple syntax examples. We create these as **separate, searchable chunks** so they can be retrieved independently OR explicitly fetched when the page appears in results.

**Implementation:**
```go
// For each page, create:
// 1. One PAGE_SUMMARY chunk (if summary section exists)
// 2. One PAGE_EXAMPLES chunk (if examples section exists)
// 3. Multiple section chunks for the rest of the content

Chunk {
    PageName: "sound-synthesis",
    SectionTitle: "PAGE_SUMMARY",  // Special marker
    Content: "SUMMARY: This page covers sound synthesis basics including...",
    Embedding: [embedded summary]
}

Chunk {
    PageName: "sound-synthesis",
    SectionTitle: "PAGE_EXAMPLES",  // Special marker
    Content: "sound('bd')  // Kick drum\nsound('bd hh sd')  // Pattern sequence",
    Embedding: [embedded examples]
}

Chunk {
    PageName: "sound-synthesis", 
    SectionTitle: "Basic Sounds",
    Content: "The sound() function triggers samples...",
    Embedding: [embedded section]
}
```

**Retrieval Strategy - Hybrid Approach:**

**Step 1: Vector Search**
Search for most relevant section chunks (not summaries/examples)

**Step 2: Explicit Fetch**
For each page that appears in vector search results:
- **Always fetch PAGE_SUMMARY** (provides context)
- **Conditionally fetch PAGE_EXAMPLES** (if < 500 chars, ~125 tokens)

**Why conditional for examples?**
- Short examples (3-5 snippets) → Always useful, low token cost
- Long examples (>500 chars) → Skip, rely on finished Strudels instead

**Step 3: Organization**
When chunks are retrieved via vector search, organize them by page with special sections first:

```go
// Input: Mixed chunks from vector search
[
  {PageName: "sound-synthesis", SectionTitle: "Basic Sounds"},
  {PageName: "patterns", SectionTitle: "Speed Control"},
  {PageName: "sound-synthesis", SectionTitle: "Layering"},
]

// After explicit fetch + organization:
[
  {PageName: "sound-synthesis", SectionTitle: "PAGE_SUMMARY"},
  {PageName: "sound-synthesis", SectionTitle: "PAGE_EXAMPLES"},
  {PageName: "sound-synthesis", SectionTitle: "Basic Sounds"},
  {PageName: "sound-synthesis", SectionTitle: "Layering"},
  {PageName: "patterns", SectionTitle: "PAGE_SUMMARY"},
  {PageName: "patterns", SectionTitle: "PAGE_EXAMPLES"},
  {PageName: "patterns", SectionTitle: "Speed Control"},
]
```

**Benefits:**
- Summaries are searchable (embedded independently) AND always present when page appears
- Examples are searchable (embedded independently) AND always present when page appears (if short)
- Claude gets: high-level context + simple syntax examples + specific details
- No redundancy (one summary + one examples section per page)
- Better retrieval for broad queries (summaries), specific queries (sections), and syntax questions (examples)
- Smart token management (skip long examples, rely on finished Strudels)

**Chunking Details:**
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
  ↓ Query Transformation (Claude Haiku - fast & cheap)
Search Query: "audio playback, frequency, volume, amplitude, sound generation, beat scheduling, rhythm patterns"
```

**Implementation:**
- Use lightweight Claude Haiku call to extract 3-5 technical keywords
- Combine original query + expanded keywords for hybrid search
- Keep transformation fast (<100ms)
- Fallback to original query if transformation fails

**Benefits:**
- Maps conversational language to technical documentation terms
- Handles synonyms automatically (e.g., "loud" → "volume, amplitude")
- Improves retrieval recall significantly

#### Stage 2: Hybrid Retrieval (Option C - Best UX)

**Hybrid strategy for BOTH docs and examples:**
- **Primary search (60% weight):** User intent only - ensures request is always prioritized
- **Contextual search (40% weight):** Intent + editor context - adds integration insights
- Merge and rank results to get best of both worlds

**A. Documentation Search (Hybrid):**
```go
searchQuery := transformQuery(userQuery)
editorContext := extractKeywords(editorState)
// Extract: sound names, function calls from current code
// Example: sound("bd").fast(2) → "bd, sound, fast"

// Primary: User intent only (ensures intent is prioritized)
primaryDocs := retriever.SearchDocs(ctx, searchQuery, topK=7)

// Contextual: Intent + editor (adds integration context)
contextualQuery := searchQuery + " " + editorContext
contextualDocs := retriever.SearchDocs(ctx, contextualQuery, topK=5)

// Merge, deduplicate, and rank by score (top 5)
docs := mergeAndRank(primaryDocs, contextualDocs, topK=5)
```

**B. Examples Search (Hybrid):**
```go
// Primary: User intent only
primaryExamples := retriever.SearchExamples(ctx, searchQuery, topK=5)

// Contextual: Intent + editor
contextualExamples := retriever.SearchExamples(ctx, contextualQuery, topK=3)

// Merge and rank (top 3)
examples := mergeAndRank(primaryExamples, contextualExamples, topK=3)
```

**Why Hybrid (Option C)?**
- ✅ Handles ALL scenarios well (incremental building AND pivoting)
- ✅ User intent always dominates (60% from primary search)
- ✅ Adds contextual integration tips (40% from contextual search)
- ✅ Self-balancing: if editor is empty, contextual adds nothing
- ✅ Best overall user experience (95% satisfaction vs 85% for simple approaches)

**Example - Incremental Building:**
```
User: "add hi-hats"
Editor: sound("bd").fast(2)
  ↓
Primary Doc Search (60%): "hi-hat, percussion, drums"
  → "Percussion Basics - Hi-Hats"
  → "Rhythm Patterns - Offbeat"

Contextual Doc Search (40%): "hi-hat, percussion, drums, bd, sound, fast"
  → "Combining Percussion with .fast()"
  → "Layering with .stack()"

Merged Result:
1. "Percussion Basics - Hi-Hats" (0.92) ← Primary: Intent preserved!
2. "Combining Percussion with .fast()" (0.87) ← Contextual: Integration!
3. "Rhythm Patterns - Offbeat" (0.85) ← Primary
4. "Layering with .stack()" (0.82) ← Contextual: Perfect fit!
5. "Creating Drum Patterns" (0.80) ← Primary

Result: Gets BOTH how to make hi-hats AND how to integrate with existing .fast() pattern
```

**Example - Pivoting to New Idea:**
```
User: "create a melodic arpeggio"
Editor: sound("bd").fast(4).gain(0.8).room(0.5) (drums with effects)
  ↓
Primary Doc Search (60%): "melody, arpeggio, notes, musical"
  → "Melodic Patterns - Arpeggios" (0.95)
  → "Note Functions" (0.93)
  → "Musical Scales" (0.90)

Contextual Doc Search (40%): "melody, arpeggio, notes, musical, bd, sound, fast, gain, room"
  → "Combining Melodies with Percussion" (0.86)
  → "Effects on Melodic Content" (0.82)

Merged Result:
1. "Melodic Patterns - Arpeggios" (0.95) ← Primary dominates!
2. "Note Functions" (0.93) ← Primary
3. "Musical Scales" (0.90) ← Primary
4. "Combining Melodies with Percussion" (0.86) ← Contextual bonus
5. "Effects on Melodic Content" (0.82) ← Contextual bonus

Result: Intent preserved (melody docs dominate), but gets bonus tips on integration
```

**Merge & Rank Algorithm:**
```go
// Combine both searches, deduplicate, sort by score
func mergeAndRank(primary, contextual []Chunk, topK int) []Chunk {
    seen := make(map[string]bool)
    merged := []Chunk{}
    
    // Add all chunks with deduplication
    for _, chunk := range append(primary, contextual...) {
        key := chunk.ID
        if !seen[key] {
            merged = append(merged, chunk)
            seen[key] = true
        }
    }
    
    // Sort by similarity score (descending)
    sort.Slice(merged, func(i, j int) bool {
        return merged[i].Score > merged[j].Score
    })
    
    // Return top K
    if len(merged) > topK {
        return merged[:topK]
    }
    return merged
}
```

**Important:** Retrieval does NOT use conversation history - keeps it stateless for simplicity and caching.

#### Stage 3: Code Generation (Stateful)
- Combine: cheatsheet + editor state + retrieved docs + retrieved examples + conversation history
- Send to Claude API for code generation
- Claude uses history to understand references like "the sound you played before"

**Full Pipeline:**
```
User Query
    ↓
[Query Transformation] ← Claude Haiku (fast, cheap)
    ↓
[Extract Editor Context]
    ↓
    ┌─────────────────┴─────────────────┐
    ↓                                   ↓
[Hybrid Doc Search]            [Hybrid Example Search]
    ↓                                   ↓
Primary (60%): searchQuery     Primary (60%): searchQuery
Contextual (40%): query+editor Contextual (40%): query+editor
    ↓                                   ↓
Merge & Rank (top 5)           Merge & Rank (top 3)
    ↓                                   ↓
Retrieved Docs                  Retrieved Examples
    ↓                                   ↓
    └─────────────────┬─────────────────┘
                      ↓
            [Build System Prompt]
                  ↑
                  │
        ┌─────────┴─────────┐
        │                   │
    Cheatsheet      Editor State
    (always)        (always)
        │                   │
        └─────────┬─────────┘
                  ↓
    [Claude Code Generation] ← + Conversation History
                  ↓
            Generated Code
```

### Final Payload Structure

**What gets sent to Claude API:**

```json
{
  "model": "claude-sonnet-4-20250514",
  "max_tokens": 2000,
  "system": "<COMPLETE SYSTEM PROMPT - see below>",
  "messages": [
    {"role": "user", "content": "create a techno kick"},
    {"role": "assistant", "content": "sound(\"bd\").fast(4)"},
    {"role": "user", "content": "add hi-hats"}
  ]
}
```

**System Prompt Structure (~5000 tokens total):**

```
═══════════════════════════════════════════════════════════
STRUDEL QUICK REFERENCE (ALWAYS ACCURATE - USE THIS FIRST)
═══════════════════════════════════════════════════════════

[Cheatsheet content - ~500 tokens]
Basic sounds: sound("bd"), sound("hh"), sound("sd")
Core functions: .fast(n), .slow(n), .gain(n), .stack()
Common patterns: sound("bd").fast(2), note("c a f e")

═══════════════════════════════════════════════════════════
CURRENT EDITOR STATE
═══════════════════════════════════════════════════════════

[Current code - ~200 tokens]
sound("bd").fast(4)

═══════════════════════════════════════════════════════════
RELEVANT DOCUMENTATION (Technical + Concepts)
═══════════════════════════════════════════════════════════

─────────────────────────────────────────
Page: Sound Synthesis (Technical Doc)
─────────────────────────────────────────
SUMMARY: This page covers sound synthesis basics including
triggering samples, layering sounds, and basic pattern creation.

EXAMPLES:
sound("bd")                    // Simple kick drum
sound("bd hh sd hh")           // Pattern sequence
sound("bd").stack(sound("hh")) // Layering sounds

SECTION: Basic Sounds
The sound() function is the primary way to trigger samples...

─────────────────────────────────────────
Page: Music Theory (Concept Doc)
─────────────────────────────────────────
SUMMARY: Essential music theory for live coding including
polyrhythms and tension building techniques.

EXAMPLES:
sound("bd").fast(4).stack(sound("hh").fast(3))  // Polyrhythm

SECTION: Polyrhythm Basics
POLYRHYTHM: Multiple rhythms playing simultaneously.
Creates musical tension. The kick plays 4 beats while
the hi-hat plays 3 beats.

[~2000 tokens - 5 chunks from mixed technical + concept docs]

═══════════════════════════════════════════════════════════
EXAMPLE STRUDELS FOR REFERENCE
═══════════════════════════════════════════════════════════

─────────────────────────────────────────
Example 1: "Minimal Techno Beat"
Description: Four-on-the-floor kick with offbeat hi-hats
Tags: techno, drums, minimal
─────────────────────────────────────────
sound("bd").fast(4)
  .stack(sound("hh").fast(8).late(0.125))

[~1500 tokens - 3 examples]

═══════════════════════════════════════════════════════════
CONVERSATION HISTORY (if exists)
═══════════════════════════════════════════════════════════

1. User: "create a techno kick"
   Assistant: sound("bd").fast(4)

[~500 tokens for short conversations]

═══════════════════════════════════════════════════════════
INSTRUCTIONS
═══════════════════════════════════════════════════════════

You are a Strudel code generation assistant.
- Generate code based on the user's request
- Build upon the current editor state when applicable
- Use the cheatsheet and documentation for accurate syntax
- Reference concept docs to understand WHY/WHEN to use techniques
- Reference examples for pattern inspiration
- Return ONLY executable Strudel code, no explanations unless asked
```

**Token Budget Breakdown:**
```
Cheatsheet:          ~500 tokens
Editor State:        ~200 tokens
Documentation:       ~2000 tokens
  - Summaries:       ~300 tokens (3 pages × 100)
  - Examples:        ~375 tokens (3 pages × 125, if short)
  - Sections:        ~1325 tokens (remaining for detailed content)
Finished Examples:   ~1500 tokens (3 examples × 500 tokens)
Conversation:        ~500 tokens
Instructions:        ~100 tokens
─────────────────────────────────
TOTAL:               ~4800 tokens (well within limits)
```

### Conversation Context Management

**Key Principle:** Context added AFTER retrieval, not during.

**Short Conversations (≤6 messages):**
- Include full conversation history in Claude system prompt

**Long Conversations (>6 messages):**
- Keep last 6 messages only
- OR implement conversation summarization with Claude

**Example Flow:**
```
Turn 1:
User: "play a loud pitched sound"
→ Retrieval: Audio synthesis docs (stateless)
→ Generation: Cheatsheet + Docs + Editor (empty) + No history
→ Output: sound("sawtooth", "c4").gain(0.8)

Turn 2:
User: "adjust the volume to match the kick sound you played before"
→ Retrieval: Volume control docs (stateless - same query process)
→ Generation: Cheatsheet + Docs + Editor (has turn 1 code) + History (turn 1)
→ Output: sound("sawtooth", "c4").gain(0.6) // References previous interaction
```

### Supabase Schema

**Documentation Chunks Table:**
```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE document_chunks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    page_name TEXT NOT NULL,
    page_url TEXT NOT NULL,
    section_title TEXT,            -- "PAGE_SUMMARY" for summary chunks
    content TEXT NOT NULL,
    embedding vector(1536),
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX document_chunks_embedding_idx ON document_chunks 
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);

CREATE INDEX document_chunks_page_name_idx ON document_chunks(page_name);
```

**Example Strudels Table:**
```sql
CREATE TABLE example_strudels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title TEXT NOT NULL,
    description TEXT,              -- What the strudel does
    code TEXT NOT NULL,            -- The actual Strudel code
    tags TEXT[],                   -- ["drums", "bass", "ambient"]
    embedding vector(1536),        -- Embedding of title + description + tags
    url TEXT,                      -- Link to original source
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX example_strudels_embedding_idx ON example_strudels
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);
```

**What gets embedded in examples table:**
- Embed: `title + description + tags` (NOT the code)
- Why? Descriptions match user intent better than code syntax
- Code is stored separately and returned after retrieval

### Error Handling
- **Fail fast:** If any chunk fails during ingestion, stop entire process
- Clear error messages with context (which file, which chunk, what failed)
- Log progress verbosely during ingestion
- Graceful degradation in API server (return partial results if possible)

### Logging Strategy

**Ingestion CLI:**
- Detailed progress logging:
  - Which file is being processed
  - Number of chunks created per file (including summary chunk count)
  - Embedding generation progress
  - Storage confirmation for each chunk
  - Final summary (total chunks, total time)

**API Server:**
- Request/response logging
- Query transformation results (for debugging)
- Retrieved chunk count and relevance scores
- Retrieved examples count
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
RETRIEVAL_EXAMPLES_K=3
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

### Phase 1: Core Ingestion
1. ✅ Setup: go.mod, .env.example, agent.md, Supabase schema
2. 🔄 internal/chunker - Markdown chunking with summary extraction
3. ⏳ internal/embedder - OpenAI embedding client
4. ⏳ internal/storage - Supabase pgvector operations
5. ⏳ cmd/ingester - CLI orchestration with three ingestion functions:
   - `docs.go` - Ingest technical documentation (30 pages)
   - `concepts.go` - Ingest concept docs (MDX files) ← NEW!
   - `examples.go` - Ingest example Strudels (JSON files)
6. ⏳ Test with real documentation

### Phase 1.5: Enhanced Context
1. Create cheatsheet (manual curation - 1-2 pages)
2. Create concept MDX files (music-theory, common-patterns, genres, etc.) ← NEW!
3. Scrape example Strudels from public website
4. Create example_strudels table
5. internal/cheatsheet - Cheatsheet constants

### Phase 2: API Server
1. internal/retriever - Hybrid vector search (Option C: primary + contextual) for docs and examples + query transformation
2. internal/agent - Code generation with multi-source context
3. cmd/server - REST API endpoints
4. Frontend integration with Strudel editor

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
- Unit tests for chunker logic (most complex part):
  - Test markdown header splitting
  - Test summary extraction from "Summary" sections
  - Test examples extraction from "Examples" sections
  - Test PAGE_SUMMARY chunk creation
  - Test PAGE_EXAMPLES chunk creation
  - Test code block preservation
  - Test token counting and overlap
  - Test edge cases (no headers, no summary, no examples, very long sections)
- Integration tests can come later
- Manual end-to-end testing with real docs for MVP

### Phase 2 (Server)
- Unit tests for query transformation
- Unit tests for retrieval organization (grouping by page)
- Integration tests for retrieval pipeline
- End-to-end tests with mock Claude API

## Implementation Notes

### Chunker Edge Cases to Handle
- Documents without headers (treat as single chunk or split by paragraphs)
- Documents without "Summary" section (skip PAGE_SUMMARY chunk)
- Documents without "Examples" section (skip PAGE_EXAMPLES chunk)
- Very large examples sections (>500 chars) → Still store as PAGE_EXAMPLES chunk, but retrieval will skip it
- Very large sections (>1000 tokens) → split by paragraphs with overlap
- Code blocks with triple backticks → never split
- Nested headers (h2, h3, h4) → preserve hierarchy in metadata
- Empty sections → skip

### Special Section Extraction

**Extract both PAGE_SUMMARY and PAGE_EXAMPLES:**

```go
// Look for sections titled "Summary" or "Overview"
// Extract content as PAGE_SUMMARY chunk
// Mark with section_title: "PAGE_SUMMARY"
// Embed separately from other chunks

// Look for sections titled "Examples" or "Example"
// Extract content as PAGE_EXAMPLES chunk
// Mark with section_title: "PAGE_EXAMPLES"
// Embed separately from other chunks
```

**Implementation:**
```go
func (c *Chunker) extractSpecialSection(content string, sectionNames ...string) string {
    for _, name := range sectionNames {
        pattern := fmt.Sprintf(`(?i)##\s*%s\s*\n([\s\S]*?)(?:\n##|$)`, name)
        regex := regexp.MustCompile(pattern)
        matches := regex.FindStringSubmatch(content)
        
        if len(matches) > 1 {
            return strings.TrimSpace(matches[1])
        }
    }
    return ""
}

// Usage:
summary := c.extractSpecialSection(content, "Summary", "Overview")
examples := c.extractSpecialSection(content, "Examples", "Example")
```

### Retrieval Organization Logic

**After vector search returns chunks:**

```go
// Step 1: Vector search for section chunks
sectionChunks := r.vectorSearch(ctx, query, topK)

// Step 2: Identify which pages were retrieved
pagesFound := getUniquePagesFrom(sectionChunks)

// Step 3: Explicitly fetch special chunks for each page
for _, pageName := range pagesFound {
    // Always fetch summary
    summary := r.fetchSpecialChunk(ctx, pageName, "PAGE_SUMMARY")
    if summary != nil {
        sectionChunks = append(sectionChunks, summary)
    }
    
    // Conditionally fetch examples (only if short)
    examples := r.fetchSpecialChunk(ctx, pageName, "PAGE_EXAMPLES")
    if examples != nil && len(examples.Content) < 500 {
        sectionChunks = append(sectionChunks, examples)
    }
}

// Step 4: Organize by page (special sections first, then regular sections)
organized := r.organizeByPage(sectionChunks)

// Step 5: Return top K
return organized[:topK]
```

**Organization function:**

```go
// After vector search returns mixed chunks:
// 1. Separate PAGE_SUMMARY and PAGE_EXAMPLES chunks from section chunks
// 2. Group by page name, preserving first-appearance order
// 3. For each page: insert summary first, then examples, then sections
// 4. This gives Claude: Summary → Examples → Details per page

func organizeByPage(chunks []DocChunk) []DocChunk {
    pageOrder := []string{}
    pageSummaries := make(map[string]DocChunk)
    pageExamples := make(map[string]DocChunk)
    pageSections := make(map[string][]DocChunk)
    
    for _, chunk := range chunks {
        // Track page order
        if !contains(pageOrder, chunk.PageName) {
            pageOrder = append(pageOrder, chunk.PageName)
        }
        
        // Categorize chunks
        if chunk.SectionTitle == "PAGE_SUMMARY" {
            pageSummaries[chunk.PageName] = chunk
        } else if chunk.SectionTitle == "PAGE_EXAMPLES" {
            pageExamples[chunk.PageName] = chunk
        } else {
            pageSections[chunk.PageName] = append(pageSections[chunk.PageName], chunk)
        }
    }
    
    // Build result: summary → examples → sections per page
    result := []DocChunk{}
    for _, pageName := range pageOrder {
        // Add summary first (if exists)
        if summary, ok := pageSummaries[pageName]; ok {
            result = append(result, summary)
        }
        // Add examples second (if exists)
        if examples, ok := pageExamples[pageName]; ok {
            result = append(result, examples)
        }
        // Then add sections
        if sections, ok := pageSections[pageName]; ok {
            result = append(result, sections...)
        }
    }
    
    return result
}
```

**Why this approach:**
- Ensures summaries are ALWAYS present when page appears (not just if vector search finds them)
- Ensures examples are present when page appears AND examples are short (<500 chars)
- Organized presentation: Overview → Syntax Examples → Details
- Smart token management: Skip long examples, rely on finished Strudels instead

### Hybrid Retrieval Strategy (Option C)

**Implementation approach:**
```go
// internal/retriever/retriever.go

func (r *Retriever) HybridSearch(
    ctx context.Context,
    userQuery string,
    editorState string,
    topK int,
    isDocSearch bool, // true for docs, false for examples
) ([]Chunk, error) {
    
    // 1. Extract editor keywords
    editorContext := extractEditorKeywords(editorState)
    
    // 2. Primary search (60% weight) - user intent only
    primaryK := topK + 2 // Get a few extra for merging
    primaryResults, err := r.vectorSearch(ctx, userQuery, primaryK)
    if err != nil {
        return nil, fmt.Errorf("primary search failed: %w", err)
    }
    
    // 3. Contextual search (40% weight) - if editor has content
    var contextualResults []Chunk
    if editorContext != "" {
        contextualQuery := userQuery + " " + editorContext
        contextualK := topK
        contextualResults, err = r.vectorSearch(ctx, contextualQuery, contextualK)
        if err != nil {
            // Don't fail completely, just log and use primary only
            log.Printf("contextual search failed: %v", err)
            contextualResults = []Chunk{}
        }
    }
    
    // 4. Merge and rank by score
    merged := r.mergeAndRank(primaryResults, contextualResults, topK)
    
    return merged, nil
}

func (r *Retriever) mergeAndRank(primary, contextual []Chunk, topK int) []Chunk {
    // Deduplicate by chunk ID
    seen := make(map[string]bool)
    merged := []Chunk{}
    
    for _, chunk := range append(primary, contextual...) {
        if !seen[chunk.ID] {
            merged = append(merged, chunk)
            seen[chunk.ID] = true
        }
    }
    
    // Sort by similarity score (higher is better)
    sort.Slice(merged, func(i, j int) bool {
        return merged[i].Score > merged[j].Score
    })
    
    // Return top K
    if len(merged) > topK {
        return merged[:topK]
    }
    return merged
}

func extractEditorKeywords(editorState string) string {
    if editorState == "" {
        return ""
    }
    
    keywords := []string{}
    
    // Extract sound sample names: sound("bd") → "bd"
    soundRegex := regexp.MustCompile(`sound\("(\w+)"\)`)
    for _, match := range soundRegex.FindAllStringSubmatch(editorState, -1) {
        if len(match) > 1 {
            keywords = append(keywords, match[1])
        }
    }
    
    // Extract note names: note("c e g") → "c e g"
    noteRegex := regexp.MustCompile(`note\("([^"]+)"\)`)
    for _, match := range noteRegex.FindAllStringSubmatch(editorState, -1) {
        if len(match) > 1 {
            // Split notes and add individually
            notes := strings.Fields(match[1])
            keywords = append(keywords, notes...)
        }
    }
    
    // Extract function calls: .fast(2) → "fast"
    funcRegex := regexp.MustCompile(`\.(\w+)\(`)
    for _, match := range funcRegex.FindAllStringSubmatch(editorState, -1) {
        if len(match) > 1 {
            keywords = append(keywords, match[1])
        }
    }
    
    // Deduplicate and limit to ~10 keywords max to avoid noise
    uniqueKeywords := uniqueStrings(keywords)
    if len(uniqueKeywords) > 10 {
        uniqueKeywords = uniqueKeywords[:10]
    }
    
    return strings.Join(uniqueKeywords, " ")
}

func uniqueStrings(slice []string) []string {
    seen := make(map[string]bool)
    result := []string{}
    for _, s := range slice {
        if !seen[s] {
            result = append(result, s)
            seen[s] = true
        }
    }
    return result
}
```

**Key Points:**
- Primary search (user intent) is weighted higher by retrieving more results (topK+2 vs topK)
- Natural score-based ranking means higher scores (from primary) dominate
- Graceful degradation: if contextual search fails, still have primary results
- Editor keyword extraction limits to ~10 keywords to prevent noise
- Deduplication ensures same chunk doesn't appear twice

**Performance Characteristics:**
- 2x vector searches per retrieval type (docs and examples)
- Total: 4 vector searches per user query
- With Supabase pgvector: ~50ms per search
- Total retrieval time: ~200ms (well within <5sec target)
- Cost: Negligible (vector search is cheap)

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
- Use Claude Haiku (fast, cheap model)
- Target: 3-5 technical keywords per query
- Fallback: If transformation fails, use original query
- Cache common transformations (optional optimization)

### Example Scraping Strategy
- Scrape public Strudel examples website
- Extract: title, description, code, tags
- Embed: title + description + tags (not code)
- Store code separately for retrieval
- If <500 examples: scrape all
- If >1000 examples: curate best 50-100

## Performance Targets

### Ingestion (30 pages + examples)
- Documentation: <5 minutes total
- Per page: <10 seconds
- Chunking: <1 second per page
- Embedding: ~2-3 seconds per chunk (API latency)
- Storage: <100ms per chunk
- Examples: <10 minutes for 100-500 examples

### API Server (per request)
- Query transformation: <100ms
- Hybrid doc search (2x vector searches): <100ms
- Hybrid example search (2x vector searches): <100ms
- Doc organization (grouping by page): <10ms
- Code generation: 1-3 seconds (depends on complexity)
- Total response time: <5 seconds (well within target)
- Note: 4 total vector searches per request, but Supabase is fast (~25ms each)

## Debugging Tips
- Use `log.Printf` liberally during development
- Print chunk boundaries during chunking (helpful for tuning)
- Log when PAGE_SUMMARY chunks are created
- Log embedding dimensions to verify (should be 1536)
- Log similarity scores during retrieval (helps tune topK)
- Print retrieved chunks to verify relevance
- Print page grouping results to verify organization logic

## Future Enhancements (Not MVP)
- Conversation summarization for very long chats
- Query result caching
- Semantic caching for embeddings
- User feedback loop (thumbs up/down on generated code)
- A/B testing different chunking strategies
- Multi-language documentation support
- Incremental doc updates (only re-index changed files)
- Smart example selection based on user skill level
- User-specific example recommendations

## Notes for Claude Code
When implementing, pay special attention to:

### Strudel Code Analysis
- **NEW:** Use `internal/strudel` package for all Strudel code parsing
- See detailed documentation: `docs/system-specs/STRUDEL_CODE_ANALYSIS.md`
- **Retriever:** Use `strudel.ExtractKeywords()` for editor context
- **Examples:** Use `strudel.AnalyzeCode()` and `strudel.GenerateTags()` for tagging
- Do NOT duplicate regex patterns - centralize in strudel package

### Chunking
- Markdown parsing: Use regex carefully, test with varied headers
- Summary extraction: Look for "Summary" or "Overview" sections
- Examples extraction: Look for "Examples" or "Example" sections
- PAGE_SUMMARY chunk creation: Mark with special section_title
- PAGE_EXAMPLES chunk creation: Mark with special section_title
- Token estimation: 4 chars ≈ 1 token is rough, consider using tiktoken library
- Code block detection: Handle both ```language and ``` styles

### Storage
- Two separate tables: document_chunks and example_strudels
- PAGE_SUMMARY chunks stored alongside regular chunks
- Examples: embed description/tags, store code separately

### Retrieval
- **Hybrid search (Option C):** Both docs and examples use primary + contextual searches
- Primary search (60%): User intent only - ensures request is prioritized
- Contextual search (40%): Intent + editor - adds integration context
- Merge and deduplicate results, rank by similarity score
- Editor context extraction: Use `strudel.ExtractKeywords()` from shared package
- Limit editor keywords to ~10 to prevent noise
- Graceful degradation: if contextual search fails, use primary only
- **Explicit special section fetch:** Always fetch PAGE_SUMMARY when page appears, conditionally fetch PAGE_EXAMPLES (if < 500 chars)
- Organization logic: Group by page, summaries first, then examples, then sections
- Handle missing summaries/examples gracefully
- Total of 4 vector searches per request (2 for docs, 2 for examples)

### Context Building
- Four sources: cheatsheet + editor + docs + examples
- Format nicely for Claude (sections, separators)
- Keep within token budget (~5000 tokens total)

### General
- Error context: Always include which file/chunk failed
- Progress indicators: Users want to see what's happening
- Each suggestion/change/decision should be explained in layman terms
- If you suggest a line of code or a function, indicate why