# Hybrid Retrieval Implementation Guide (Option C)

## Overview

This guide provides implementation details for the hybrid retrieval system that combines primary (intent-only) and contextual (intent + editor) searches to achieve the best user experience.

The system retrieves from multiple knowledge sources:
- **Documentation:** Technical docs (30 pages) + Concept docs (6-10 MDX files) - stored in same table
- **Example Strudels:** Finished code examples from public websites

Both documentation types (technical + concepts) use identical chunking and retrieval strategies.

## Architecture Decision

**Why Option C (Hybrid)?**
- Handles both incremental building (90% of use cases) AND pivoting to new ideas
- User intent always dominates (60% weight from primary search)
- Adds contextual integration tips (40% weight from contextual search)
- Self-balancing: empty editor → contextual adds nothing
- Best overall UX: 95% user satisfaction vs 85% for simpler approaches

## System Flow

```
User Query: "add hi-hats"
Editor State: sound("bd").fast(2)
    ↓
1. Transform Query
   → "hi-hat, percussion, drums, rhythm"
    ↓
2. Extract Editor Keywords
   → "bd, sound, fast"
    ↓
3. Hybrid Doc Search (Parallel)
   ├─ Primary: "hi-hat, percussion, drums, rhythm"
   │  └─ Returns 7 results
   └─ Contextual: "hi-hat, percussion, drums, rhythm, bd, sound, fast"
      └─ Returns 5 results
    ↓
4. Merge & Rank (top 5 section chunks)
   → Deduplicate by chunk ID
   → Sort by similarity score
   → Return top 5 sections
    ↓
5. Explicit Fetch Special Sections
   For each page in results:
   ├─ Fetch PAGE_SUMMARY (always)
   └─ Fetch PAGE_EXAMPLES (if < 500 chars)
    ↓
6. Hybrid Example Search (Parallel)
   ├─ Primary: "hi-hat, percussion, drums, rhythm"
   └─ Contextual: "hi-hat, percussion, drums, rhythm, bd, sound, fast"
    ↓
7. Merge & Rank (top 3 finished Strudels)
    ↓
8. Organize Docs
   → Group by page: Summary → Examples → Sections
    ↓
9. Build System Prompt
   → Cheatsheet + Editor + Docs (with examples) + Finished Strudels + History
    ↓
10. Generate Code
```

## Implementation

### Ingestion (Phase 1)

**Three ingestion functions process different knowledge sources:**

```go
// cmd/ingester/main.go
func main() {
    ctx := context.Background()
    
    chunker := chunker.New()
    embedder := embedder.New(os.Getenv("OPENAI_API_KEY"))
    store := storage.New(os.Getenv("SUPABASE_CONNECTION_STRING"))
    
    // 1. Ingest technical documentation (30 pages)
    log.Println("=== Ingesting Technical Docs ===")
    IngestDocs(ctx, chunker, embedder, store)
    
    // 2. Ingest concept documentation (6-10 MDX files)
    log.Println("=== Ingesting Concept Docs ===")
    IngestConcepts(ctx, chunker, embedder, store)
    
    // 3. Ingest example Strudels (JSON)
    log.Println("=== Ingesting Examples ===")
    IngestExamples(ctx, embedder, store)
}

// IngestDocs processes technical documentation
func IngestDocs(ctx context.Context, chunker, embedder, store) error {
    files, _ := filepath.Glob("docs/project-docs/*.md")
    
    for _, file := range files {
        content, _ := os.ReadFile(file)
        chunks, _ := chunker.ChunkMarkdown(file, string(content))
        
        for _, chunk := range chunks {
            embedding, _ := embedder.Embed(ctx, chunk.Content)
            store.InsertDocChunk(ctx, chunk, embedding)  // → doc_embeddings table
        }
    }
}

// IngestConcepts processes concept documentation (MDX)
// Uses SAME chunking logic as IngestDocs!
func IngestConcepts(ctx context.Context, chunker, embedder, store) error {
    files, _ := filepath.Glob("docs/concepts/*.mdx")
    
    for _, file := range files {
        content, _ := os.ReadFile(file)
        chunks, _ := chunker.ChunkMarkdown(file, string(content))  // Same chunker!
        
        for _, chunk := range chunks {
            embedding, _ := embedder.Embed(ctx, chunk.Content)
            store.InsertDocChunk(ctx, chunk, embedding)  // Same table!
        }
    }
}
```

