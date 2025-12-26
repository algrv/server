# CLI/TUI Architecture

This document describes the Terminal User Interface (TUI) for Algorave, providing an interactive command-line interface for local development and remote access.

## Overview

The Algorave CLI is a terminal-based interface that provides:
- Interactive command menu for server and ingester operations
- Live code editor with AI-powered assistance
- Real-time output and question/answer flow
- Remote access via SSH for collaborative coding
- Production-safe command filtering

## Architecture Components

### 1. Local TUI Application

**Location**: `cmd/algorave/`, `internal/tui/`

Built using the [Charm](https://charm.sh/) ecosystem:
- **Bubbletea**: Elm-architecture TUI framework
- **Lipgloss**: Terminal styling and layouts
- **Bubbles**: Pre-built UI components

#### Component Structure

```
internal/tui/
├── app.go          # Main Bubbletea application & state machine
├── welcome.go      # Welcome screen model
├── editor.go       # Code editor model
├── output.go       # Output/results model
├── styles.go       # Lipgloss style definitions
└── commands.go     # Command execution handlers
```

### 2. Remote Access Server

**Location**: `internal/ssh/`

SSH server for remote terminal access using **Wish** (Charm's SSH library):
- Wraps Bubbletea app in SSH session
- Per-user session management
- Authentication via SSH keys or tokens
- Production mode filtering

### 3. Integration Layer

**Location**: `internal/tui/agent.go`

Bridges TUI with existing agent system:
- Sends editor code to agent for analysis
- Streams AI responses to output view
- Handles clarification questions
- Manages conversation history

## User Interface

### Screen Layout

```
┌─────────────────────────────────────────────────────────────┐
│                     🎵 ALGORAVE                             │
│               Create music with human language              │
├─────────────────────────────────────────────────────────────┤
│  MODE: [MENU]                                               │
├─────────────────────────────────────────────────────────────┤
│  Commands:                                                  │
│    start    Start the Algorave server                      │
│    ingest   Run documentation ingester (dev only)           │
│    editor   Enter interactive code editor                  │
│    quit     Exit Algorave                                   │
│                                                             │
│  > _                                                        │
└─────────────────────────────────────────────────────────────┘
```

### Editor Mode

```
┌─────────────────────────────────────────────────────────────┐
│  EDITOR MODE                            [Ctrl+C to exit]    │
├─────────────────────────────────────────────────────────────┤
│  1 │ s("bd sd cp")                                          │
│  2 │   .slow(2)                                             │
│  3 │   .room(0.5)                                           │
│  4 │ _                                                      │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│  OUTPUT:                                                    │
│  ✓ Generated Strudel code with drum pattern                │
│  💡 Try adding .gain(0.8) to adjust volume                 │
│                                                             │
│  🤔 Would you like to:                                     │
│     1. Add more percussion sounds                           │
│     2. Adjust the tempo                                     │
│     3. Continue editing                                     │
│                                                             │
│  Choice: _                                                  │
└─────────────────────────────────────────────────────────────┘
```

## Implementation Phases

### Phase 1: Local CLI (Core)

**Scope**: Basic local-only terminal interface

**Components**:
- Welcome screen with command menu
- Command execution (start server, run ingester)
- Environment detection (dev vs production)
- Conditional command visibility

**Entry Point**: `cmd/algorave/main.go`

**Estimated Complexity**: Low-Medium (1-2 days)

### Phase 2: Interactive Editor

**Scope**: Live code editor with AI integration

**Components**:
- Multi-line code editor component
- Integration with agent system
- Real-time output streaming
- Question/answer interaction flow
- Session state management

**New Files**:
- `internal/tui/editor.go` - Editor model and update logic
- `internal/tui/output.go` - Output view and formatting
- `internal/tui/agent.go` - Agent integration

**Estimated Complexity**: Medium (2-3 days)

### Phase 3: Remote SSH Access

**Scope**: Remote access via SSH server

**Components**:
- SSH server using Wish
- Session management (multi-user)
- Authentication (SSH keys + optional tokens)
- Production mode enforcement
- Connection limits and security

**New Files**:
- `internal/ssh/server.go` - Wish SSH server
- `internal/ssh/auth.go` - Authentication handlers
- `internal/ssh/sessions.go` - Session management

**Estimated Complexity**: Medium (1-2 days)

## State Management

### Application States

```go
type AppState int

const (
    StateWelcome AppState = iota  // Welcome/command menu
    StateEditor                    // Code editor mode
    StateOutput                    // Viewing output
    StateQuestion                  // Answering AI question
    StateLoading                   // Processing/waiting
)
```

### Model Structure

```go
type Model struct {
    state        AppState
    mode         string          // "dev" or "production"

    // Editor state
    editorCode   string
    cursorLine   int
    cursorCol    int

    // Output state
    outputText   string
    question     *Question

    // Session state
    agentClient  *agent.Client
    history      []Message

    // UI components
    textInput    textinput.Model
    viewport     viewport.Model
    spinner      spinner.Model
}
```

## Integration Points

### 1. Server Integration

```go
// Start server from CLI
func (m *Model) startServer() tea.Cmd {
    return func() tea.Msg {
        // Reuse existing cmd/server/main.go logic
        return ServerStartedMsg{}
    }
}
```

### 2. Ingester Integration

```go
// Run ingester from CLI (dev only)
func (m *Model) runIngester() tea.Cmd {
    if m.mode == "production" {
        return func() tea.Msg {
            return ErrorMsg{err: "ingester not available in production"}
        }
    }
    return func() tea.Msg {
        // Reuse existing cmd/ingester/main.go logic
        return IngesterCompleteMsg{}
    }
}
```

### 3. Agent Integration

```go
// Send code to agent for analysis
func (m *Model) sendToAgent(code string) tea.Cmd {
    return func() tea.Msg {
        resp, err := m.agentClient.ProcessQuery(ctx, agent.Request{
            Query:       code,
            EditorState: m.editorCode,
            History:     m.history,
        })
        if err != nil {
            return ErrorMsg{err: err}
        }
        return AgentResponseMsg{response: resp}
    }
}
```

## Remote Access Architecture

### SSH Server Flow

```
┌──────────────┐
│  SSH Client  │ (Terminal on user's machine)
└──────┬───────┘
       │ ssh algorave@remote-host
       ↓
┌──────────────────────┐
│   Wish SSH Server    │ (internal/ssh/server.go)
│  - Auth handling     │
│  - Session creation  │
└──────┬───────────────┘
       │
       ↓
┌──────────────────────┐
│  Bubbletea Session   │ (Per-user TUI instance)
│  - Editor state      │
│  - Agent integration │
└──────────────────────┘
```

### Multi-User Considerations

**Phase 3 Implementation**:
- Each SSH connection gets isolated TUI instance
- No shared state between users initially
- Future: Shared sessions for collaborative editing

**Security**:
- SSH key authentication required
- Optional token-based auth for web integrations
- Rate limiting on agent requests
- Connection limits per user

## Environment Modes

### Development Mode

```bash
# All commands available
ALGORAVE_ENV=development algorave

Commands:
  start    - Start server
  ingest   - Run ingester
  editor   - Code editor
  quit     - Exit
```

### Production Mode

```bash
# Ingester hidden
ALGORAVE_ENV=production algorave

Commands:
  start    - Start server
  editor   - Code editor
  quit     - Exit
```

Detection logic:
```go
mode := os.Getenv("ALGORAVE_ENV")
if mode == "" {
    mode = "development" // default to dev
}
```

## Configuration

### CLI Config

```go
type Config struct {
    Mode           string  // "development" or "production"
    SSHEnabled     bool
    SSHPort        int
    SSHHostKey     string
    MaxConnections int
    AgentEndpoint  string
}
```

### Environment Variables

```bash
# Mode
ALGORAVE_ENV=production

# SSH Server (Phase 3)
ALGORAVE_SSH_ENABLED=true
ALGORAVE_SSH_PORT=2222
ALGORAVE_SSH_HOST_KEY=/path/to/host_key

# Agent
ALGORAVE_AGENT_ENDPOINT=http://localhost:8080
```

## Dependencies

### External Packages

```go
require (
    github.com/charmbracelet/bubbletea v0.25.0
    github.com/charmbracelet/lipgloss v0.9.1
    github.com/charmbracelet/bubbles v0.17.1
    github.com/charmbracelet/wish v1.3.0      // Phase 3
)
```

### Internal Dependencies

- `internal/agent` - Agent client for code generation
- `internal/config` - Configuration management
- `cmd/server` - Server startup logic
- `cmd/ingester` - Ingester logic (dev mode)

## User Workflows

### Workflow 1: Local Development

1. Developer runs `algorave` locally
2. Sees welcome screen with commands
3. Types `start` to launch server
4. Types `editor` to write code
5. Gets real-time AI assistance
6. Server runs in background

### Workflow 2: Remote Coding Session

1. User SSHs to remote Algorave instance
2. Sees welcome screen (production mode)
3. Types `editor` to start coding
4. Writes Strudel code with AI help
5. Asks questions, gets suggestions
6. Disconnects, session state saved

### Workflow 3: Production Access

1. Remote user connects via SSH
2. Only sees `start`, `editor`, `quit` commands
3. Ingester hidden (production mode)
4. Safe for public/untrusted users

## Error Handling

### Graceful Degradation

- Agent unavailable → Show error, allow retry
- Network issues → Cache locally, retry on reconnect
- Invalid commands → Show help text
- SSH auth failure → Show clear error message

### User Feedback

```go
type Notification struct {
    Type    string // "success", "error", "info", "warning"
    Message string
    TTL     time.Duration
}
```

Display in status bar with appropriate styling.

## Future Enhancements

### Potential Features

- Syntax highlighting for Strudel code
- Code completion/IntelliSense
- Session recording/playback
- Collaborative multi-user editing
- WebSocket alternative to SSH
- Browser-based terminal (xterm.js)
- Plugin system for custom commands

### Scalability Considerations

**Current**: Single-server SSH access
**Future**:
- Load balancer with multiple SSH servers
- Redis-backed session storage
- Horizontal scaling for agent requests

## Related Documentation

- [Product Architecture](./PRODUCT_ARCHITECTURE.md) - User-facing product features
- [RAG Architecture](./RAG_ARCHITECTURE.md) - Agent and retrieval system
- [Hybrid Retrieval Guide](./HYBRID_RETRIEVAL_GUIDE.md) - Search implementation

## Implementation Checklist

### Phase 1: Local CLI
- [ ] Create `cmd/algorave/main.go` entry point
- [ ] Implement welcome screen (`internal/tui/welcome.go`)
- [ ] Add command menu and input handling
- [ ] Integrate server start command
- [ ] Integrate ingester command (dev mode only)
- [ ] Add environment mode detection
- [ ] Create basic styling with Lipgloss

### Phase 2: Interactive Editor
- [ ] Implement multi-line editor component
- [ ] Add cursor navigation and text editing
- [ ] Create output view component
- [ ] Integrate agent client
- [ ] Implement streaming response handling
- [ ] Add question/answer interaction
- [ ] Persist session state

### Phase 3: Remote SSH Access
- [ ] Set up Wish SSH server
- [ ] Implement SSH authentication
- [ ] Add per-session TUI instances
- [ ] Create session management
- [ ] Add connection limits
- [ ] Test multi-user scenarios
- [ ] Production deployment guide
