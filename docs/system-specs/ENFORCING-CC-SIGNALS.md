# Enforcing CC Signals

## Overview

Behavioral detection system that protects CC Signal restrictions (especially `no-ai`) from being bypassed through copy-paste. The system uses a multi-layered approach:

1. **Large delta detection** - Detects sudden large code changes (paste behavior)
2. **Ownership validation** - Checks if code belongs to the user
3. **Public content validation** - Checks if code is from allowed public sources
4. **Fingerprint similarity detection** - Uses SimHash + LSH to detect similar content

When protected code is detected, the AI agent is temporarily blocked until the user makes significant edits (~20%), demonstrating genuine engagement with the code.

## Problem Statement

Users can bypass CC Signal restrictions by:

1. Viewing public code with `no-ai` signal
2. Copy-pasting into their own editor
3. Requesting AI assistance on the copied code

This system detects paste behavior and temporarily blocks AI access until the user demonstrates genuine engagement with the code through significant edits.

## How It Works

```
User pastes code → WS detects large code_update → Validates against sources
                                                          ↓
                    ┌─────────────────────────────────────────────────────────┐
                    │ VALIDATION CHECKS (in order)                            │
                    │                                                         │
                    │ 1. User's own strudel? (exact match) → No lock          │
                    │ 2. Public strudel that allows AI? → No lock             │
                    │ 3. Public strudel with no-ai? → LOCK (sticky)           │
                    │ 4. Fingerprint similar to no-ai content? → LOCK (sticky)│
                    │ 5. External paste (no match)? → LOCK (temporary)        │
                    └─────────────────────────────────────────────────────────┘
                                                          ↓
User requests AI → REST checks paste lock → If locked: reject with message
                                                          ↓
User makes ~20% edits → WS detects significant change → Removes lock
                                                          ↓
User requests AI → REST checks paste lock → Unlocked → Proceed
```

## Fingerprint Similarity Detection

The system uses **SimHash + Locality-Sensitive Hashing (LSH)** to detect content similar to protected `no-ai` strudels, even if slightly modified.

### How Fingerprinting Works

```
┌─────────────────────────────────────────────────────────────────┐
│                     FINGERPRINT GENERATION                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Input: "d1 $ sound \"bd sd\" # speed 2"                        │
│           ↓                                                     │
│  Shingles (3-char): ["d1 ", "1 $", " $ ", "$ s", " so", ...]    │
│           ↓                                                     │
│  Hash each shingle → weighted bit vectors                       │
│           ↓                                                     │
│  Combine vectors → 64-bit SimHash fingerprint                    │
│           ↓                                                     │
│  Output: 0xABCD1234EFGH5678                                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                     LSH INDEX STRUCTURE                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  64-bit fingerprint split into 4 bands of 16 bits each           │
│                                                                 │
│  Band 1: bits 0-15   → bucket[0x1234] = [work1, work5, ...]     │
│  Band 2: bits 16-31  → bucket[0xABCD] = [work1, work3, ...]     │
│  Band 3: bits 32-47  → bucket[0xEFGH] = [work2, work1, ...]     │
│  Band 4: bits 48-63  → bucket[0x5678] = [work1, work7, ...]     │
│                                                                 │
│  Query: Find candidates sharing ANY band → verify Hamming dist  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Similarity Threshold

- **Match threshold**: Hamming distance ≤ 10 bits (out of 64)
- **Equivalent to**: ~84% similarity
- **Minimum content length**: 200 characters (prevents false positives on short patterns)

### Index Lifecycle

```
Server startup:
  └→ Load all no-ai strudels from user_strudels (code length ≥ 200)
  └→ Compute fingerprints and populate LSH index

Runtime - Strudel created:
  └→ If cc_signal = "no-ai" and len(code) ≥ 200
  └→ Compute fingerprint and add to index

Runtime - Strudel updated:
  └→ Remove old fingerprint from index
  └→ If new cc_signal = "no-ai" and len(code) ≥ 200
  └→ Compute new fingerprint and add to index

Runtime - Strudel deleted:
  └→ Remove fingerprint from index