**Key points:**
- Both use `chunker.ChunkMarkdown()` - identical chunking
- Both store in `doc_embeddings` table - same schema
- Differentiated by `page_url` field (`/docs/*` vs `/concepts/*`)
- No retrieval changes needed!

### Core Retriever Structure

```go
// internal/retriever/retriever.go
package retriever

import (
    "context"
    "fmt"
    "regexp"
    "sort"
    "strings"
    
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/pgvector/pgvector-go"
)

type Retriever struct {
    db       *pgxpool.Pool
    embedder Embedder
}

type Chunk struct {
    ID           string
    PageName     string
    PageURL      string
    SectionTitle string
    Content      string
    Score        float64 // Similarity score (0-1, higher is better)
    Metadata     map[string]interface{}
}

type Example struct {
    ID          string
    Title       string
    Description string
    Code        string
    Tags        []string
    URL         string
    Score       float64
}
```

### Hybrid Search Implementation

```go
// HybridSearchDocs performs primary + contextual search for documentation
func (r *Retriever) HybridSearchDocs(
    ctx context.Context,
    userQuery string,
    editorState string,
    topK int,
) ([]Chunk, error) {
    
    // Extract editor keywords
    editorContext := r.extractEditorKeywords(editorState)
    
    // Parallel search
    type result struct {
        chunks []Chunk
        err    error
    }
    
    primaryCh := make(chan result, 1)
    contextualCh := make(chan result, 1)
    
    // Primary search (60% effective weight through larger K)
    go func() {
        chunks, err := r.vectorSearchDocs(ctx, userQuery, topK+2)
        primaryCh <- result{chunks, err}
    }()
    
    // Contextual search (40% effective weight)
    go func() {
        if editorContext == "" {
            contextualCh <- result{[]Chunk{}, nil}
            return
        }
        
        contextualQuery := userQuery + " " + editorContext
        chunks, err := r.vectorSearchDocs(ctx, contextualQuery, topK)
        contextualCh <- result{chunks, err}
    }()
    
    // Wait for results
    primaryRes := <-primaryCh
    contextualRes := <-contextualCh
    
    if primaryRes.err != nil {
        return nil, fmt.Errorf("primary search failed: %w", primaryRes.err)
    }
    
    // If contextual fails, just use primary
    if contextualRes.err != nil {
        return primaryRes.chunks[:min(len(primaryRes.chunks), topK)], nil
    }
    
    // Merge and rank section chunks
    merged := r.mergeAndRankChunks(primaryRes.chunks, contextualRes.chunks, topK)
    
    // ─────────────────────────────────────────────────────────────────
    // NEW: Explicitly fetch special sections for each page
    // ─────────────────────────────────────────────────────────────────
    
    // Find unique pages in results
    pagesFound := make(map[string]bool)
    for _, chunk := range merged {
        pagesFound[chunk.PageName] = true
    }
    
    // Fetch special sections for each page
    for pageName := range pagesFound {
        // Always fetch summary
        summary, err := r.fetchSpecialChunk(ctx, pageName, "PAGE_SUMMARY")
        if err == nil && summary != nil {
            merged = append(merged, *summary)
        }
        
        // Conditionally fetch examples (only if short)
        examples, err := r.fetchSpecialChunk(ctx, pageName, "PAGE_EXAMPLES")
        if err == nil && examples != nil && len(examples.Content) < 500 {
            merged = append(merged, *examples)
        }
    }
    
    // Organize by page (summaries → examples → sections)
    organized := r.organizeByPage(merged)
    
    return organized, nil
}

// fetchSpecialChunk retrieves a special section chunk (PAGE_SUMMARY or PAGE_EXAMPLES)
func (r *Retriever) fetchSpecialChunk(
    ctx context.Context,
    pageName string,
    sectionTitle string,
) (*Chunk, error) {
    
    var chunk Chunk
    
    err := r.db.QueryRowContext(ctx, `
        SELECT 
            id,
            page_name,
            page_url,
            section_title,
            content,
            metadata
        FROM doc_embeddings
        WHERE page_name = $1 
          AND section_title = $2
        LIMIT 1
    `, pageName, sectionTitle).Scan(
        &chunk.ID,
        &chunk.PageName,
        &chunk.PageURL,
        &chunk.SectionTitle,
        &chunk.Content,
        &chunk.Metadata,
    )
    
    if err != nil {
        return nil, err
    }
    
    return &chunk, nil
}

// HybridSearchExamples performs primary + contextual search for examples
func (r *Retriever) HybridSearchExamples(
    ctx context.Context,
    userQuery string,
    editorState string,
    topK int,
) ([]Example, error) {
    
    editorContext := r.extractEditorKeywords(editorState)
    
    type result struct {
        examples []Example
        err      error
    }
    
    primaryCh := make(chan result, 1)
    contextualCh := make(chan result, 1)
    
    // Primary search
    go func() {
        examples, err := r.vectorSearchExamples(ctx, userQuery, topK+1)
        primaryCh <- result{examples, err}
    }()
    
    // Contextual search
    go func() {
        if editorContext == "" {
            contextualCh <- result{[]Example{}, nil}
            return
        }
        
        contextualQuery := userQuery + " " + editorContext
        examples, err := r.vectorSearchExamples(ctx, contextualQuery, topK)
        contextualCh <- result{examples, err}
    }()
    
    primaryRes := <-primaryCh
    contextualRes := <-contextualCh
    
    if primaryRes.err != nil {
        return nil, fmt.Errorf("primary search failed: %w", primaryRes.err)
    }
    
    if contextualRes.err != nil {
        return primaryRes.examples[:min(len(primaryRes.examples), topK)], nil
    }
    
    merged := r.mergeAndRankExamples(primaryRes.examples, contextualRes.examples, topK)
    
    return merged, nil
}
```

