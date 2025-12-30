# Code Annotation System Specification

## Implementation Status

### Phase 1: Enhanced Instructions - COMPLETED

**Status:** Validated & Sufficient

Enhanced the system prompt in `/internal/agent/prompt.go` with:
1. Request type classification (ADDITIVE, MODIFICATION, DELETION, QUESTIONS)
2. 4-step surgical precision process for modifications
3. Few-shot examples for each request type

**Result:** Surgical accuracy achieved without additional API calls.

### Phase 2: Code Annotation - ON HOLD

Not needed. Phase 1 achieves required precision without:
- Additional API calls (+10% cost)
- Added latency
- Implementation complexity

**When to revisit:**
- Precision issues emerge with edge cases
- Users request preview of changes
- Metrics show frequent unwanted modifications

---

## Phase 2 Design (Reference)

### Concept

Two-stage pipeline where a small LLM (Haiku) annotates code with editing directives before the main LLM (Sonnet) generates.

### Flow

```
User Query → Classify Request Type
                ↓
    NEW/ADD → Skip annotation
    MODIFY/DELETE → Annotate Code (Haiku)
                        ↓
                    Confidence Check
                        ↓
                    Low → Fallback to non-annotated
                    High → Use annotated code
                        ↓
                    Generate (Sonnet)
```

### Annotation Directives

```javascript
// MODIFY THIS: [reason]   → Change this line/block
// DELETE THIS: [reason]   → Remove entirely
// ADD NEW CODE BELOW:     → Insert new code here
// KEEP AS-IS              → Don't change
```

### Interface

```go
type Annotator interface {
    AnnotateCode(ctx, req AnnotationRequest) (*AnnotationResult, error)
}

type AnnotationRequest struct {
    UserQuery   string
    CurrentCode string
    History     []Message
}

type AnnotationResult struct {
    AnnotatedCode string
    Annotations   []Annotation
    Confidence    float64  // 0.0-1.0
    UseFallback   bool
}
```

### Request Classification

```
func classifyRequest(query, editorState) RequestType:
    if editorState is empty:
        return NEW

    if query contains "make", "change", "adjust", "fix":
        return MODIFY

    if query contains "remove", "delete", "drop":
        return DELETE

    if query contains "add", "create", "build":
        return ADD

    return NEW  // default
```

### Confidence Scoring

Score based on:
- Number of annotations (too many = uncertain)
- Reason quality (short = vague)
- Conflicting directives

Threshold: 0.7 (below = fallback to non-annotated)

### Cost

- Without annotation: $0.003/request (Sonnet only)
- With annotation: $0.0033/request (+10% for Haiku call)

---

## Related

- [RAG_ARCHITECTURE.md](./RAG_ARCHITECTURE.md)
- [HYBRID_RETRIEVAL_GUIDE.md](./HYBRID_RETRIEVAL_GUIDE.md)
- `/internal/agent/prompt.go` - Enhanced instructions
