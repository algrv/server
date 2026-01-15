package main

import (
	"context"
	"fmt"

	"codeberg.org/algorave/server/algorave/strudels"
	"codeberg.org/algorave/server/internal/ccsignals"
	"codeberg.org/algorave/server/internal/logger"
	"github.com/redis/go-redis/v9"
)

const (
	// minimum content length to index for fingerprint protection
	// prevents common short patterns from blocking unrelated users
	minContentLengthForProtection = 200

	// LSH configuration
	lshNumBands            = 4  // 4 bands of 16 bits each
	lshSimilarityThreshold = 10 // ~84% similarity required for match
	lshShingleSize         = 3  // 3-character shingles for fingerprinting
)

// holds all CC signals detection components
type CCSignalsSystem struct {
	Detector     *ccsignals.Detector
	Fingerprints *ccsignals.IndexedFingerprintStore
	LockStore    *ccsignals.RedisLockStore
}

// sets up the CC signals detection system
func InitializeCCSignals(
	ctx context.Context,
	redisClient *redis.Client,
	strudelRepo *strudels.Repository,
) (*CCSignalsSystem, error) {
	// create lock store using existing redis client
	lockStore := ccsignals.NewRedisLockStore(redisClient)

	// create content validator using strudels repository
	validator := ccsignals.NewStrudelValidator(strudelRepo)

	// create indexed fingerprint store with LSH (in-memory only, computed from strudels)
	indexedFpStore := ccsignals.NewInMemoryIndexedStore(
		lshNumBands,
		lshSimilarityThreshold,
		lshShingleSize,
	)

	// load existing no-ai strudels and compute fingerprints
	if err := loadNoAIFingerprints(ctx, indexedFpStore, strudelRepo); err != nil {
		logger.ErrorErr(err, "failed to load no-ai fingerprints, continuing without them")
		// don't fail startup - fingerprint protection is optional
	}

	// create detector with all components
	config := ccsignals.DefaultConfig()
	detector := ccsignals.NewDetector(config, lockStore, validator).
		WithFingerprints(indexedFpStore)

	logger.Info("CC signals system initialized",
		"fingerprints_loaded", indexedFpStore.Size(),
		"min_content_length", minContentLengthForProtection,
	)

	return &CCSignalsSystem{
		Detector:     detector,
		Fingerprints: indexedFpStore,
		LockStore:    lockStore,
	}, nil
}

// loadNoAIFingerprints loads no-ai strudels and computes fingerprints into the LSH index
func loadNoAIFingerprints(
	ctx context.Context,
	indexed *ccsignals.IndexedFingerprintStore,
	strudelRepo *strudels.Repository,
) error {
	noaiStrudels, err := strudelRepo.ListNoAIStrudels(ctx, minContentLengthForProtection)
	if err != nil {
		return fmt.Errorf("failed to load no-ai strudels: %w", err)
	}

	for _, s := range noaiStrudels {
		indexed.AddFromStrudel(s.ID, s.UserID, ccsignals.CCSignal(s.CCSignal), s.Code)
	}

	return nil
}

// adds a strudel to the fingerprint index if it has no-ai signal
// and meets minimum content length requirements
func (s *CCSignalsSystem) IndexStrudel(strudelID, creatorID, code string, ccSignal ccsignals.CCSignal) {
	// only index no-ai strudels
	if ccSignal != ccsignals.SignalNoAI {
		return
	}

	// only index substantial content
	if len(code) < minContentLengthForProtection {
		return
	}

	s.Fingerprints.AddFromStrudel(strudelID, creatorID, ccSignal, code)
	logger.Debug("indexed no-ai strudel", "strudel_id", strudelID, "content_length", len(code))
}

// RemoveStrudel removes a strudel from the fingerprint index
func (s *CCSignalsSystem) RemoveStrudel(strudelID string) {
	s.Fingerprints.Remove(strudelID)
}