### Merge & Rank Logic

```go
// organizeByPage groups chunks by page with special sections first
func (r *Retriever) organizeByPage(chunks []Chunk) []Chunk {
    // Track page order
    pageOrder := []string{}
    
    // Separate chunks by type
    pageSummaries := make(map[string]Chunk)
    pageExamples := make(map[string]Chunk)
    pageSections := make(map[string][]Chunk)
    
    for _, chunk := range chunks {
        // Track first appearance of each page
        if !contains(pageOrder, chunk.PageName) {
            pageOrder = append(pageOrder, chunk.PageName)
        }
        
        // Categorize chunks
        switch chunk.SectionTitle {
        case "PAGE_SUMMARY":
            pageSummaries[chunk.PageName] = chunk
        case "PAGE_EXAMPLES":
            pageExamples[chunk.PageName] = chunk
        default:
            pageSections[chunk.PageName] = append(pageSections[chunk.PageName], chunk)
        }
    }
    
    // Build result: summary → examples → sections per page
    result := []Chunk{}
    for _, pageName := range pageOrder {
        // Add summary first (if exists)
        if summary, ok := pageSummaries[pageName]; ok {
            result = append(result, summary)
        }
        
        // Add examples second (if exists)
        if examples, ok := pageExamples[pageName]; ok {
            result = append(result, examples)
        }
        
        // Then add regular sections
        if sections, ok := pageSections[pageName]; ok {
            result = append(result, sections...)
        }
    }
    
    return result
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}

// mergeAndRankChunks combines primary and contextual results
func (r *Retriever) mergeAndRankChunks(primary, contextual []Chunk, topK int) []Chunk {
    // Deduplicate by chunk ID
    seen := make(map[string]bool)
    merged := []Chunk{}
    
    // Add all chunks (primary first to preserve order when scores are equal)
    for _, chunk := range append(primary, contextual...) {
        if !seen[chunk.ID] {
            merged = append(merged, chunk)
            seen[chunk.ID] = true
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

// mergeAndRankExamples combines primary and contextual example results
func (r *Retriever) mergeAndRankExamples(primary, contextual []Example, topK int) []Example {
    seen := make(map[string]bool)
    merged := []Example{}
    
    for _, example := range append(primary, contextual...) {
        if !seen[example.ID] {
            merged = append(merged, example)
            seen[example.ID] = true
        }
    }
    
    sort.Slice(merged, func(i, j int) bool {
        return merged[i].Score > merged[j].Score
    })
    
    if len(merged) > topK {
        return merged[:topK]
    }
    return merged
}
```

