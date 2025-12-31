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

Key columns: `id`, `user_id`, `title`, `code`, `tags[]`, `categories[]`, `is_public`

#### Features

- Tags/categories for organization
- Public/private visibility
- Full CRUD operations

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

#### WebSocket Events

Client → Server: `join`, `code_update`, `promote`, `end_session`
Server → Client: `user_joined`, `code_update`, `user_promoted`, `session_ended`, `user_left`

### 5. Events & Scheduled Sessions (Phase 2)

**Status**: Planned for future

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

Phase 1: In-memory WebSocket hub, single server
Phase 2+: Redis pub/sub, horizontal scaling, read replicas
