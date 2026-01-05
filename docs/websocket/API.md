# WebSocket API

Real-time collaboration API for Algorave sessions.

## Connection

```
ws://host/api/v1/ws
wss://host/api/v1/ws  (production)
```

### Query Parameters

| Parameter             | Type   | Required | Description                                                  |
| --------------------- | ------ | -------- | ------------------------------------------------------------ |
| `session_id`          | UUID   | No       | Session to join. If omitted, creates a new anonymous session |
| `token`               | string | No       | JWT authentication token                                     |
| `invite`              | string | No       | Invite token for joining a session                           |
| `display_name`        | string | No       | Display name (max 100 chars). Defaults to "Anonymous"        |
| `previous_session_id` | UUID   | No       | Copy code from this session when creating a new one          |

### Connection Scenarios

**1. Create new anonymous session:**

```
ws://host/api/v1/ws
ws://host/api/v1/ws?display_name=DJ%20Cool
```

**2. Create new session as authenticated user:**

```
ws://host/api/v1/ws?token=<jwt>
```

**3. Join existing session with JWT:**

```
ws://host/api/v1/ws?session_id=<uuid>&token=<jwt>
```

**4. Join existing session with invite:**

```
ws://host/api/v1/ws?session_id=<uuid>&invite=<token>&display_name=Guest
```

### Roles

| Role        | Permissions                                                       |
| ----------- | ----------------------------------------------------------------- |
| `host`      | Full access (edit code, use agent, chat, manage session, switch strudel) |
| `co-author` | Edit code, use agent, chat, switch strudel                        |
| `viewer`    | Read-only, chat only (no AI conversation history)                 |

---

## Message Format

All messages use this envelope:

```json
{
  "type": "message_type",
  "session_id": "uuid",
  "user_id": "uuid or empty",
  "timestamp": "2024-01-01T00:00:00Z",
  "seq": 123,
  "payload": { ... }
}
```

The `seq` field is a sequence number for message ordering within a session.

---

## Client Messages (Send)

### `code_update`

Update the shared code editor. Requires `host` or `co-author` role.

**Note:** This broadcasts the update to other clients but does NOT persist to database. Use `auto_save` for persistence.

```json
{
  "type": "code_update",
  "payload": {
    "code": "sound(\"bd sd\").fast(2)",
    "cursor_line": 1,
    "cursor_col": 15
  }
}
```

| Field         | Type   | Required | Description                     |
| ------------- | ------ | -------- | ------------------------------- |
| `code`        | string | Yes      | Full editor content (max 100KB) |
| `cursor_line` | int    | No       | Cursor line position            |
| `cursor_col`  | int    | No       | Cursor column position          |

**Rate limit:** 10 updates/second

---

### `switch_strudel`

Switch strudel context without reconnecting. Requires `host` or `co-author` role.

**1. Switch to a saved strudel (authenticated users only):**

```json
{
  "type": "switch_strudel",
  "payload": {
    "strudel_id": "uuid"
  }
}
```

Backend fetches the strudel from database (verifies ownership) and returns code + conversation history.

**2. Switch to fresh scratch context:**

```json
{
  "type": "switch_strudel",
  "payload": {
    "strudel_id": null
  }
}
```

Returns empty code and empty conversation history.

**3. Restore from localStorage (after accidental tab close):**

```json
{
  "type": "switch_strudel",
  "payload": {
    "strudel_id": null,
    "code": "sound(\"bd sd\")",
    "conversation_history": [
      {
        "id": "msg-1",
        "role": "user",
        "content": "make a beat",
        "is_code_response": false,
        "display_name": "User",
        "timestamp": 1704067200000
      },
      {
        "id": "msg-2",
        "role": "assistant",
        "content": "sound(\"bd sd\")",
        "is_code_response": true,
        "display_name": "Assistant",
        "timestamp": 1704067201000
      }
    ]
  }
}
```

| Field                  | Type   | Required | Description                                     |
| ---------------------- | ------ | -------- | ----------------------------------------------- |
| `strudel_id`           | string | No       | Strudel UUID, or null for scratch               |
| `code`                 | string | No       | Code to restore (only for scratch/localStorage) |
| `conversation_history` | array  | No       | Conversation to restore (only for localStorage) |

**Response:** Server sends `session_state` back to the requesting client only.

---

### `agent_request`

Request AI code generation. Requires `host` or `co-author` role.