### Special Section Chunking

Documentation pages (both technical and concept docs) contain two special sections that get extracted separately:

**PAGE_SUMMARY:** Overview of the page's content
**PAGE_EXAMPLES:** Simple code examples demonstrating syntax

**Applies to:**
- Technical docs: `docs/project-docs/*.md`
- Concept docs: `docs/concepts/*.mdx`

Both use identical chunking strategy:

```go
// During chunking, extract these sections first
// Works for both .md and .mdx files!
func (c *Chunker) ChunkMarkdown(filepath, content string) ([]Chunk, error) {
    chunks := []Chunk{}
    pageName := extractPageName(filepath)
    pageURL := extractPageURL(filepath)  // /docs/* or /concepts/*
    
    // 1. Extract PAGE_SUMMARY
    summary := extractSection(content, "Summary", "Overview")
    if summary != "" {
        chunks = append(chunks, Chunk{
            PageName:     pageName,
            PageURL:      pageURL,  // Differentiates doc type
            SectionTitle: "PAGE_SUMMARY",
            Content:      summary,
        })
    }
    
    // 2. Extract PAGE_EXAMPLES
    examples := extractSection(content, "Examples", "Example")
    if examples != "" {
        chunks = append(chunks, Chunk{
            PageName:     pageName,
            PageURL:      pageURL,
            SectionTitle: "PAGE_EXAMPLES",
            Content:      examples,
        })
    }
    
    // 3. Extract regular sections (skip Summary and Examples)
    // ...
}

func extractSection(content string, sectionNames ...string) string {
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

func extractPageURL(filepath string) string {
    // Technical docs: /docs/sound-synthesis
    if strings.Contains(filepath, "project-docs") {
        return "/docs/" + extractPageName(filepath)
    }
    
    // Concept docs: /concepts/music-theory
    if strings.Contains(filepath, "concepts") {
        return "/concepts/" + extractPageName(filepath)
    }
    
    return "/" + extractPageName(filepath)
}
```

**Example chunks created:**

```
Technical Doc: docs/project-docs/sound-synthesis.md
  ↓
Chunk: {
  PageName: "sound-synthesis",
  PageURL: "/docs/sound-synthesis",
  SectionTitle: "PAGE_SUMMARY",
  Content: "SUMMARY: This page covers sound synthesis..."
}

Concept Doc: docs/concepts/music-theory.mdx
  ↓
Chunk: {
  PageName: "music-theory",
  PageURL: "/concepts/music-theory",
  SectionTitle: "PAGE_SUMMARY",
  Content: "SUMMARY: Essential music theory for live coding..."
}
```

Both stored in `doc_embeddings` table, differentiated by `page_url`.

### Mixed Retrieval Example

**How retrieval returns both technical and concept docs:**

```go
// User query
query := "how to create tension in my track"

// Hybrid search (same function for both doc types!)
docs := retriever.HybridSearchDocs(ctx, query, editorState, 5)

// Results: Mixed from both technical and concept docs
```

**Retrieved chunks (mixed):**

