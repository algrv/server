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

**Location**: `cmd/algorave/`, `internal/tui/`, `internal/ssh/`

**Status**: Planned (Phases 1-3)

Interactive terminal interface for local development and remote access:
- Welcome screen with command menu
- Live code editor with AI assistance
- Remote SSH access for collaborative coding
- Production-safe command filtering

See [CLI Architecture](./CLI_ARCHITECTURE.md) for detailed design.

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

#### Database Schema

**users table**
```sql
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL,
  provider TEXT NOT NULL,           -- "google", "github", "apple"
  provider_id TEXT NOT NULL,        -- OAuth provider's user ID
  name TEXT,
  avatar_url TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(provider, provider_id)
);
```

#### API Endpoints

- `GET /api/v1/auth/:provider` - Start OAuth flow
- `GET /api/v1/auth/:provider/callback` - OAuth callback
- `POST /api/v1/auth/logout` - Clear session
- `GET /api/v1/auth/me` - Get current user (protected)

### 3. User Strudels

**Location**: `algorave/strudels/`, `api/rest/strudels/`

Users can save, organize, and share their Strudel code.

#### Database Schema

**user_strudels table**
```sql
CREATE TABLE user_strudels (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  description TEXT,
  code TEXT NOT NULL,
  tags TEXT[],
  categories TEXT[],
  is_public BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_user_strudels_tags ON user_strudels USING GIN (tags);
CREATE INDEX idx_user_strudels_categories ON user_strudels USING GIN (categories);
```

#### Features

- **Organization**: Tags and categories for easy filtering
- **Visibility**: Public/private strudels
- **Discovery**: Browse public strudels from other users
- **Metadata**: Title, description, tags, categories
- **Full CRUD**: Create, read, update, delete operations

#### API Endpoints

Protected (require authentication):
- `GET /api/v1/strudels` - List user's strudels
- `POST /api/v1/strudels` - Create new strudel
- `GET /api/v1/strudels/:id` - Get single strudel
- `PUT /api/v1/strudels/:id` - Update strudel
- `DELETE /api/v1/strudels/:id` - Delete strudel

Public:
- `GET /api/v1/public/strudels?limit=50` - List public strudels

### 4. Collaborative Sessions (Phase 1)

**Status**: Planned, not yet implemented

Real-time collaborative coding sessions where multiple users can code together.

#### Roles

- **Host**: Session creator, full control
- **Co-author**: Can edit code, invited via link or manually promoted
- **Viewer**: Read-only access, can view code changes in real-time

#### Architecture

```
┌─────────────┐
│   Client    │
│ (WebSocket) │
└──────┬──────┘
       │
       ↓
┌─────────────────┐
│  WebSocket Hub  │ ← In-memory for Phase 1
│   (Go Server)   │   Redis pub/sub for Phase 2+
└─────────┬───────┘
          │
          ↓
┌──────────────────┐
│  PostgreSQL DB   │
│  - Sessions      │
│  - Participants  │
│  - Invite Tokens │
└──────────────────┘
```

#### Database Schema (Planned)

**sessions table**
```sql
CREATE TABLE sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  host_user_id UUID NOT NULL REFERENCES users(id),
  title TEXT NOT NULL,
  code TEXT NOT NULL,                    -- Current session code
  is_active BOOLEAN DEFAULT true,
  event_id UUID REFERENCES events(id),   -- NULL for ad-hoc sessions
  created_at TIMESTAMPTZ DEFAULT NOW(),
  ended_at TIMESTAMPTZ
);
```

**session_participants table**
```sql
CREATE TABLE session_participants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  user_id UUID REFERENCES users(id),     -- NULL if not logged in
  role TEXT NOT NULL,                    -- 'host', 'co-author', 'viewer'
  status TEXT NOT NULL,                  -- 'invited', 'waiting', 'active', 'left'
  joined_at TIMESTAMPTZ DEFAULT NOW(),
  left_at TIMESTAMPTZ
);
```

**invite_tokens table**
```sql
CREATE TABLE invite_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID REFERENCES sessions(id) ON DELETE CASCADE,
  event_id UUID REFERENCES events(id) ON DELETE CASCADE,
  token TEXT UNIQUE NOT NULL,            -- Shareable token
  role TEXT NOT NULL,                    -- 'co-author' or 'viewer'
  max_uses INTEGER DEFAULT 1,            -- NULL = unlimited
  uses_count INTEGER DEFAULT 0,
  expires_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
```