**Important:** Only broadcast to other hosts/co-authors, NOT to viewers.

```json
{
  "type": "agent_request",
  "payload": {
    "user_query": "add a kick drum on every beat",
    "editor_state": "sound(\"hh*8\")",
    "conversation_history": [
      { "role": "user", "content": "make a beat" },
      { "role": "assistant", "content": "sound(\"hh*8\")" }
    ],
    "provider": "anthropic",
    "provider_api_key": "sk-..."
  }
}
```

| Field                  | Type   | Required | Description                   |
| ---------------------- | ------ | -------- | ----------------------------- |
| `user_query`           | string | Yes      | Natural language request      |
| `editor_state`         | string | No       | Current code in editor        |
| `conversation_history` | array  | No       | Previous conversation turns   |
| `provider`             | string | No       | BYOK: `anthropic` or `openai` |
| `provider_api_key`     | string | No       | BYOK: Your API key            |

**Rate limit:** 10 requests/minute (per-minute) + daily limits based on tier

**Note:** `editor_state`, `conversation_history`, `provider`, and `provider_api_key` are NOT broadcast to other clients.

---

### `chat_message`

Send a chat message to the session. All roles can send chat messages.

```json
{
  "type": "chat_message",
  "payload": {
    "message": "Hey, try adding some reverb!"
  }
}
```

| Field     | Type   | Required | Description                   |
| --------- | ------ | -------- | ----------------------------- |
| `message` | string | Yes      | Chat message (max 5000 chars) |

**Rate limit:** 20 messages/minute

---

### `play`

Start playback for all session participants. Requires `host` or `co-author` role.

```json
{
  "type": "play",
  "payload": {}
}
```

---

### `stop`

Stop playback for all session participants. Requires `host` or `co-author` role.

```json
{
  "type": "stop",
  "payload": {}
}
```

---

### `ping`

Keep connection alive. Server responds with `pong`.

```json
{
  "type": "ping",
  "payload": {}
}
```

---

## Server Messages (Receive)

### `session_state`

Sent in two scenarios:
1. Immediately after connection is established
2. In response to a `switch_strudel` message

```json
{
  "type": "session_state",
  "session_id": "uuid",
  "user_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "payload": {
    "code": "sound(\"bd sd\").fast(2)",
    "your_role": "co-author",
    "participants": [
      { "user_id": "uuid", "display_name": "Host", "role": "host" },
      { "user_id": "", "display_name": "Guest", "role": "viewer" }
    ],
    "conversation_history": [
      {
        "id": "msg-123",
        "role": "user",
        "content": "make a beat",
        "is_code_response": false,
        "display_name": "User",
        "timestamp": 1704067200000
      },
      {
        "id": "msg-124",
        "role": "assistant",
        "content": "sound(\"bd sd\")",
        "is_code_response": true,
        "display_name": "Assistant",
        "timestamp": 1704067201000
      }
    ],
    "chat_history": [
      {
        "display_name": "Host",
        "avatar_url": "https://...",
        "content": "Welcome to the session!",
        "timestamp": 1704067200000
      }
    ]
  }
}
```

| Field                  | Type   | Description                                            |
| ---------------------- | ------ | ------------------------------------------------------ |
| `code`                 | string | Current editor content                                 |
| `your_role`            | string | Your role in the session                               |
| `participants`         | array  | Currently connected participants                       |
| `conversation_history` | array  | LLM conversation history (user prompts + AI responses) |
| `chat_history`         | array  | Chat message history with timestamps                   |

**Note:**
- `conversation_history` is **empty for viewers** - they don't see AI conversation.
- When sent in response to `switch_strudel`, `participants` may be empty (not re-sent).
- `chat_history` is session-scoped (shared with all participants including viewers).

---

### `code_update` (broadcast)

Sent when another user updates the code.

```json
{
  "type": "code_update",
  "session_id": "uuid",
  "user_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "seq": 42,
  "payload": {
    "code": "sound(\"bd sd\").fast(2)",
    "cursor_line": 1,
    "cursor_col": 15,
    "display_name": "DJ Cool"
  }
}
```

---

### `agent_request` (broadcast)

Sent when a user submits an AI request (sanitized - no private data).

**Important:** Only sent to hosts and co-authors, NOT to viewers.

