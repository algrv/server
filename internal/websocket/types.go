package websocket

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// message type constants for websocket communication
const (
	// is sent when a user updates the code
	TypeCodeUpdate = "code_update"

	// is sent when a new user joins the session
	TypeUserJoined = "user_joined"

	// is sent when a user leaves the session
	TypeUserLeft = "user_left"

	// is sent when a user sends a chat message
	TypeChatMessage = "chat_message"

	// is sent when an error occurs
	TypeError = "error"

	// is sent by clients to keep the connection alive
	TypePing = "ping"

	// is sent by server in response to ping
	TypePong = "pong"

	// is sent by server before shutdown
	TypeServerShutdown = "server_shutdown"

	// is sent to connecting client with session info
	TypeSessionState = "session_state"

	// is sent when host/co-author starts playback
	TypePlay = "play"

	// is sent when host/co-author stops playback
	TypeStop = "stop"

	// is sent when host ends the session
	TypeSessionEnded = "session_ended"

	// is sent when paste lock status changes
	TypePasteLockChanged = "paste_lock_changed"
)

// client connection constants
const (
	// time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512 KB

	// rate limiting constants
	maxCodeUpdatesPerSecond  = 10 // maximum code updates per second
	maxChatMessagesPerMinute = 20 // maximum chat messages per minute

	// content size limits
	maxCodeSize        = 100 * 1024 // 100 KB maximum code size
	maxChatMessageSize = 5000       // 5000 characters maximum chat message size
)

// hub connection limit constants
const (
	maxConnectionsPerUser = 5
	maxConnectionsPerIP   = 10
)

// errors
var (
	ErrSessionNotFound         = errors.New("session not found")
	ErrUnauthorized            = errors.New("unauthorized")
	ErrInvalidMessage          = errors.New("invalid message format")
	ErrClientNotFound          = errors.New("client not found")
	ErrClientAlreadyRegistered = errors.New("client already registered")
	ErrSessionFull             = errors.New("session is full")
	ErrReadOnly                = errors.New("read-only access")
	ErrConnectionClosed        = errors.New("connection closed")
	ErrRateLimitExceeded       = errors.New("rate limit exceeded")
	ErrCodeTooLarge            = errors.New("code too large")
)

// represents a websocket message with typed payload
type Message struct {
	Type      string          `json:"type"`
	SessionID string          `json:"session_id"`
	ClientID  string          `json:"-"` // Internal only, not sent to clients
	UserID    string          `json:"user_id,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Sequence  uint64          `json:"seq,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

// contains code update information
type CodeUpdatePayload struct {
	Code        string `json:"code"`
	CursorLine  int    `json:"cursor_line,omitempty"`
	CursorCol   int    `json:"cursor_col,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Source      string `json:"source,omitempty"` // 'typed' | 'loaded_strudel' | 'forked' | 'paste'
}

// contains information about a newly joined user
type UserJoinedPayload struct {
	UserID      string `json:"user_id,omitempty"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"` // "host", "co-author", "viewer"
}

// contains information about a user who left
type UserLeftPayload struct {
	UserID      string `json:"user_id,omitempty"`
	DisplayName string `json:"display_name"`
}

// contains a chat message from a user
type ChatMessagePayload struct {
	Message     string `json:"message"`
	DisplayName string `json:"display_name,omitempty"`
}

// contains information about server shutdown
type ServerShutdownPayload struct {
	Reason string `json:"reason"`
}

// contains session info sent to connecting client
type SessionStatePayload struct {
	Code         string                    `json:"code"`
	YourRole     string                    `json:"your_role"`
	Participants []SessionStateParticipant `json:"participants"`
	ChatHistory  []SessionStateChatMessage `json:"chat_history"`
}

// represents a chat message in the chat history
type SessionStateChatMessage struct {
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	Content     string `json:"content"`
	Timestamp   int64  `json:"timestamp"` // Unix milliseconds
}

// represents a participant in session_state
type SessionStateParticipant struct {
	UserID      string `json:"user_id,omitempty"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

// contains playback start information
type PlayPayload struct {
	DisplayName string `json:"display_name"`
}

// contains playback stop information
type StopPayload struct {
	DisplayName string `json:"display_name"`
}

// contains session termination information
type SessionEndedPayload struct {
	Reason string `json:"reason,omitempty"`
}

// contains paste lock status change
type PasteLockChangedPayload struct {
	Locked bool   `json:"locked"`
	Reason string `json:"reason,omitempty"` // "paste_detected", "edits_sufficient", "ttl_expired"
}

// represents a websocket client connection
type Client struct {
	// unique identifier for this client
	ID string

	// session ID this client is connected to
	SessionID string

	// user ID (empty for anonymous users)
	UserID string

	// display name for this client
	DisplayName string

	// role in the session (host, co-author, viewer)
	Role string

	// whether this client has an authenticated user account
	IsAuthenticated bool

	// IP address of the client (for connection tracking)
	IPAddress string

	// initial code to send on connect (for joining existing sessions)
	InitialCode string

	// initial chat history to send on connect
	InitialChatHistory []SessionStateChatMessage

	// websocket connection
	conn *websocket.Conn

	// hub reference for message broadcasting
	hub *Hub

	// buffered channel of outbound messages
	send chan []byte

	// mutex for thread-safe operations
	mu sync.RWMutex

	// flag indicating if client is closed
	closed bool

	// rate limiting: code update timestamps (sliding window)
	codeUpdateTimestamps []time.Time

	// rate limiting: chat message timestamps (sliding window)
	chatMessageTimestamps []time.Time
}

// maintains the set of active clients and broadcasts messages to sessions
type Hub struct {
	// registered clients by session ID and client ID
	sessions map[string]map[string]*Client

	// register requests from clients
	Register chan *Client

	// unregister requests from clients
	Unregister chan *Client

	// broadcast messages to all clients in a session
	Broadcast chan *Message

	// mutex for thread-safe access to sessions
	mu sync.RWMutex

	// message handlers for different message types
	handlers map[string]MessageHandler

	// flag indicating if hub is running
	running bool

	// channel to signal shutdown
	shutdown chan struct{}

	// connection tracking: user ID -> count of connections
	userConnections map[string]int

	// connection tracking: IP address -> count of connections
	ipConnections map[string]int

	// sequence numbers per session for message ordering
	sessionSequences map[string]uint64

	// callback for client disconnect (e.g., save code to DB)
	onClientDisconnect func(client *Client)
}

// processes a specific message type
type MessageHandler func(hub *Hub, client *Client, msg *Message) error
