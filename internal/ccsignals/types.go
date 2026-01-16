// package ccsignals provides CC Signal enforcement for AI agents.
// detects paste operations and blocks AI requests on protected content
// until the user makes significant edits (demonstrating transformative use).
package ccsignals

import (
	"context"
	"time"
)

// represents Creative Commons Signals for AI usage consent
type CCSignal string

const (
	SignalCredit    CCSignal = "cc-cr" // credit: allow AI use with attribution
	SignalDirect    CCSignal = "cc-dc" // credit + direct: attribution + financial support
	SignalEcosystem CCSignal = "cc-ec" // credit + ecosystem: attribution + contribute to commons
	SignalOpen      CCSignal = "cc-op" // credit + open: attribution + keep derivatives open
	SignalNoAI      CCSignal = "no-ai" // no AI: explicitly opt-out of AI usage
)

// returns true if the signal is a valid CC Signal value
func (s CCSignal) IsValid() bool {
	switch s {
	case SignalCredit, SignalDirect, SignalEcosystem, SignalOpen, SignalNoAI:
		return true
	default:
		return false
	}
}

// returns true if this signal permits AI usage
// empty/missing signal defaults to no-ai (restrictive default)
func (s CCSignal) AllowsAI() bool {
	switch s {
	case SignalCredit, SignalDirect, SignalEcosystem, SignalOpen:
		return true
	default:
		// no-ai or empty/missing signal blocks AI
		return false
	}
}

// holds configuration for the detection system
type Config struct {
	PasteDeltaThreshold int
	PasteLineThreshold  int
	UnlockThreshold     float64
	LockTTL             time.Duration
}

// returns sensible defaults for the detection system
func DefaultConfig() Config {
	return Config{
		PasteDeltaThreshold: 200,
		PasteLineThreshold:  50,
		UnlockThreshold:     0.30,
		LockTTL:             1 * time.Hour,
	}
}

// represents the current lock state for a session
type LockState struct {
	Locked       bool
	BaselineCode string
	LockedAt     time.Time
	Reason       string
}

// represents a match result from content validation
type ContentMatch struct {
	Found    bool
	OwnerID  string
	IsPublic bool
	CCSignal CCSignal
}

// defines the interface for storing paste locks
type LockStore interface {
	SetLock(ctx context.Context, sessionID, baselineCode string, ttl time.Duration) error
	GetLock(ctx context.Context, sessionID string) (*LockState, error)
	RemoveLock(ctx context.Context, sessionID string) error
	RefreshTTL(ctx context.Context, sessionID string, ttl time.Duration) error
}

// defines the interface for validating content ownership
type ContentValidator interface {
	ValidateOwnership(ctx context.Context, userID, code string) (*ContentMatch, error)
	ValidatePublicContent(ctx context.Context, code string) (*ContentMatch, error)
}

// defines the interface for session management
type SessionProvider interface {
	IsSessionActive(sessionID string) bool
	GetSessionCode(ctx context.Context, sessionID string) (string, error)
}
