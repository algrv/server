package buffer

import "time"

// chat message waiting to be flushed to postgres
type BufferedChatMessage struct {
	SessionID   string    `json:"session_id"`
	UserID      string    `json:"user_id,omitempty"`
	Content     string    `json:"content"`
	DisplayName string    `json:"display_name,omitempty"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// redis key patterns
const (
	// session:{sessionID}:code - stores current code as string
	keySessionCode = "session:%s:code"

	// session:{sessionID}:messages - stores messages as JSON list
	keySessionMessages = "session:%s:messages"

	// dirty_sessions:code - set of session IDs with unflushed code changes
	keyDirtySessionsCode = "dirty_sessions:code"

	// dirty_sessions:messages - set of session IDs with unflushed messages
	keyDirtySessionsMessages = "dirty_sessions:messages"

	// paste_lock:{sessionID} - indicates session has paste lock active
	keyPasteLock = "paste_lock:%s"

	// paste_baseline:{sessionID} - stores code at time of paste for edit distance calculation
	keyPasteBaseline = "paste_baseline:%s"

	// rag_cache:{sessionID} - stores cached rag results (docs + examples)
	keyRAGCache = "rag_cache:%s"
)

// ttl for rag cache (reuse docs for follow-up messages within this window)
const RAGCacheTTL = 10 * time.Minute

// stores retrieved docs and examples for reuse
type CachedRAGResult struct {
	Docs     []CachedDoc     `json:"docs"`
	Examples []CachedExample `json:"examples"`
	Query    string          `json:"query"` // original query that triggered retrieval
}

// simplified doc for caching (matches retriever.SearchResult)
type CachedDoc struct {
	ID           string  `json:"id"`
	PageName     string  `json:"page_name"`
	PageURL      string  `json:"page_url"`
	SectionTitle string  `json:"section_title"`
	Content      string  `json:"content"`
	Similarity   float32 `json:"similarity"`
}

// simplified example for caching (matches retriever.ExampleResult)
type CachedExample struct {
	ID          string   `json:"id"`
	UserID      string   `json:"user_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Code        string   `json:"code"`
	Tags        []string `json:"tags"`
	AuthorName  string   `json:"author_name"`
	URL         string   `json:"url"`
	Similarity  float32  `json:"similarity"`
}

// paste detection constants
const (
	PasteLockTTL        = 1 * time.Hour
	PasteDeltaThreshold = 200  // characters added in single update
	PasteLineThreshold  = 50   // lines added in single update
	UnlockThreshold     = 0.30 // 30% edit distance required to unlock
)
