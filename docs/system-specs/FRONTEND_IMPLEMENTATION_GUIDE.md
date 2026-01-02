# Frontend Implementation Guide

This document describes how the frontend should interact with the Algorave API, covering authentication, sessions, and the distinction between REST and WebSocket usage.

## Authentication Model

### JWT Tokens

- Obtained via OAuth flow (`GET /api/v1/auth/:provider`)
- 7-day expiration
- Include in REST requests: `Authorization: Bearer {token}`
- Include in WebSocket connections: `?token={jwt}`

### User Types

| Type          | Has JWT | Can Create Sessions | Can Invite Others | Can Transfer Session |
| ------------- | ------- | ------------------- | ----------------- | -------------------- |
| Anonymous     | No      | Yes (solo only)     | No                | Yes (after login)    |
| Authenticated | Yes     | Yes                 | Yes               | N/A                  |

## API Usage Pattern

### REST API

Use REST for **authenticated CRUD operations only**.

| Endpoint                                 | Auth     | Purpose                               |
| ---------------------------------------- | -------- | ------------------------------------- |
| `GET /api/v1/auth/me`                    | Required | Get current user                      |
| `PUT /api/v1/auth/me`                    | Required | Update profile                        |
| `GET/POST/PUT/DELETE /api/v1/strudels/*` | Required | Strudel management                    |
| `GET/POST/PUT/DELETE /api/v1/sessions/*` | Required | Session management                    |
| `POST /api/v1/sessions/transfer`         | Required | Transfer anonymous session to strudel |
| `PUT /api/v1/sessions/{id}/discoverable` | Required | Toggle session discoverability (host) |
| `GET /api/v1/sessions/live`              | Public   | List discoverable live sessions       |
| `GET /api/v1/public/strudels`            | Public   | Browse public strudels                |
| `POST /api/v1/sessions/join`             | Optional | Join session with invite token        |

### WebSocket

Use WebSocket for **real-time session state and collaboration**.

```
wss://algorave.ai/ws?session_id={uuid}&token={jwt}&display_name={name}
```

| Parameter      | Required | Description                  |
| -------------- | -------- | ---------------------------- |
| `session_id`   | No       | Omit to create new session   |
| `token`        | No       | JWT for authenticated users  |
| `invite_token` | No       | For joining via invite link  |
| `display_name` | No       | Display name (max 100 chars) |

## Session Flows

### Anonymous User Flow

```
1. Connect WebSocket (no token, no session_id)
   → Server creates anonymous session
   → Returns session_id in welcome message

2. Store session_id in localStorage/URL

3. User codes solo via WebSocket
   - Send: code_update, agent_request, chat
   - Receive: session_state, code_update, agent_response

4. On page refresh:
   → Reconnect WebSocket with stored session_id
   → Receive session_state with current code

5. User decides to save work:
   → Complete OAuth login
   → Call POST /api/v1/sessions/transfer with session_id
   → Session becomes a strudel in their account
```

### Authenticated User Flow

```
1. User logs in via OAuth
   → Store JWT token

2. Create session:
   Option A: POST /api/v1/sessions (REST)
   Option B: Connect WebSocket with token (no session_id)

3. Invite collaborators:
   → POST /api/v1/sessions/{id}/invite
   → Share invite link with token

4. Collaborate via WebSocket
   → All participants connect with session_id + (JWT or invite_token)

5. Manage session via REST:
   → GET /api/v1/sessions/{id} - view details
   → PUT /api/v1/sessions/{id} - update code
   → DELETE /api/v1/sessions/{id} - end session
```

### Joining via Invite Link

```
URL format: https://algorave.ai/join?invite={invite_token}

1. Parse invite_token from URL

2. Connect WebSocket:
   ?session_id={from_token}&invite_token={token}&display_name={user_input}

3. If user is logged in, also include JWT token
```

### Go Live Flow (Discoverable Sessions)

Hosts can make their sessions publicly discoverable so anyone can join from a "Live Sessions" page.

```
1. Host creates or opens a session

2. Host clicks "Go Live" button:
   → PUT /api/v1/sessions/{id}/discoverable
   → Body: { "is_discoverable": true }

3. Session appears in public listing:
   → GET /api/v1/sessions/live returns:
   {
     "sessions": [
       {
         "id": "uuid",
         "title": "Session Title",
         "participant_count": 5,
         "created_at": "...",
         "last_activity": "..."
       }
     ]
   }

4. Viewer clicks on a live session:
   → Host must create invite token for viewers
   → Or implement auto-join for discoverable sessions

5. Host clicks "Stop Live" to hide session:
   → PUT /api/v1/sessions/{id}/discoverable
   → Body: { "is_discoverable": false }
```

**Note:** Discoverable sessions still require an invite token to join. The live listing shows available sessions, but joining requires the host to have created an invite token (typically with unlimited uses for public sessions).

## WebSocket Messages

### Client Sends

| Type            | Payload                                        | Permission      |
| --------------- | ---------------------------------------------- | --------------- |
| `code_update`   | `{ code: string }`                             | host, co-author |
| `agent_request` | `{ user_query: string, editor_state: string }` | all             |
| `chat`          | `{ message: string }`                          | all             |
| `ping`          | `{}`                                           | all             |

### Server Sends

| Type                 | Description                   |
| -------------------- | ----------------------------- |
| `session_state`      | Full session state on connect |
| `code_update`        | Code changed by another user  |
| `agent_response`     | AI generation result          |
| `chat`               | Chat message from user        |
| `participant_joined` | New participant               |
| `participant_left`   | Participant disconnected      |
| `error`              | Error message                 |

## Rate Limits

| Action                         | Limit                 |
| ------------------------------ | --------------------- |
| Code updates                   | 10/second             |
| Chat messages                  | 20/minute             |
| AI generation (free)           | 10/minute + daily cap |
| WebSocket connections per IP   | 10                    |
| WebSocket connections per user | 5                     |

## Security Notes

1. **Anonymous sessions are solo-only** - users cannot invite others without authenticating
2. **Session IDs are secrets** - treat them like tokens for anonymous users
3. **REST session endpoints require auth** - anonymous users use WebSocket exclusively
4. **Invite tokens are single-use or limited** - respect max_uses and expiration

## Admin Endpoints

Admin endpoints require a JWT token from a user with `is_admin = true`.

| Endpoint                                         | Auth  | Purpose                                |
| ------------------------------------------------ | ----- | -------------------------------------- |
| `GET /api/v1/admin/strudels/{id}`                | Admin | Get any strudel (regardless of owner)  |
| `PUT /api/v1/admin/strudels/{id}/use-in-training`| Admin | Mark strudel for AI training data      |

### Admin Authentication

Admins use the same JWT authentication as regular users. The `is_admin` claim is embedded in the JWT when the user logs in.

```
Authorization: Bearer {jwt_with_admin_claim}
```

### Use in Training Flag

Admins can mark public strudels for inclusion in AI training data:

```
PUT /api/v1/admin/strudels/{id}/use-in-training
Body: { "use_in_training": true }
```

This is separate from user consent - both the user's `training_consent` AND the strudel's `use_in_training` must be true for the strudel to be used in training.