```json
{
  "type": "agent_request",
  "session_id": "uuid",
  "user_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "seq": 43,
  "payload": {
    "user_query": "add a kick drum on every beat",
    "display_name": "DJ Cool"
  }
}
```

---

### `agent_response`

Sent when AI completes code generation or answers a question.

**Important:** Only sent to hosts and co-authors, NOT to viewers.

**Code generation response (should update editor):**

```json
{
  "type": "agent_response",
  "session_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "seq": 44,
  "payload": {
    "code": "sound(\"bd\").fast(4)\n.stack(sound(\"hh*8\"))",
    "docs_retrieved": 3,
    "examples_retrieved": 2,
    "model": "claude-sonnet-4-20250514",
    "is_actionable": true,
    "is_code_response": true,
    "rate_limit": {
      "requests_remaining": 8,
      "requests_limit": 10,
      "reset_seconds": 45
    }
  }
}
```

**Question/explanation response (should NOT update editor):**

```json
{
  "type": "agent_response",
  "session_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "payload": {
    "code": "The lpf (low-pass filter) removes high frequencies...",
    "docs_retrieved": 2,
    "examples_retrieved": 1,
    "model": "claude-sonnet-4-20250514",
    "is_actionable": true,
    "is_code_response": false
  }
}
```

**Vague query (needs clarification):**

```json
{
  "type": "agent_response",
  "session_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "payload": {
    "is_actionable": false,
    "is_code_response": false,
    "clarifying_questions": [
      "What BPM would you like?",
      "Which drum sounds should I use?"
    ],
    "docs_retrieved": 0,
    "examples_retrieved": 0,
    "model": "claude-sonnet-4-20250514"
  }
}
```

| Field                  | Type   | Description                                        |
| ---------------------- | ------ | -------------------------------------------------- |
| `code`                 | string | Generated code or explanation                      |
| `is_actionable`        | bool   | Whether the response is actionable                 |
| `is_code_response`     | bool   | If true, frontend should update the editor         |
| `clarifying_questions` | array  | Questions to ask user (when is_actionable = false) |
| `rate_limit`           | object | Current rate limit status                          |

---

### `chat_message` (broadcast)

Sent when a user sends a chat message. Broadcast to ALL participants including viewers.

```json
{
  "type": "chat_message",
  "session_id": "uuid",
  "user_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "seq": 45,
  "payload": {
    "message": "Hey, try adding some reverb!",
    "display_name": "DJ Cool"
  }
}
```

---

### `user_joined` (broadcast)

Sent when a new user joins the session.

```json
{
  "type": "user_joined",
  "session_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "seq": 46,
  "payload": {
    "user_id": "uuid",
    "display_name": "New User",
    "role": "co-author"
  }
}
```

---

### `user_left` (broadcast)

Sent when a user leaves the session.

```json
{
  "type": "user_left",
  "session_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "seq": 47,
  "payload": {
    "user_id": "uuid",
    "display_name": "DJ Cool"
  }
}
```

---

### `play` (broadcast)

Sent when host or co-author starts playback. Frontend should programmatically trigger Strudel play.

```json
{
  "type": "play",
  "session_id": "uuid",
  "user_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "seq": 48,
  "payload": {
    "display_name": "DJ Cool"
  }
}
```

---

### `stop` (broadcast)

Sent when host or co-author stops playback. Frontend should programmatically trigger Strudel stop.

```json
{
  "type": "stop",
  "session_id": "uuid",
  "user_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "seq": 49,
  "payload": {
    "display_name": "DJ Cool"
  }
}
```

---

### `session_ended` (broadcast)

Sent when the host ends the session. Connection will be closed shortly after.

```json
{
  "type": "session_ended",
  "session_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "payload": {
    "reason": "session ended by host"
  }
}
```

Frontend should handle this by:

- Showing a "session ended" notification
- Switching to offline/replay mode where viewer can control their own playback
- Optionally offering to fork the final code

---

### `server_shutdown` (broadcast)

Sent when the server is shutting down for maintenance.

```json
{
  "type": "server_shutdown",
  "session_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "payload": {
    "reason": "server is shutting down for maintenance"
  }
}
```

---

### `error`

Sent when an error occurs processing a message.

```json
{
  "type": "error",
  "session_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "payload": {
    "error": "too_many_requests",
    "message": "too many agent requests. maximum 10 per minute.",
    "details": ""
  }
}
```

