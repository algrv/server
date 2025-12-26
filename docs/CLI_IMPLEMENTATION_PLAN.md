# CLI Implementation Plan - Review Document

**Date**: 2025-12-26
**Status**: Ready for Review
**Total Estimated Time**: 4-7 days for all 3 phases

---

## Executive Summary

This document outlines the implementation plan for the Algorave CLI/TUI system - an interactive terminal interface that provides:
- Local command menu for development operations
- Live code editor with AI-powered assistance
- Remote SSH access for collaborative coding
- Production-safe deployment with filtered commands

## Why Build This?

### Current State
- Server runs via `cmd/server/main.go`
- Ingester runs via `cmd/ingester/main.go`
- Users interact via HTTP API or web frontend
- No interactive terminal experience

### Target State
- Single `algorave` command for all operations
- Interactive TUI for coding with AI assistance
- Remote access for users to code from anywhere
- Professional terminal experience with fancy UI

### Benefits
1. **Developer Experience**: One command to rule them all
2. **Remote Accessibility**: SSH access means anyone can code from terminal
3. **Production Ready**: Safe filtering of dangerous commands
4. **AI Integration**: Real-time assistance while coding
5. **Professional Polish**: Modern TUI with Charm components

---

## Technical Approach

### Technology Stack

| Component | Technology | Why |
|-----------|-----------|-----|
| TUI Framework | Bubbletea | Industry standard, Elm-inspired architecture |
| Styling | Lipgloss | Beautiful terminal layouts and colors |
| UI Components | Bubbles | Pre-built text input, viewports, spinners |
| SSH Server | Wish | Built by Charm specifically for TUI apps |
| Language | Go | Matches existing codebase |

### Architecture Pattern

```
┌─────────────────────────────────────────────┐
│         User's Terminal                     │
│  (Local or SSH connected)                   │
└─────────────────┬───────────────────────────┘
                  │
                  ↓
┌─────────────────────────────────────────────┐
│     Bubbletea Application                   │
│  ┌──────────────────────────────────────┐   │
│  │  State Machine                       │   │
│  │  - Welcome Screen                    │   │
│  │  - Editor Mode                       │   │
│  │  - Output View                       │   │
│  │  - Question Handling                 │   │
│  └──────────────────────────────────────┘   │
└─────────────────┬───────────────────────────┘
                  │
                  ↓
┌─────────────────────────────────────────────┐
│     Integration Layer                       │
│  - Agent Client (AI requests)               │
│  - Server Starter (cmd/server)              │
│  - Ingester Runner (cmd/ingester)           │
└─────────────────────────────────────────────┘
```

---

## Implementation Phases

### Phase 1: Local CLI (Core)
**Duration**: 1-2 days
**Complexity**: Low-Medium

#### What We're Building
Basic terminal interface with command menu for local development.

#### Components
```
cmd/algorave/
  main.go                    # Entry point, initializes Bubbletea app

internal/tui/
  app.go                     # Main application model
  welcome.go                 # Welcome screen (command menu)
  styles.go                  # Lipgloss styling
  commands.go                # Command execution (start, ingest, quit)
```

#### Features
- Welcome screen with ASCII art logo
- Command menu: `start`, `ingest`, `editor`, `quit`
- Environment detection (dev vs production)
- Conditional commands: hide `ingest` in production
- Keyboard navigation and command input
- Execute existing `cmd/server` and `cmd/ingester` logic

#### User Flow
```
1. User runs: algorave
2. Sees welcome screen
3. Types: start
4. Server starts in background
5. Types: quit
6. Clean shutdown
```

#### Success Criteria
- [ ] Can launch CLI with `algorave` command
- [ ] Welcome screen displays with commands
- [ ] Can start server successfully
- [ ] Can run ingester (dev mode only)
- [ ] Production mode hides ingester command
- [ ] Clean exit on `quit`

---

### Phase 2: Interactive Editor
**Duration**: 2-3 days
**Complexity**: Medium

#### What We're Building
Live code editor with real-time AI assistance for Strudel code.

#### Components (Additional)
```
internal/tui/
  editor.go                  # Multi-line code editor
  output.go                  # Results/output view
  agent.go                   # Agent client integration
  state.go                   # Session state management
```

