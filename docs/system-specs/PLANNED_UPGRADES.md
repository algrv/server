# Planned Architecture Upgrades

Potential improvements evaluated but not yet implemented. Each item includes rationale for deferral.

---

## Retrieval: Context + Intent Hybrid Search

**Status:** Deferred
**Reason:** Current intent-only search produces accurate results. LLM receives editor state directly and can combine context itself.

**Current Implementation:**
```
User Query → BM25 + Vector (intent only) → Results
```

**Potential Upgrade:**
```
User Query + Editor State → BM25 + Vector (intent + context) → Results
                         ↓
              Merge with intent-only results
```

**Interface Ready:** The `editorState` parameter exists in `Retriever.HybridSearchDocs()` and `HybridSearchExamples()` but is currently unused.

**When to Revisit:**
- If users report irrelevant retrieval results
- If complex multi-element queries underperform
- If LLM struggles to combine retrieved examples with editor context

---

## Proposed Architecture Summary

User Request: "add delay"
Editor: s("bd sd").fast(2)
                │
                ▼
┌─────────────────────────────────────────────────────────────┐
│                    DOCS RETRIEVAL                           │
├─────────────────────────────────────────────────────────────┤
│  Intent Only:           Intent + Context:                   │
│  ┌─────────────────┐    ┌─────────────────┐                 │
│  │ Vector("delay") │    │ Vector("delay   │                 │
│  │ BM25("delay")   │    │   bd sd fast")  │                 │
│  └────────┬────────┘    │ BM25("delay     │                 │
│           │             │   bd sd fast")  │                 │
│           │             └────────┬────────┘                 │
│           └──────────┬───────────┘                          │
│                      ▼                                      │
│              Merge & Rank                                   │
└─────────────────────────────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────────────────────────┐
│                  EXAMPLES RETRIEVAL                         │
│                  (Same pattern)                             │
└─────────────────────────────────────────────────────────────┘

### Analysis

| Aspect       | Assessment                                                    |
|--------------|---------------------------------------------------------------|
| Logic        | ✅ Sound - intent ensures relevance, context adds specificity |
| Latency      | ✅ OK - all 4 searches run in parallel                        |
| Empty editor | ⚠️ Skip context search when empty (avoid duplicate work)      |
| Merging      | ✅ Existing RRF merge extends to 4 sources                    |

### Cost Consideration

Each search with context needs an extra embedding call:
- Current: 1 embedding per search type
- Proposed: 2 embeddings per search type (when editor has content)

But this is minimal (~$0.0001 per embedding) and only when editor isn't empty.

### Edge Case

Editor: s("bd sd")           // drums
Query: "add a melody"        // completely different

Intent+context search → "melody bd sd drums" might:
- Help: Find examples showing melody + drums together (useful!)
- Hurt: Dilute pure melody results

*Solution:* Weight intent-only slightly higher (60%) than context (40%), so pure intent results still dominate.
