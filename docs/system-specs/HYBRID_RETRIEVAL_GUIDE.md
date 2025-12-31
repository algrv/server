# Hybrid Retrieval Implementation Guide

## Overview

Combines vector similarity search with BM25 keyword search for optimal retrieval. Retrieves from:
- **Documentation:** Technical docs + Concept docs (same table, different `page_url`)
- **Example Strudels:** Finished code examples

## Architecture Decision

**Why Hybrid (Vector + BM25)?**
- Vector (70%): Semantic similarity for intent
- BM25 (30%): Keyword matching for exact terms
- Best UX: 95% user satisfaction

## System Flow

```
User Query + Editor State
    ↓
1. Transform Query (Haiku) → technical keywords
    ↓
2. Parallel Search:
   ├─ Vector Search (transformed query)
   └─ BM25 Search (original query)
    ↓
3. Merge & Rank (RRF fusion)
    ↓
4. Fetch Special Sections:
   ├─ PAGE_SUMMARY (always)
   └─ PAGE_EXAMPLES (if < 500 chars)
    ↓
5. Organize by Page (summary → examples → sections)
    ↓
6. Build System Prompt + Generate
```

## Core Types

```go
type Retriever struct {
    db   *pgxpool.Pool
    llm  LLM
    topK int
}

type SearchResult struct {
    ID, PageName, PageURL, SectionTitle, Content string
    Similarity float32
}

type ExampleResult struct {
    ID, Title, Description, Code, URL string
    Tags       []string
    Similarity float32
}
```

## Key Functions

### HybridSearchDocs

```
func HybridSearchDocs(ctx, userQuery, editorState, topK) []SearchResult:
    searchQuery = llm.TransformQuery(userQuery)  // fallback to original on error

    // Parallel searches
    vectorResults = VectorSearch(searchQuery, topK+5)
    bm25Results = BM25Search(userQuery, topK+5)

    merged = mergeVectorAndBM25(vectorResults, bm25Results, topK)
    organized = organizeByPage(merged)  // adds summaries/examples

    return organized
```

### HybridSearchExamples

```
func HybridSearchExamples(ctx, userQuery, editorState, topK) []ExampleResult:
    searchQuery = llm.TransformQuery(userQuery)

    // Parallel searches (70/30 weighting)
    vectorResults = SearchExamples(searchQuery, topK+5)
    bm25Results = BM25SearchExamples(userQuery, topK+5)

    return mergeVectorAndBM25Examples(vectorResults, bm25Results, topK)
```

### Merge & Rank (Weighted Score Fusion)

```
func mergeVectorAndBM25(vector, bm25 []SearchResult, topK) []SearchResult:
    // Weighted similarity score fusion
    scores = map[id]float64{}

    for _, result in vector:
        scores[result.ID] = result.Similarity * 0.7  // vector weight

    for _, result in bm25:
        if scores[result.ID] exists:
            scores[result.ID] += result.Similarity * 0.3  // add bm25 weight
        else:
            scores[result.ID] = result.Similarity * 0.3

    // Sort by combined score, return top K
```

### Organize By Page

```
func organizeByPage(chunks []SearchResult) []SearchResult:
    // For each unique page in results:
    //   1. Fetch PAGE_SUMMARY (always)
    //   2. Fetch PAGE_EXAMPLES (if < 500 chars)
    //   3. Order: summary → examples → sections

    return organized
```

## Special Section Chunking

During ingestion, extract from each doc:
- `PAGE_SUMMARY` chunk (from "Summary" or "Overview" section)
- `PAGE_EXAMPLES` chunk (from "Examples" section)
- Regular section chunks

Both technical (`/docs/*`) and concept (`/concepts/*`) docs use identical chunking.

## Editor Keyword Extraction

Uses `internal/strudel` package:

```go
func extractEditorKeywords(editorState string) string {
    return strudel.ExtractKeywords(editorState)
    // Extracts: sounds, notes, functions (max 10, deduplicated)
}
```

## Performance

- **Vector searches:** ~25ms each (Supabase pgvector)
- **BM25 searches:** ~15ms each
- **Total retrieval:** ~100ms
- **Merge/organize:** ~10ms

## Database Queries

### Vector Search
```sql
SELECT id, page_name, page_url, section_title, content,
       1 - (embedding <=> $1) as similarity
FROM doc_embeddings
ORDER BY embedding <=> $1
LIMIT $2
```

### BM25 Search
```sql
SELECT id, page_name, page_url, section_title, content,
       ts_rank(content_tsvector, plainto_tsquery($1)) as rank
FROM doc_embeddings
WHERE content_tsvector @@ plainto_tsquery($1)
ORDER BY rank DESC
LIMIT $2
```

## Edge Cases

- **Empty editor:** Skip context-based augmentation
- **Transform fails:** Use original query
- **BM25 fails:** Use vector only (graceful degradation)
- **No results:** Return empty (agent handles this)