#### WebSocket Events

**Client → Server**
```typescript
// Join session
{ type: 'join', sessionId: string, token?: string }

// Update code
{ type: 'code_update', code: string, cursor?: { line, col } }

// Promote viewer to co-author
{ type: 'promote', participantId: string }

// End session
{ type: 'end_session' }
```

**Server → Client**
```typescript
// User joined
{ type: 'user_joined', user: User, role: string }

// Code updated
{ type: 'code_update', code: string, userId: string, cursor?: Position }

// User promoted
{ type: 'user_promoted', userId: string, role: string }

// Session ended
{ type: 'session_ended', finalCode: string }

// User left
{ type: 'user_left', userId: string }
```

#### Session Lifecycle

1. **Create**: Host creates session, gets session ID
2. **Invite**:
   - Generate single-use co-author invite link
   - Generate unlimited viewer link
   - Or manually promote viewers after they join
3. **Join**: Users join via invite link or session ID
4. **Collaborate**: Real-time code updates via WebSocket
5. **Save**: Users can save session code to their strudels
6. **End**: Host ends session, code snapshot saved

### 5. Events & Scheduled Sessions (Phase 2)

**Status**: Planned for future

Scheduled live coding events with waiting rooms and participant management.

#### Database Schema (Planned)

**events table**
```sql
CREATE TABLE events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  host_user_id UUID NOT NULL REFERENCES users(id),
  title TEXT NOT NULL,
  description TEXT,
  scheduled_start TIMESTAMPTZ NOT NULL,
  scheduled_end TIMESTAMPTZ,
  max_participants INTEGER,
  status TEXT NOT NULL,                  -- 'scheduled', 'live', 'ended'
  created_at TIMESTAMPTZ DEFAULT NOW()
);
```

#### Event Flow

1. **Schedule**: Host creates event with date/time
2. **Promote**: Share event link, users RSVP
3. **Waiting Room**: Before start, users gather in waiting room
4. **Go Live**: At scheduled time, session starts
5. **Participate**: Real-time collaboration
6. **End**: Event concludes, recording saved

#### Features

- Event calendar view
- RSVP system
- Waiting room with participant list
- Scheduled notifications
- Event recordings/playback
- Post-event strudel gallery

## API Architecture

### REST API

**Structure**: `api/rest/`

Domain-based organization:
- `auth/` - Authentication routes
- `strudels/` - User strudels CRUD
- `generate/` - Code generation (public)
- `health/` - Health checks

Each domain has:
- `handlers.go` - HTTP handlers
- `routes.go` - Route registration
- `types.go` - Request/Response types (if needed)

### Domain Layer

**Structure**: `algorave/`

Business logic for product domains:
- `users/` - User management
- `strudels/` - Strudel operations

Each domain has:
- `<domain>.go` - Repository methods
- `queries.go` - SQL query constants
- `types.go` - Domain types

### Infrastructure Layer

**Structure**: `internal/`

Shared infrastructure:
- `auth/` - Auth utilities (JWT, OAuth, middleware)
- `agent/` - Code generation agent
- `retriever/` - RAG system
- `storage/` - Database operations
- `llm/` - LLM client abstraction

## Security Considerations

### Authentication
- OAuth state parameter prevents CSRF
- JWT tokens with expiration
- Secure session cookies (HttpOnly, Secure in production)
- HTTPS required in production

### Authorization
- Row-level security: users can only access their own strudels
- Session participants verified via tokens or user_id
- Host-only actions: promote users, end session

### WebSocket
- JWT-based WebSocket authentication
- Rate limiting on code updates
- Message size limits
- Connection limits per user

## Scalability Design

### Current (Phase 1)
- In-memory WebSocket hub
- Single server instance
- PostgreSQL database

### Future (Phase 2+)
- Redis pub/sub for multi-server WebSocket
- Horizontal scaling behind load balancer
- Database read replicas
- CDN for static assets
- Separate WebSocket servers

## Related Documentation

- [CLI Architecture](./CLI_ARCHITECTURE.md) - Terminal interface design
- [RAG Architecture](./RAG_ARCHITECTURE.md) - Code generation system
- [Hybrid Retrieval Guide](./HYBRID_RETRIEVAL_GUIDE.md) - Retrieval implementation
- [AUTH_SETUP.md](../../AUTH_SETUP.md) - OAuth provider setup guide