#### Features
- Multi-line text editor with cursor navigation
- Syntax-aware input handling
- Integration with existing agent system
- Stream AI responses to output view
- Question/answer interaction
- Session history persistence
- Mode switching (menu ↔ editor)

#### Editor Interface
```
┌─────────────────────────────────────────────┐
│  EDITOR MODE              [Ctrl+C to exit]  │
├─────────────────────────────────────────────┤
│  1 │ s("bd sd cp")                          │
│  2 │   .slow(2)                             │
│  3 │   .room(0.5)                           │
│  4 │ _                                      │
├─────────────────────────────────────────────┤
│  OUTPUT:                                    │
│  ✓ Generated code with drum pattern        │
│  💡 Try .gain(0.8) to adjust volume        │
│                                             │
│  🤔 Did you mean to add reverb?            │
│     [y/n]: _                                │
└─────────────────────────────────────────────┘
```

#### Agent Integration
```go
// Send code to agent
request := agent.Request{
    Query:       userInput,
    EditorState: currentCode,
    History:     previousMessages,
}

// Stream response
response := agentClient.ProcessQuery(ctx, request)

// Update output view with streaming text
```

#### Success Criteria
- [ ] Can enter editor mode from menu
- [ ] Multi-line editing works smoothly
- [ ] Agent integration sends code successfully
- [ ] AI responses stream to output view
- [ ] Can handle clarification questions
- [ ] Session state persists during session
- [ ] Can exit back to menu

---

### Phase 3: Remote SSH Access
**Duration**: 1-2 days
**Complexity**: Medium

#### What We're Building
SSH server for remote terminal access with multi-user support.

#### Components (Additional)
```
internal/ssh/
  server.go                  # Wish SSH server
  auth.go                    # SSH authentication
  sessions.go                # Per-user session management
  middleware.go              # Rate limiting, logging
```

#### Features
- SSH server using Wish library
- SSH key authentication
- Per-connection TUI instances
- Session isolation (no shared state)
- Production mode enforcement
- Connection limits and security
- Graceful shutdown handling

#### Architecture
```
┌──────────────────┐
│  SSH Client #1   │ ──┐
└──────────────────┘   │
                       │
┌──────────────────┐   │    ┌──────────────────┐
│  SSH Client #2   │ ──┼───→│  Wish Server     │
└──────────────────┘   │    │  :2222           │
                       │    └────────┬─────────┘
┌──────────────────┐   │             │
│  SSH Client #3   │ ──┘             │
└──────────────────┘                 ↓
                            ┌─────────────────┐
                            │  TUI Instance 1 │
                            │  TUI Instance 2 │
                            │  TUI Instance 3 │
                            └─────────────────┘
                                     │
                            ┌─────────────────┐
                            │  Shared Agent   │
                            │  Backend        │
                            └─────────────────┘
```

#### SSH Server Setup
```go
// Create Wish server
server, err := wish.NewServer(
    wish.WithAddress(fmt.Sprintf(":%d", cfg.SSHPort)),
    wish.WithHostKeyPath(cfg.SSHHostKey),
    wish.WithMiddleware(
        bubbletea.Middleware(tuiHandler),
        activeterm.Middleware(),
        logging.Middleware(),
    ),
)

// Handle each connection
func tuiHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
    // Create isolated TUI instance for this user
    model := tui.NewModel(productionMode)
    return model, []tea.ProgramOption{tea.WithAltScreen()}
}
```

#### Security Considerations
- SSH key auth (no passwords)
- Rate limiting on agent requests (prevent abuse)
- Connection limits per IP
- Production mode forces safe commands only
- Logging all connections and commands
- Timeout idle sessions

#### Success Criteria
- [ ] Can SSH to `algorave@host`
- [ ] Each connection gets isolated TUI
- [ ] SSH authentication works correctly
- [ ] Multiple users can connect simultaneously
- [ ] Production mode enforced for remote users
- [ ] Connection limits prevent DoS
- [ ] Sessions timeout after inactivity

---

## File Structure

### After All Phases