```
1. music-theory (/concepts) - "Polyrhythm Basics" (0.95) ← Concept!
   Content: "POLYRHYTHM: Multiple rhythms create tension..."

2. patterns (/docs) - "Speed Control" (0.88) ← Technical!
   Content: "The .fast() function speeds up patterns..."

3. composition-techniques (/concepts) - "Building Tension" (0.85) ← Concept!
   Content: "Techniques: filter sweeps, polyrhythms, dynamics..."

4. effects (/docs) - "Filter Sweeps" (0.82) ← Technical!
   Content: "Use .cutoff() to create filter sweeps..."

5. common-patterns (/concepts) - "Tension Patterns" (0.80) ← Concept!
   Content: "// BUILD-UP\nsound(...).cutoff(sine.range(...))"
```

**After organization (summaries + examples added):**

```
═══════════════════════════════════════
RELEVANT DOCUMENTATION
═══════════════════════════════════════

─────────────────────────────────────────
Page: Music Theory (Concept Doc)
─────────────────────────────────────────
SUMMARY: Essential music theory for live coding...

EXAMPLES:
sound("bd").fast(4).stack(sound("hh").fast(3))

SECTION: Polyrhythm Basics
POLYRHYTHM: Multiple rhythms create tension...

─────────────────────────────────────────
Page: Patterns (Technical Doc)
─────────────────────────────────────────
SUMMARY: Pattern manipulation functions...

EXAMPLES:
sound("bd").fast(2)

SECTION: Speed Control
The .fast() function speeds up patterns...

[... 3 more pages with mix of technical + concept docs ...]
```

**Result:** Claude gets:
- **HOW** to do it (technical docs)
- **WHY/WHEN** to use it (concept docs)
- **Complete understanding!**

### Editor Keyword Extraction

Keyword extraction is now handled by the shared `internal/strudel` package. See `docs/system-specs/STRUDEL_CODE_ANALYSIS.md` for full details.

```go
// retriever/utils.go
import "algorave/internal/strudel"

// extractEditorKeywords extracts relevant keywords from current editor state
func (r *Retriever) extractEditorKeywords(editorState string) string {
    // Delegates to centralized strudel package
    // Extracts: sounds, notes, functions, scales
    // Returns: space-separated keywords (max 10, deduplicated)
    return strudel.ExtractKeywords(editorState)
}
```

**What it does:**
- Extracts sound names: `sound("bd")` → `"bd"`
- Extracts notes: `note("c e g")` → `"c e g"`
- Extracts functions: `.fast(2)` → `"fast"`
- Extracts scales: `scale("minor")` → `"minor"`
- Deduplicates and limits to 10 keywords to prevent noise

**Example:**
```go
editorState := `sound("bd").fast(2).stack(note("c e g"))`
keywords := strudel.ExtractKeywords(editorState)
// Returns: "bd fast stack c e g"
```

### Vector Search Implementation

