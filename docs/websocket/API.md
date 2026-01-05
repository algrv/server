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

| Role        | Permissions                              |
| ----------- | ---------------------------------------- |
| `host`      | Full access (edit code, chat, manage session, control playback) |
| `co-author` | Edit code, chat, control playback        |
| `viewer`    | Read-only, chat only                     |

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

Sent immediately after connection is established.

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

| Field          | Type   | Description                      |
| -------------- | ------ | -------------------------------- |
| `code`         | string | Current editor content           |
| `your_role`    | string | Your role in the session         |
| `participants` | array  | Currently connected participants |
| `chat_history` | array  | Chat message history             |

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
    "message": "too many code updates. maximum 10 per second.",
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

## Access Control Summary

| Feature              | Host | Co-author | Viewer |
| -------------------- | ---- | --------- | ------ |
| See code             | Y    | Y         | Y      |
| Edit code            | Y    | Y         | N      |
| Send chat messages   | Y    | Y         | Y      |
| See chat messages    | Y    | Y         | Y      |
| Control playback     | Y    | Y         | N      |
| End session          | Y    | N         | N      |

---

## AI Assistant

AI code generation is handled via the REST API, not WebSocket. This keeps AI conversation history personal to each user.

- **Drafts:** AI conversation stored in localStorage (frontend)
- **Saved strudels:** AI conversation stored in `strudel_messages` table (per-user)

When AI generates code, the frontend updates the editor locally and sends a `code_update` via WebSocket to sync with collaborators.

See the REST API documentation for AI endpoints.

---

## Limits

| Limit                | Value      |
| -------------------- | ---------- |
| Max message size     | 512 KB     |
| Max code size        | 100 KB     |
| Max chat message     | 5000 chars |
| Max display name     | 100 chars  |
| Code updates         | 10/second  |
| Chat messages        | 20/minute  |
| Connections per user | 5          |
| Connections per IP   | 10         |
| Ping timeout         | 60 seconds |

---

## Auto-Save Behavior

The server automatically saves code to the session - **no explicit save message needed from clients**.

**Server auto-saves on:**

| Event | What Happens |
|-------|--------------|
| First `code_update` after connect | Saves immediately to ensure session has something |
| `play` received | Saves before broadcasting (user is about to hear their code) |
| Client disconnect | Saves `LastCode` (best effort fallback) |

**Frontend just sends `code_update`** - server handles persistence automatically.

**Strudel save:** Separate from session auto-save. When user explicitly saves a strudel via REST API, code is persisted to the strudel table.