```
algorave/
├── cmd/
│   ├── algorave/              # NEW: CLI entry point
│   │   └── main.go
│   ├── ingester/              # Existing
│   │   └── main.go
│   └── server/                # Existing
│       └── main.go
│
├── internal/
│   ├── tui/                   # NEW: TUI components
│   │   ├── app.go             # Main Bubbletea app
│   │   ├── welcome.go         # Welcome screen
│   │   ├── editor.go          # Code editor
│   │   ├── output.go          # Output view
│   │   ├── agent.go           # Agent integration
│   │   ├── state.go           # State management
│   │   ├── styles.go          # Lipgloss styles
│   │   └── commands.go        # Command handlers
│   │
│   ├── ssh/                   # NEW: SSH server (Phase 3)
│   │   ├── server.go
│   │   ├── auth.go
│   │   ├── sessions.go
│   │   └── middleware.go
│   │
│   ├── agent/                 # Existing
│   ├── auth/                  # Existing
│   ├── config/                # Existing
│   └── ...
│
└── docs/
    ├── system-specs/
    │   ├── CLI_ARCHITECTURE.md       # NEW
    │   ├── PRODUCT_ARCHITECTURE.md   # Updated
    │   └── ...
    └── CLI_IMPLEMENTATION_PLAN.md    # NEW (this doc)
```

---

## Dependencies

### New Dependencies to Add

```go
require (
    // Phase 1 & 2
    github.com/charmbracelet/bubbletea v0.25.0
    github.com/charmbracelet/lipgloss v0.9.1
    github.com/charmbracelet/bubbles v0.17.1

    // Phase 3
    github.com/charmbracelet/wish v1.3.0
    github.com/charmbracelet/ssh v0.0.0-20230822194956-1a051f898e09

    // Existing dependencies continue
    // (no changes to agent, llm, storage, etc.)
)
```

### Installation
```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/charmbracelet/wish@latest  # Phase 3
```

---

## Configuration

### Environment Variables

```bash
# Mode (dev or production)
ALGORAVE_ENV=production

# SSH Server (Phase 3)
ALGORAVE_SSH_ENABLED=true
ALGORAVE_SSH_PORT=2222
ALGORAVE_SSH_HOST_KEY=/path/to/ssh_host_key
ALGORAVE_SSH_MAX_CONNECTIONS=50

# Agent endpoint
ALGORAVE_AGENT_ENDPOINT=http://localhost:8080

# Existing vars
OPENAI_API_KEY=...
SUPABASE_CONNECTION_STRING=...
ANTHROPIC_API_KEY=...
```

### Config File (Optional)

```yaml
# algorave.yaml
mode: production

ssh:
  enabled: true
  port: 2222
  host_key_path: /etc/algorave/ssh_host_key
  max_connections: 50

agent:
  endpoint: http://localhost:8080
  timeout: 30s

editor:
  auto_save: true
  session_timeout: 30m
```

---

## Testing Strategy

### Unit Tests
- TUI component state transitions
- Command execution logic
- Agent integration mocking
- SSH auth handlers

### Integration Tests
- End-to-end CLI flows
- Server/ingester integration
- SSH connection handling
- Multi-user scenarios

### Manual Testing Scenarios

**Phase 1**:
- [ ] Start CLI, run each command
- [ ] Test dev vs production mode
- [ ] Verify server starts correctly
- [ ] Verify ingester hidden in production

**Phase 2**:
- [ ] Write multi-line Strudel code
- [ ] Get AI suggestions
- [ ] Answer clarification questions
- [ ] Exit and re-enter editor

**Phase 3**:
- [ ] SSH from remote machine
- [ ] Multiple simultaneous connections
- [ ] Connection timeout
- [ ] Rate limiting under load

---

## Deployment Plan

### Local Development
```bash
# Clone repo
git clone <repo>
cd algorave

# Install dependencies
go mod download

# Build CLI
go build -o algorave cmd/algorave/main.go

# Run in dev mode
ALGORAVE_ENV=development ./algorave
```

### Production Deployment
```bash
# Build for production
CGO_ENABLED=0 GOOS=linux go build -o algorave cmd/algorave/main.go

# Copy to server
scp algorave user@server:/usr/local/bin/

# Generate SSH host key
ssh-keygen -t ed25519 -f /etc/algorave/ssh_host_key

# Run with systemd
sudo systemctl start algorave-ssh
```

