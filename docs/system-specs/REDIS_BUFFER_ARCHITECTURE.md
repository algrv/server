# Redis Buffer Architecture

## Problem

Direct Postgres writes on every WebSocket event (code updates, chat, AI messages) don't scale:

- 2000 concurrent sessions × 10 edits/sec = 20,000 writes/sec
- Per-client state tracking was broken for multi-client sync
- Data loss risk on ungraceful disconnects

## Solution

Buffer writes to Redis, flush to Postgres periodically.

```
┌─────────────────────────────────────────────────────────┐
│                      Handlers                           │
│   (CodeUpdateHandler, ChatHandler, GenerateHandler)     │
│              ↓ sessionRepo.UpdateSessionCode()          │
└─────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────┐
│              BufferedRepository (wrapper)               │
│   - UpdateSessionCode() → Redis                         │
│   - AddMessage() → Redis                                │
│   - Other methods → pass through to Postgres            │
└─────────────────────────────────────────────────────────┘
          ↓ writes                    ↓ reads
    ┌──────────┐              ┌──────────────┐
    │  Redis   │              │   Postgres   │
    └──────────┘              └──────────────┘
          ↓ Flusher (every 5s)
    ┌──────────────┐
    │   Postgres   │
    └──────────────┘
```

## Key Design Decisions

### 1. Repository Wrapper Pattern

Handlers use `sessions.Repository` interface - they don't know about Redis. Buffering is transparent.

### 2. What's Buffered

- `UpdateSessionCode` - code edits
- `AddMessage` - chat messages, AI prompts/responses

### 3. What's NOT Buffered (pass-through)

- Session creation/deletion
- Participant management
- Invite tokens
- Read operations

### 4. Flush Triggers

- Every 5 seconds (background worker)
- On client disconnect
- On strudel context switch
- On graceful shutdown

## Files

| File                            | Purpose                           |
| ------------------------------- | --------------------------------- |
| `internal/buffer/buffer.go`     | Redis client, SetCode, AddMessage |
| `internal/buffer/repository.go` | BufferedRepository wrapper        |
| `internal/buffer/flusher.go`    | Background flush worker             |
| `internal/buffer/types.go`      | BufferedMessage, Redis keys       |

## Configuration

```
REDIS_URL=redis://localhost:6379           # Local
REDIS_URL=rediss://xxx@xxx.upstash.io:6379 # Upstash (TLS)
```

## Trade-offs

| Aspect      | Trade-off                                                       |
| ----------- | --------------------------------------------------------------- |
| Latency     | Writes are faster (Redis vs Postgres)                           |
| Durability  | Up to 5s of data at risk if Redis + server crash simultaneously |
| Complexity  | Extra infrastructure (Redis)                                    |
| Scalability | Supports horizontal scaling (multiple server instances)         |
