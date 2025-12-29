package anonsessions

import (
	"sync"
	"time"

	"github.com/algorave/server/algorave/strudels"
)

type Manager struct {
	sessions map[string]*AnonymousSession
	mu       sync.RWMutex
	stopChan chan struct{}
}

type AnonymousSession struct {
	ID                  string                       `json:"id"`
	EditorState         string                       `json:"editor_state"`
	ConversationHistory strudels.ConversationHistory `json:"conversation_history"`
	CreatedAt           time.Time                    `json:"created_at"`
	LastActivity        time.Time                    `json:"last_activity"`
	mu                  sync.RWMutex                 `json:"-"`
}