### Systemd Service (Phase 3)
```ini
[Unit]
Description=Algorave SSH Server
After=network.target

[Service]
Type=simple
User=algorave
Environment="ALGORAVE_ENV=production"
Environment="ALGORAVE_SSH_ENABLED=true"
Environment="ALGORAVE_SSH_PORT=2222"
ExecStart=/usr/local/bin/algorave --ssh
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

---

## Risk Assessment

### Potential Issues

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Bubbletea learning curve | Medium | Use examples from official docs, start simple |
| SSH security vulnerabilities | High | Use Wish (battle-tested), key-only auth, rate limiting |
| Multi-user state conflicts | Medium | Isolated sessions per connection (no sharing) |
| Agent API rate limits | Medium | Connection limits, request queuing |
| Terminal compatibility | Low | Bubbletea handles most edge cases |

### Unknowns
- Performance with 50+ simultaneous SSH connections
- Agent response time under load
- Terminal rendering on all platforms

### Mitigation Strategies
- Start with Phase 1 (minimal risk)
- Test Phase 3 thoroughly before production
- Implement connection limits and monitoring
- Add graceful degradation for agent failures

---

## Success Metrics

### Phase 1
- ✓ CLI launches without errors
- ✓ All commands execute correctly
- ✓ Environment modes work as expected

### Phase 2
- ✓ Editor is usable and responsive
- ✓ Agent integration works reliably
- ✓ Users can complete coding tasks

### Phase 3
- ✓ SSH server runs stably
- ✓ Multiple users can connect
- ✓ Production deployment successful

---

## Next Steps

1. **Review This Document**: User approves plan
2. **Phase 1 Implementation**: Build local CLI
3. **Phase 1 Testing**: Verify core functionality
4. **Phase 2 Implementation**: Add editor
5. **Phase 2 Testing**: Test AI integration
6. **Phase 3 Implementation**: Add SSH server
7. **Phase 3 Testing**: Multi-user scenarios
8. **Production Deployment**: Deploy to remote server

---

## Questions for Review

Before we start implementation, please confirm:

1. **Scope**: Are all 3 phases approved, or start with Phase 1 only?
2. **Timeline**: Is 4-7 days acceptable, or should we prioritize?
3. **Features**: Any additions/removals to the planned features?
4. **Tech Stack**: Happy with Charm ecosystem (Bubbletea, Wish)?
5. **Production**: Will this be deployed for public SSH access?
6. **Authentication**: SSH keys only, or also support passwords/tokens?
7. **Branding**: Any specific ASCII art or styling preferences for welcome screen?

---

## Appendix: Example User Interactions

### Example 1: Local Development Session

```bash
$ algorave
┌─────────────────────────────────────────┐
│          🎵 ALGORAVE                    │
│   Create music with human language      │
├─────────────────────────────────────────┤
│  Commands:                              │
│    start    Start the server            │
│    ingest   Run doc ingester            │
│    editor   Interactive code editor     │
│    quit     Exit                        │
│                                         │
│  > start                                │
└─────────────────────────────────────────┘

✓ Server started on http://localhost:8080

> editor

┌─────────────────────────────────────────┐
│  EDITOR MODE         [Ctrl+C to exit]   │
├─────────────────────────────────────────┤
│  1 │ make a drum beat                   │
│  2 │ _                                  │
├─────────────────────────────────────────┤
│  OUTPUT:                                │
│  ✓ Generated:                           │
│    s("bd sd cp hh").fast(2)             │
│                                         │
│  💡 This creates a basic drum pattern   │
│     Try adding .room(0.5) for reverb   │
└─────────────────────────────────────────┘
```

### Example 2: Remote SSH Session

```bash
$ ssh algorave@remote.algorave.io
┌─────────────────────────────────────────┐
│          🎵 ALGORAVE                    │
│        Remote Coding Session            │
├─────────────────────────────────────────┤
│  Commands:                              │
│    start    Start the server            │
│    editor   Interactive code editor     │
│    quit     Disconnect                  │
│                                         │
│  > editor                               │
└─────────────────────────────────────────┘

[User writes code with AI assistance...]
```

---

**Ready for implementation?** Please review and provide feedback!
