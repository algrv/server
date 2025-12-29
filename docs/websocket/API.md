# WebSocket API

Real-time collaboration API for Algorave sessions.

## Connection

```
ws://host/api/v1/ws
wss://host/api/v1/ws  (production)
```

### Query Parameters

| Parameter      | Type   | Required | Description                                                  |
| -------------- | ------ | -------- | ------------------------------------------------------------ |
| `session_id`   | UUID   | No       | Session to join. If omitted, creates a new anonymous session |
| `token`        | string | No       | JWT authentication token                                     |
| `invite`       | string | No       | Invite token for joining a session                           |
| `display_name` | string | No       | Display name (max 100 chars). Defaults to "Anonymous"        |

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

| Role        | Permissions                                              |
| ----------- | -------------------------------------------------------- |
| `host`      | Full access (edit code, use agent, chat, manage session) |
| `co-author` | Edit code, use agent, chat                               |
| `viewer`    | Read-only, chat only                                     |

---

## Message Format

All messages use this envelope:

```json
{
  "type": "message_type",
  "session_id": "uuid",
  "user_id": "uuid or empty",
  "timestamp": "2024-01-01T00:00:00Z",
  "payload": { ... }
}
```

---

## Client Messages (Send)

### `code_update`

Update the shared code editor. Requires `host` or `co-author` role.

```json
{
  "type": "code_update",
  "session_id": "uuid",
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

### `agent_request`

Request AI code generation. Requires `host` or `co-author` role.

```json
{
  "type": "agent_request",
  "session_id": "uuid",
  "provider": "anthropic",
  "provider_api_key": "sk-..."
  "payload": {
    "user_query": "add a kick drum on every beat",
    "editor_state": "sound(\"hh*8\")",
    "conversation_history": [
      {"role": "user", "content": "make a beat"},
      {"role": "assistant", "content": "sound(\"hh*8\")"}
    ],
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

Send a chat message to the session.

```json
{
  "type": "chat_message",
  "session_id": "uuid",
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

### `ping`

Keep connection alive. Server responds with `pong`.

```json
{
  "type": "ping",
  "session_id": "uuid",
  "payload": {}
}
```

---

## Server Messages (Receive)

### `code_update` (broadcast)

Sent when another user updates the code.

```json
{
  "type": "code_update",
  "session_id": "uuid",
  "user_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
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

```json
{
  "type": "agent_request",
  "session_id": "uuid",
  "user_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "payload": {
    "user_query": "add a kick drum on every beat",
    "display_name": "DJ Cool"
  }
}
```

---

### `agent_response` (broadcast)

Sent when AI completes code generation.

```json
{
  "type": "agent_response",
  "session_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "payload": {
    "code": "sound(\"bd\").fast(4)\n.stack(sound(\"hh*8\"))",
    "docs_retrieved": 3,
    "examples_retrieved": 2,
    "model": "claude-sonnet-4-20250514",
    "is_actionable": true
  }
}
```

**Or if query was vague:**

```json
{
  "type": "agent_response",
  "session_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "payload": {
    "is_actionable": false,
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

---

### `chat_message` (broadcast)

Sent when a user sends a chat message.

```json
{
  "type": "chat_message",
  "session_id": "uuid",
  "user_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
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
  "payload": {
    "user_id": "uuid",
    "display_name": "DJ Cool"
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
| Pro       | 1000         |
| BYOK      | Unlimited    |