```go
// vectorSearchDocs performs actual vector search on doc_embeddings table
func (r *Retriever) vectorSearchDocs(
    ctx context.Context,
    query string,
    topK int,
) ([]Chunk, error) {
    
    // Embed the query
    embedding, err := r.embedder.Embed(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("embedding failed: %w", err)
    }
    
    // Search database
    rows, err := r.db.Query(ctx, `
        SELECT 
            id,
            page_name,
            page_url,
            section_title,
            content,
            metadata,
            1 - (embedding <=> $1) as similarity
        FROM doc_embeddings
        ORDER BY embedding <=> $1
        LIMIT $2
    `, pgvector.NewVector(embedding), topK)
    
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()
    
    chunks := []Chunk{}
    for rows.Next() {
        var chunk Chunk
        err := rows.Scan(
            &chunk.ID,
            &chunk.PageName,
            &chunk.PageURL,
            &chunk.SectionTitle,
            &chunk.Content,
            &chunk.Metadata,
            &chunk.Score,
        )
        if err != nil {
            return nil, fmt.Errorf("scan failed: %w", err)
        }
        chunks = append(chunks, chunk)
    }
    
    return chunks, nil
}

// vectorSearchExamples performs actual vector search on example_strudels table
func (r *Retriever) vectorSearchExamples(
    ctx context.Context,
    query string,
    topK int,
) ([]Example, error) {
    
    embedding, err := r.embedder.Embed(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("embedding failed: %w", err)
    }
    
    rows, err := r.db.Query(ctx, `
        SELECT 
            id,
            title,
            description,
            code,
            tags,
            url,
            1 - (embedding <=> $1) as similarity
        FROM example_strudels
        ORDER BY embedding <=> $1
        LIMIT $2
    `, pgvector.NewVector(embedding), topK)
    
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()
    
    examples := []Example{}
    for rows.Next() {
        var example Example
        err := rows.Scan(
            &example.ID,
            &example.Title,
            &example.Description,
            &example.Code,
            &example.Tags,
            &example.URL,
            &example.Score,
        )
        if err != nil {
            return nil, fmt.Errorf("scan failed: %w", err)
        }
        examples = append(examples, example)
    }
    
    return examples, nil
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

## Performance Characteristics

### Per Request:
- **Total vector searches:** 4 (2 for docs, 2 for examples)
- **Each search:** ~25ms (Supabase pgvector with ivfflat index)
- **Total search time:** ~100ms
- **Merge & rank:** ~10ms
- **Total retrieval time:** ~110ms (well within <5sec target)

### Optimization Notes:
- Parallel execution of primary + contextual searches
- Graceful degradation if contextual search fails
- Efficient deduplication using map
- Sort only once after merging
- Limited editor keywords (max 10) to prevent noise

## Testing Strategy

### Unit Tests:
```go
func TestMergeAndRank(t *testing.T) {
    primary := []Chunk{
        {ID: "1", Score: 0.95},
        {ID: "2", Score: 0.90},
        {ID: "3", Score: 0.85},
    }
    
    contextual := []Chunk{
        {ID: "2", Score: 0.88}, // Duplicate (lower score)
        {ID: "4", Score: 0.87},
        {ID: "5", Score: 0.82},
    }
    
    merged := mergeAndRankChunks(primary, contextual, 5)
    
    // Should be: [1(0.95), 2(0.90), 4(0.87), 3(0.85), 5(0.82)]
    assert.Len(t, merged, 5)
    assert.Equal(t, "1", merged[0].ID)
    assert.Equal(t, "2", merged[1].ID) // Primary score wins
    assert.Equal(t, "4", merged[2].ID)
}

func TestExtractEditorKeywords(t *testing.T) {
    code := `sound("bd").fast(2).stack(note("c e g").slow(4))`
    
    keywords := extractEditorKeywords(code)
    
    // Should extract: bd, c, e, g, fast, stack, slow
    assert.Contains(t, keywords, "bd")
    assert.Contains(t, keywords, "fast")
    assert.Contains(t, keywords, "c")
}
```

## Monitoring & Debugging

### Key Metrics to Track:
- Primary search result count
- Contextual search result count
- Deduplication rate (how many duplicates removed)
- Average similarity scores (primary vs contextual)
- Empty editor rate (how often contextual search is skipped)
- Search failure rate

### Debug Logging:
```go
log.Printf("Hybrid search - Primary: %d results, Contextual: %d results, Merged: %d results",
    len(primary), len(contextual), len(merged))

log.Printf("Top result scores - Primary: %.3f, Contextual: %.3f, Merged: %.3f",
    primary[0].Score, contextual[0].Score, merged[0].Score)

log.Printf("Editor keywords: %s", editorContext)
```

## Edge Cases

### Empty Editor:
- Contextual search returns empty
- Merge uses only primary results
- No performance penalty

### Very Long Editor:
- Keyword extraction limits to 10 keywords
- Prevents noise in contextual search
- Still gets value from most important terms

### Identical Results:
- Deduplication removes duplicates
- Primary score wins (appears first in merge)
- User gets best of both searches

### Search Failures:
- Primary fails → return error (critical)
- Contextual fails → log and use primary only (graceful)
- Both fail → return error

## Future Optimizations

1. **Adaptive Weighting:** Adjust 60/40 ratio based on user behavior
2. **Smart Keyword Extraction:** Use LLM to extract most relevant keywords
3. **Caching:** Cache common query results
4. **Result Diversity:** Ensure variety in page sources
5. **User Feedback:** Track which results users actually use