```

## Lock Types

| Type               | Description                          | Can Unlock?      | Re-locks on similar paste? |
| ------------------ | ------------------------------------ | ---------------- | -------------------------- |
| **No Lock**        | Content recognized as legitimate     | N/A              | No                         |
| **Temporary Lock** | External paste, no fingerprint match  | Yes (~20% edits) | No                         |
| **Sticky Lock**    | Matches no-ai fingerprint             | Yes (~20% edits) | **Yes**                    |

## Paste Lock Decision Table

| Action                                | Large Delta? | Ownership? | Public Match? | Fingerprint? | Result         |
| ------------------------------------- | ------------ | ---------- | ------------- | ------------ | -------------- |
| User loads their saved strudel        | Yes          | ✓ Match    | -             | -            | No lock        |
| User forks public strudel (allows AI) | Yes          | ✗          | ✓ allows AI   | -            | No lock        |
| User forks public strudel (no-ai)     | Yes          | ✗          | ✓ no-ai       | -            | Sticky lock    |
| User pastes similar to no-ai content  | Yes          | ✗          | ✗             | ✓ Match      | Sticky lock    |
| User pastes external code             | Yes          | ✗          | ✗             | ✗            | Temporary lock |
| User types code gradually             | No           | -          | -             | -            | No lock        |

## Design Decisions

| Decision                    | Value                            | Rationale                                                  |
| --------------------------- | -------------------------------- | ---------------------------------------------------------- |
| Paste threshold             | 200+ chars OR 10+ lines delta    | Normal typing is 1-5 chars; paste is hundreds+             |
| Unlock threshold            | ~20% edit distance (Levenshtein) | Requires genuine engagement with code                      |
| Similarity threshold        | 10 bits Hamming distance (~84%)  | Catches minor modifications while avoiding false positives  |
| Min content for fingerprint  | 200 characters                   | Short patterns too common, would cause false blocks        |
| Lock TTL                    | Configurable (default 30 min)     | Auto-cleanup for disconnected sessions                     |
| Shingle size                | 3 characters                     | Good balance of granularity and noise resistance           |
| LSH bands                   | 4 bands of 16 bits               | Optimal for ~84% similarity detection                      |

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────────┐
│                        CC Signals System                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────────┐     ┌──────────────────────┐          │
│  │  Fingerprint Index   │     │   Lock Store         │          │
│  │  (In-Memory)         │     │   (Redis)            │          │
│  ├──────────────────────┤     ├──────────────────────┤          │
│  │ • LSH buckets        │     │ • Session lock state │          │
│  │ • SimHash values     │     │ • Baseline code      │          │
│  │ • Loaded at startup  │     │ • TTL expiration     │          │
│  │   from user_strudels │     │                      │          │
│  └──────────────────────┘     └──────────────────────┘          │
│           │                            │                        │
│           ▼                            ▼                        │
│  "Is this code similar         "Is this session locked?         │
│   to protected work?"           Set/remove/refresh lock"        │
│                                                                 │
│  ┌──────────────────────┐     ┌──────────────────────┐          │
│  │  Content Validator   │     │   Detector           │          │
│  ├──────────────────────┤     ├──────────────────────┤          │
│  │ • Ownership check    │     │ • Orchestrates all   │          │
│  │ • Public content     │     │   validation checks  │          │
│  │   check              │     │ • Manages lock state │          │
│  └──────────────────────┘     └──────────────────────┘          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Redis Keys

```
ccsignals:paste_lock:{sessionID}     → "1" (with TTL)
ccsignals:paste_baseline:{sessionID} → <code at time of paste> (with TTL)
```

### Data Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              FRONTEND                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│  Editor paste event → Mark next update source as 'paste'                    │
│  Code change → sendCodeUpdate({ code, source })                             │
│  AI request → POST /agent/generate { session_id, ... }                      │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              BACKEND                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│  WebSocket Hub:                                                             │
│    1. Receive code_update                                                   │
│    2. Detect large delta (behavioral detection)                             │
│    3. Run ccsignals.Detector.DetectPaste():                                 │
│       a. Check ownership (exact match to user's strudels)                   │
│       b. Check public content (exact match, respect CC signals)             │
│       c. Check fingerprint similarity (LSH query)                            │
│    4. Set/remove paste lock in Redis via LockStore                          │
│    5. Notify client of lock status via WebSocket                            │
│                                                                             │
│  REST API:                                                                  │
│    1. Receive /agent/generate with session_id                               │
│    2. Validate session is active                                            │
│    3. Check paste lock in Redis                                             │
│    4. If locked: return 403                                                 │
│    5. If unlocked: proceed with AI generation                               │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Known Loopholes & Exploits

### HIGH RISK

#### 1. Chunk-by-Chunk Paste

**Description**: User pastes protected content in small chunks, each below the detection threshold.

```
Protected code: 500 characters total

