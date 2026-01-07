package ccsignals

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrNilStore = errors.New("ccsignals: store is nil")
)

// handles paste detection and lock management
type Detector struct {
	config       Config
	store        LockStore
	validator    ContentValidator
	fingerprints *IndexedFingerprintStore
}

// creates a new detector with the given dependencies
func NewDetector(config Config, store LockStore, validator ContentValidator) *Detector {
	return &Detector{
		config:    config,
		store:     store,
		validator: validator,
	}
}

// enables fingerprint-based similarity detection
func (d *Detector) WithFingerprints(fps *IndexedFingerprintStore) *Detector {
	d.fingerprints = fps
	return d
}

// contains the result of paste detection
type DetectionResult struct {
	ShouldLock       bool
	Reason           string
	MatchedContent   *ContentMatch
	FingerprintMatch *MatchResult
}

// analyzes a code update and determines if it should be locked
func (d *Detector) DetectPaste(ctx context.Context, _, userID, previousCode, newCode string) (*DetectionResult, error) {
	if !d.IsLargeDelta(previousCode, newCode) {
		return &DetectionResult{
			ShouldLock: false,
			Reason:     "no large delta detected",
		}, nil
	}

	// large delta detected - validate against legitimate sources

	// check 1: does code match user's own content?
	if userID != "" && d.validator != nil {
		match, err := d.validator.ValidateOwnership(ctx, userID, newCode)
		if err == nil && match != nil && match.Found {
			return &DetectionResult{
				ShouldLock:     false,
				Reason:         "code matches user's own content",
				MatchedContent: match,
			}, nil
		}
	}

	// check 2: does code match public content that allows AI?
	if d.validator != nil {
		match, err := d.validator.ValidatePublicContent(ctx, newCode)
		if err == nil && match != nil && match.Found {
			if match.CCSignal.AllowsAI() {
				return &DetectionResult{
					ShouldLock:     false,
					Reason:         "code matches public content that allows AI",
					MatchedContent: match,
				}, nil
			}

			return &DetectionResult{
				ShouldLock:     true,
				Reason:         "code matches public content with no-ai restriction",
				MatchedContent: match,
			}, nil
		}
	}

	// check 3: fingerprint similarity detection
	if d.fingerprints != nil {
		fpMatch := d.fingerprints.FindBestMatch(newCode)
		if fpMatch != nil {
			if !fpMatch.Record.CCSignal.AllowsAI() {
				return &DetectionResult{
					ShouldLock:       true,
					Reason:           "content is similar to protected work with no-ai restriction",
					FingerprintMatch: fpMatch,
				}, nil
			}

			return &DetectionResult{
				ShouldLock:       false,
				Reason:           "content is similar to work that allows AI",
				FingerprintMatch: fpMatch,
			}, nil
		}
	}

	// no legitimate source found - this is likely an external paste
	return &DetectionResult{
		ShouldLock: true,
		Reason:     "large delta with no matching content in database",
	}, nil
}

// handles a code update event, managing locks as needed
func (d *Detector) ProcessCodeUpdate(ctx context.Context, sessionID, userID, previousCode, newCode string) error {
	if d.store == nil {
		return ErrNilStore
	}

	result, err := d.DetectPaste(ctx, sessionID, userID, previousCode, newCode)
	if err != nil {
		return err
	}

	if result.ShouldLock {
		return d.store.SetLock(ctx, sessionID, newCode, d.config.LockTTL)
	}

	return d.CheckUnlock(ctx, sessionID, newCode)
}

// checks if edits are significant enough to unlock a session
func (d *Detector) CheckUnlock(ctx context.Context, sessionID, currentCode string) error {
	if d.store == nil {
		return ErrNilStore
	}

	state, err := d.store.GetLock(ctx, sessionID)
	if err != nil {
		return err
	}

	if state == nil || !state.Locked {
		return nil
	}

	if d.IsSignificantEdit(state.BaselineCode, currentCode) {
		return d.store.RemoveLock(ctx, sessionID)
	}

	return d.store.RefreshTTL(ctx, sessionID, d.config.LockTTL)
}

// checks if a session is currently paste-locked
func (d *Detector) IsLocked(ctx context.Context, sessionID string) (bool, error) {
	if d.store == nil {
		return false, ErrNilStore
	}

	state, err := d.store.GetLock(ctx, sessionID)
	if err != nil {
		return false, err
	}

	return state != nil && state.Locked, nil
}

// determines if a code update has a large delta
func (d *Detector) IsLargeDelta(previousCode, newCode string) bool {
	deltaLen := len(newCode) - len(previousCode)
	if deltaLen >= d.config.PasteDeltaThreshold {
		return true
	}

	newLines := strings.Count(newCode, "\n") - strings.Count(previousCode, "\n")
	return newLines >= d.config.PasteLineThreshold
}

// determines if edits are significant enough to unlock
func (d *Detector) IsSignificantEdit(baseline, current string) bool {
	if baseline == "" {
		return true
	}

	distance := LevenshteinDistance(baseline, current)
	baselineLen := len(baseline)

	if baselineLen == 0 {
		return true
	}

	normalized := float64(distance) / float64(baselineLen)
	return normalized >= d.config.UnlockThreshold
}
