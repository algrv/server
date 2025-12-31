# CLI Implementation Plan - Review Document

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
│  Generated code with drum pattern           │
│  Tip: Try .gain(0.8) to adjust volume       │
│                                             │
│  Did you mean to add reverb?                │
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

**Status**: REMOVED - Not implemented. Would use Wish library for SSH-based TUI access. May revisit if remote access becomes a priority.

---

## File Structure

```
cmd/algorave/main.go       # CLI entry point
internal/tui/              # TUI components (app, welcome, editor, output, styles)
```

---

## Dependencies

Charm ecosystem: `bubbletea`, `lipgloss`, `bubbles`

```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
```

---

## Configuration

Key environment variables:
- `ALGORAVE_ENV` - `development` or `production`
- `ALGORAVE_AGENT_ENDPOINT` - Agent API endpoint

---

## Deployment

```bash
# Development
ALGORAVE_ENV=development go run cmd/algorave/main.go

# Production build
CGO_ENABLED=0 GOOS=linux go build -o algorave cmd/algorave/main.go
```

---

## Success Criteria

- Phase 1: CLI launches, commands execute, env modes work
- Phase 2: Editor responsive, agent integration works, users complete tasks