Attack:
  Paste 1: chars 1-150    → No lock (< 200 char threshold)
  Paste 2: chars 151-300  → No lock (delta only 150 chars)
  Paste 3: chars 301-450  → No lock (delta only 150 chars)
  Paste 4: chars 451-500  → No lock (delta only 50 chars)

Result: Full protected code pasted, never triggered detection
```

**Risk**: 🔴 HIGH - Easy to execute, complete bypass

**Potential Fix**:

```go
// Track cumulative delta across session, not just per-update
type SessionDeltaTracker struct {
    OriginalCode    string    // Code at session start or last AI request
    CumulativeAdded int       // Total chars added since baseline
    LastResetTime   time.Time // When baseline was last reset
}

// On each code update:
// 1. Calculate cumulative change from session baseline
// 2. If cumulative exceeds threshold, trigger paste detection
// 3. Reset baseline after successful AI request or significant time gap
```

#### 2. Save-Then-Paste Laundering

**Description**: User copies protected code, saves it as their own strudel with a permissive CC signal, then uses ownership check to bypass protection.

```
User A wants to steal User B's no-ai code

Attack:
  1. Copy User B's no-ai code
  2. Save as own strudel with cc_signal: "cc-cr"
  3. Later, paste into session
  4. Ownership check passes (it's "their" strudel now)
  5. No fingerprint match (only no-ai strudels are indexed)

Result: Complete bypass of protection
```

**Risk**: 🔴 HIGH - Easy to execute, complete bypass

**Potential Fixes**:

Option A - Index all strudels:

```go
// Index ALL strudels, not just no-ai
// On paste, check fingerprint against all indexed content
// If match found with more restrictive signal in DB, apply that signal's rules
// Con: Much larger index, potential performance impact
```

Option B - Fingerprint on save:

```go
// When user saves a strudel, check fingerprint against existing no-ai content
// If similar to no-ai content, either:
//   - Block save with warning
//   - Force inherit the no-ai signal
//   - Flag for review
```

Option C - Track content origin:

```go
// Store "first_seen_fingerprint" timestamp globally
// If user's strudel fingerprint matches older no-ai content, flag it
// Requires additional storage and complexity
```

#### 3. Collaborative Laundering

**Description**: Malicious user asks innocent friend to save code to their account with permissive signal.

```
Attack:
  1. User A copies no-ai code from User B
  2. User A sends code to friend User C
  3. User C saves as own strudel with cc-cr signal
  4. User A forks from User C's public strudel
  5. Fork appears legitimate

Result: Laundered through third party
```

**Risk**: 🔴 HIGH - Requires coordination but undetectable

**Potential Fix**: Same as #2 - fingerprint checking on save would catch this

### MEDIUM RISK

#### 4. TTL Expiration Wait

**Description**: User waits for lock to expire instead of making edits.

```
Attack:
  1. Paste protected code → LOCKED
  2. Don't edit, just wait for TTL (default 30 min)
  3. Lock expires
  4. Request AI assistance

Result: Bypassed edit requirement by waiting
```

**Risk**: 🟡 MEDIUM - Easy but time-consuming, may be acceptable

**Potential Fixes**:

```go
// Option A: Remove TTL-based expiration entirely
// Lock only removed by significant edits
// Con: Orphaned locks if user disconnects

// Option B: Require edit check even after TTL
// TTL only removes the lock key, but AI request re-checks code
// If code still matches protected content, re-lock

// Option C: Longer TTL with activity-based refresh
// TTL only counts down during inactivity
// Active sessions keep lock indefinitely
```

#### 5. Whitespace/Comment Dilution

**Description**: Add non-functional content to change fingerprint while preserving code functionality.

```
Original no-ai code (200 chars):
  d1 $ sound "bd sd" # speed 2

Diluted version (400 chars):
  -- This is my original work
  -- I wrote this myself
  -- Adding lots of comments
  d1 $ sound "bd sd" # speed 2
  -- More comments here
  -- To change the fingerprint

Result: Fingerprint differs enough to evade detection
```

**Risk**: 🟡 MEDIUM - Requires effort, may be caught by similarity threshold

**Potential Fixes**:

```go
// Option A: Strip comments/whitespace before fingerprinting
// Normalize code before computing SimHash
// Con: Language-specific, complex to implement

// Option B: Multiple fingerprints
// Generate fingerprints for both raw and normalized versions
// Match against either

// Option C: Lower similarity threshold
// Catch more variations at cost of more false positives
```

#### 6. Semantic-Preserving Transforms

**Description**: Restructure code to produce same output with different fingerprint.

```
Original:
  d1 $ sound "bd sd hh" # speed 2 # gain 0.8

Transformed:
  let drums = "bd sd hh"
  let spd = 2
  let vol = 0.8
  d1 $ sound drums # speed spd # gain vol

Result: Same musical output, different fingerprint
```

**Risk**: 🟡 MEDIUM - Requires skill/effort, arguably transformative use

**Potential Fixes**:

```go
// Option A: AST-based fingerprinting
// Parse code into AST, fingerprint the structure
// Con: Very complex, language-specific

// Option B: Output-based detection
// Compare actual audio output (impossible in practice)

// Option C: Accept as transformative use
// If user puts in effort to restructure, they've engaged with the code
// This may be acceptable behavior
```

#### 7. Unicode Homoglyph Substitution

**Description**: Replace ASCII characters with visually identical Unicode characters.

```
Original:
  d1 $ sound "bd"

With homoglyphs:
  d1 $ ѕоund "bd"  // Cyrillic 'ѕ' and 'о' instead of Latin

Result: Looks identical but completely different fingerprint
```

**Risk**: 🟡 MEDIUM - Obscure technique, code may not execute

**Potential Fix**:

```go
// Normalize Unicode to ASCII before fingerprinting
// Use unicode.NFKC normalization
import "golang.org/x/text/unicode/norm"
normalized := norm.NFKC.String(code)
fingerprint := computeSimHash(normalized)
```

### LOW RISK

#### 8. Multiple Account Laundering

**Description**: Create multiple accounts to pass code between them.

```
Attack:
  1. Account A copies no-ai code
  2. Account A saves to Account B (alt account)
  3. Account B makes public with cc-cr
  4. Account A forks "legitimately"
```

**Risk**: 🟢 LOW - Requires multiple accounts, detectable via IP/device fingerprinting

**Potential Fix**: Standard multi-account detection (outside scope of CC signals)

#### 9. New Session Per Attempt

**Description**: Create new session for each paste attempt to get fresh start.

```
Attack:
  1. Paste in session A → LOCKED
  2. Create session B, paste same code → LOCKED
  3. Create session C... → Still LOCKED each time
```

**Risk**: 🟢 LOW - No actual bypass, just annoyance

**Notes**: Lock applies immediately each time, no benefit to attacker

#### 10. Timing Race Condition

**Description**: Paste and immediately request AI before lock is set.

```
Attack:
  1. Paste code via WebSocket
  2. Immediately POST /agent/generate
  3. Hope AI request arrives before lock is set
```

**Risk**: 🟢 LOW - Lock is set synchronously in WebSocket handler

**Notes**: WebSocket processes code_update and sets lock before responding. AI endpoint checks lock in Redis. Race window is negligible.

## Loophole Risk Summary

| Loophole                   | Risk      | Effort to Exploit | Effort to Fix | Priority |
| -------------------------- | --------- | ----------------- | ------------- | -------- |
| Chunk-by-chunk paste       | 🔴 High   | Low               | Medium        | P0       |
| Save-then-paste laundering | 🔴 High   | Low               | High          | P0       |
| Collaborative laundering   | 🔴 High   | Medium            | High          | P1       |
| TTL expiration             | 🟡 Medium | Low               | Low           | P1       |
| Whitespace dilution        | 🟡 Medium | Medium            | Medium        | P2       |
| Semantic transforms        | 🟡 Medium | High              | Very High     | P3       |
| Unicode homoglyphs         | 🟡 Medium | Medium            | Low           | P2       |
| Multiple accounts          | 🟢 Low    | High              | N/A           | P3       |
| New session spam           | 🟢 Low    | Low               | N/A           | -        |
| Timing race                | 🟢 Low    | Low               | N/A           | -        |

## Files

### Backend - Core CC Signals

| File                                     | Purpose                               |
| ---------------------------------------- | ------------------------------------- |
| `internal/ccsignals/detector.go`         | Main detection orchestrator           |
| `internal/ccsignals/lsh.go`              | LSH index and IndexedFingerprintStore |
| `internal/ccsignals/simhash.go`          | SimHash fingerprint computation        |
| `internal/ccsignals/levenshtein.go`      | Edit distance for unlock detection    |
| `internal/ccsignals/redis_store.go`      | Redis-backed lock storage             |
| `internal/ccsignals/memory_store.go`     | In-memory lock storage (testing)      |
| `internal/ccsignals/algopatterns_adapter.go` | Strudel repository adapter            |
| `internal/ccsignals/types.go`            | Shared types and config                |

### Backend - Integration

| File                             | Purpose                                 |
| -------------------------------- | --------------------------------------- |
| `cmd/server/ccsignals.go`        | System initialization, strudel indexing |
| `cmd/server/server.go`           | Wires CCSignalsSystem into server       |
| `cmd/server/types.go`            | Server struct with ccSignals field       |
| `internal/websocket/handlers.go` | Paste detection in code_update handler  |
| `api/rest/strudels/handlers.go`  | Index updates on create/update/delete   |
| `api/rest/strudels/routes.go`    | FingerprintIndexer interface            |
| `api/rest/agent/handlers.go`     | Lock check before AI generation         |
| `algopatterns/strudels/strudels.go`  | ListNoAIStrudels for startup load       |
| `algopatterns/strudels/queries.go`   | DB queries for validation               |

### Frontend

| File                                   | Purpose                                      |
| -------------------------------------- | -------------------------------------------- |
| `lib/websocket/types.ts`               | `source` field in `CodeUpdatePayload`         |
| `lib/websocket/client.ts`              | `sendCodeUpdate` with source parameter       |
| `lib/stores/editor.ts`                 | `nextUpdateSource` state for paste tracking  |
| `lib/api/agent/types.ts`               | `session_id` in `GenerateRequest`            |
| `lib/hooks/use-agent.ts`               | Pass `session_id`, handle paste_locked error |
| `components/shared/strudel-editor.tsx` | Paste event listener                         |

## API

### WebSocket: code_update

```json
{
  "type": "code_update",
  "payload": {
    "code": "...",
    "cursor_line": 10,
    "cursor_col": 5,
    "source": "typed"
  }
}
```

### WebSocket: paste_lock_changed

```json
{
  "type": "paste_lock_changed",
  "payload": {
    "locked": true,
    "reason": "paste_detected" | "similar_to_protected" | "parent_no_ai" | "edits_sufficient" | "session_reconnect"
  }
}
```

### REST: POST /api/v1/agent/generate

Request:

```json
{
  "user_query": "...",
  "editor_state": "...",
  "session_id": "abc123"
}
```

Error response when locked:

```json
{
  "error": "paste_locked",
  "message": "AI assistant temporarily disabled - please make significant edits to the pasted code before using AI."
}
```

## Testing Checklist

### Basic Detection

- [ ] Paste 200+ chars from external source → lock is set
- [ ] Paste 10+ lines from external source → lock is set
- [ ] Normal typing → no lock
- [ ] Edit ~20% of pasted code → lock removed

### Ownership Validation

- [ ] Load own saved strudel → no lock
- [ ] Paste code matching own strudel → no lock

### Public Content Validation

- [ ] Fork public strudel (allows AI) → no lock
- [ ] Fork public strudel (no-ai) → lock

### Fingerprint Detection

- [ ] Paste exact copy of no-ai strudel → lock (sticky)
- [ ] Paste slightly modified no-ai content (~90% similar) → lock (sticky)
- [ ] Paste heavily modified content (~50% similar) → no fingerprint match

### AI Request Blocking

- [ ] AI request while locked → 403 error
- [ ] AI request after unlock → succeeds

### Edge Cases

- [ ] Session disconnect → lock auto-expires (TTL)
- [ ] Redis unavailable → fail open, log error
- [ ] Anonymous user pastes code → lock applies
- [ ] Reconnect to locked session → lock status sent on connect

### Runtime Index Updates

- [ ] Create no-ai strudel → added to fingerprint index
- [ ] Update strudel to no-ai → added to fingerprint index
- [ ] Update strudel from no-ai to cc-cr → removed from index
- [ ] Delete no-ai strudel → removed from fingerprint index
