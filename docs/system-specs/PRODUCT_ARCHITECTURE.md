# Product Architecture

This document describes the user-facing product architecture for Algorave, including authentication, user strudels, and collaborative sessions.

## Overview

Algorave is evolving from a simple code generation tool into a collaborative live coding platform where users can:
- Authenticate using OAuth providers
- Save and organize their strudel code
- Collaborate in real-time sessions
- Participate in scheduled events

## System Components

### 1. CLI/TUI Interface

**Location**: `cmd/tui/`, `internal/tui/`

**Status**: Implemented

Interactive terminal interface for local development:
- Welcome screen with command menu
- Live code editor with AI assistance
- Production-safe command filtering

### 2. Authentication System

**Location**: `internal/auth/`, `api/rest/auth/`, `algorave/users/`

Multi-provider OAuth authentication supporting:
- Google OAuth
- GitHub OAuth
- Apple Sign In

#### Flow
1. User clicks "Sign in with [Provider]"
2. Redirected to provider's OAuth page
3. Provider redirects to callback with authorization code
4. Backend exchanges code for user info
5. User created/found in database
6. JWT token issued (7-day expiration)
7. Token used for subsequent API requests

#### Database: `users` table

Key columns: `id`, `email`, `provider`, `provider_id`, `name`, `avatar_url`

#### API Endpoints

- `GET /api/v1/auth/:provider` - Start OAuth flow
- `GET /api/v1/auth/:provider/callback` - OAuth callback
- `POST /api/v1/auth/logout` - Clear session
- `GET /api/v1/auth/me` - Get current user

### 3. User Strudels

**Location**: `algorave/strudels/`, `api/rest/strudels/`

Users can save, organize, and share their Strudel code.

#### Database: `user_strudels` table

Key columns: `id`, `user_id`, `title`, `code`, `tags[]`, `categories[]`, `is_public`, `license`, `cc_signal`

#### Features

- Tags/categories for organization
- Public/private visibility
- Full CRUD operations
- CC Signals for AI usage preferences
- Creative Commons licenses for sharing rights

#### CC Signals (AI Preferences)

Users can set AI-specific preferences when saving strudels:

| Signal   | Name              | Meaning                                          |
| -------- | ----------------- | ------------------------------------------------ |
| `cc-cr`  | Credit            | Allow AI use with attribution                    |
| `cc-dc`  | Credit + Direct   | Attribution + financial/in-kind support          |
| `cc-ec`  | Credit + Ecosystem| Attribution + contribute to commons              |
| `cc-op`  | Credit + Open     | Attribution + keep derivatives open              |
| `no-ai`  | No AI             | Explicitly opt-out of AI assistance              |

The `no-ai` signal is enforced server-side via paste lock detection. See [Enforcing CC Signals](./ENFORCING-CC-SIGNALS.md).

#### Creative Commons Licenses

Standard CC licenses for sharing rights:

- `CC0 1.0` - Public Domain
- `CC BY 4.0` - Attribution
- `CC BY-SA 4.0` - Attribution-ShareAlike
- `CC BY-NC 4.0` - Attribution-NonCommercial
- `CC BY-NC-SA 4.0` - Attribution-NonCommercial-ShareAlike
- `CC BY-ND 4.0` - Attribution-NoDerivatives
- `CC BY-NC-ND 4.0` - Attribution-NonCommercial-NoDerivatives

#### API Endpoints

Protected (require authentication):
- `GET /api/v1/strudels` - List user's strudels
- `POST /api/v1/strudels` - Create new strudel
- `GET /api/v1/strudels/:id` - Get single strudel
- `PUT /api/v1/strudels/:id` - Update strudel
- `DELETE /api/v1/strudels/:id` - Delete strudel

Public:
- `GET /api/v1/public/strudels?limit=50` - List public strudels

### 4. Collaborative Sessions

**Location**: `algorave/sessions/`, `internal/websocket/`, `api/websocket/`

**Status**: Implemented

Real-time collaborative coding via WebSocket.

#### Roles

- **Host**: Session creator, full control
- **Co-author**: Can edit code
- **Viewer**: Read-only access

#### Architecture

WebSocket Hub (in-memory, with Redis pub/sub planned for scaling)

#### Database Tables

- `collaborative_sessions`: `id`, `host_user_id`, `title`, `code`, `is_active`
- Session participants and invite tokens supported

#### WebSocket Message Types

- `code_update` - Code synchronization between participants
- `chat_message` - Text chat between participants
- `play` / `stop` - Playback control (sync across participants)
- `user_joined` / `user_left` - Presence notifications
- `paste_lock_changed` - CC Signal enforcement (paste lock status)
- `session_state` - Initial session state on connect
- `session_ended` - Session terminated by host
- `ping` / `pong` - Connection health checks
- `error` - Error notifications
- `server_shutdown` - Graceful shutdown signal

See [WebSocket API Documentation](../websocket/API.md) for full details.

### 5. Events & Scheduled Sessions

**Status**: Planned

Scheduled live coding events with waiting rooms. `events` table: `id`, `host_user_id`, `title`, `scheduled_start`, `status`

## Code Structure

- `api/rest/` - REST handlers by domain (auth, strudels, generate, health)
- `api/websocket/` - WebSocket handlers for real-time collaboration
- `algorave/` - Business logic (users, strudels, sessions)
- `internal/` - Infrastructure (auth, agent, retriever, storage, llm, websocket)

## Security

- OAuth with CSRF protection
- JWT tokens (7-day expiration)
- Row-level security for user data
- WebSocket: JWT auth, rate limiting, connection limits

## Scalability

**Current**: In-memory WebSocket hub, single server
**Future**: Redis pub/sub, horizontal scaling, read replicas