| Error Code          | Description                                            |
| ------------------- | ------------------------------------------------------ |
| `too_many_requests` | Rate limit exceeded                                    |
| `forbidden`         | Insufficient permissions (e.g., viewer trying to edit) |
| `unauthorized`      | Authentication required (e.g., loading saved strudel)  |
| `not_found`         | Resource not found (e.g., strudel doesn't exist)       |
| `validation_error`  | Invalid message format                                 |
| `bad_request`       | Invalid request (e.g., code too large)                 |
| `server_error`      | Internal server error                                  |

---

### `pong`

Response to client `ping`.

```json
{
  "type": "pong",
  "session_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "payload": {}
}
```

---

## Strudel Context Management

### Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                        DATA SOURCES                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Saved Strudels (DB)          Session Messages (DB)              │
│  ┌──────────────────┐         ┌──────────────────┐              │
│  │ strudels table   │         │ session_messages │              │
│  │ - code           │         │ - chat messages  │              │
│  │ - conversation   │         │ - agent log      │              │
│  │   history        │         │ (audit trail)    │              │
│  └──────────────────┘         └──────────────────┘              │
│           │                            │                         │
│           │ switch_strudel             │ on connect              │
│           │ (strudel_id)               │ (session history)       │
│           ▼                            ▼                         │
│  ┌──────────────────────────────────────────────────┐           │
│  │              session_state response               │           │
│  │  - code (from strudel or provided)               │           │
│  │  - conversation_history (filtered for viewers)   │           │
│  │  - chat_history (session-scoped)                 │           │
│  └──────────────────────────────────────────────────┘           │
│                                                                  │
│  localStorage (Frontend)                                         │
│  ┌──────────────────┐                                           │
│  │ Unsaved sessions │                                           │
│  │ - code           │──── switch_strudel ────▶ Restore          │
│  │ - conversation   │     (null + data)                         │
│  └──────────────────┘                                           │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Switching Strudels

| Scenario                 | Payload                                         | Who Can Do It      |
| ------------------------ | ----------------------------------------------- | ------------------ |
| Open saved strudel       | `{strudel_id: "uuid"}`                          | Auth users only    |
| Start fresh scratch      | `{strudel_id: null}`                            | Anyone (host/co-author) |
| Restore from localStorage| `{strudel_id: null, code: "...", conversation_history: [...]}` | Anyone (host/co-author) |

### Access Control Summary

| Feature                    | Host | Co-author | Viewer |
| -------------------------- | ---- | --------- | ------ |
| See code                   | ✓    | ✓         | ✓      |
| Edit code                  | ✓    | ✓         | ✗      |
| Use AI assistant           | ✓    | ✓         | ✗      |
| See AI conversation        | ✓    | ✓         | ✗      |
| Send chat messages         | ✓    | ✓         | ✓      |
| See chat messages          | ✓    | ✓         | ✓      |
| Control playback           | ✓    | ✓         | ✗      |
| Switch strudel context     | ✓    | ✓         | ✗      |
| End session                | ✓    | ✗         | ✗      |

---

## Limits

| Limit                | Value      |
| -------------------- | ---------- |
| Max message size     | 512 KB     |
| Max code size        | 100 KB     |
| Max chat message     | 5000 chars |
| Max display name     | 100 chars  |
| Code updates         | 10/second  |
| Agent requests       | 10/minute  |
| Chat messages        | 20/minute  |
| Connections per user | 5          |
| Connections per IP   | 10         |
| Ping timeout         | 60 seconds |

---

## Daily Rate Limits (Agent Requests)

| Tier      | Requests/Day |
| --------- | ------------ |
| Anonymous | 50           |
| Free      | 100          |
| PAYG      | 1000         |
| BYOK      | Unlimited    |

---

## Auto-Save Behavior

The server automatically saves code to the session - **no explicit save message needed from clients**.

**Server auto-saves on:**

| Event | What Happens |
|-------|--------------|
| First `code_update` after connect/switch | Saves immediately to ensure session has something |
| `play` received | Saves before broadcasting (user is about to hear their code) |
| `switch_strudel` received | Saves current code before switching to new context |
| `agent_response` with code | Saves the generated code |
| Client disconnect | Saves `LastCode` (best effort fallback) |

**Frontend just sends `code_update`** - server handles persistence automatically.

**Strudel save:** Separate from auto_save. When user explicitly saves a strudel via REST API, code + conversation history are persisted to the strudel table.
