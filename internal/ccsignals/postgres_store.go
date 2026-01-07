package ccsignals

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	createTableSQL = `
		CREATE TABLE IF NOT EXISTS cc_fingerprints (
			id TEXT PRIMARY KEY,
			fingerprint BIGINT NOT NULL,
			work_id TEXT NOT NULL UNIQUE,
			creator_id TEXT NOT NULL,
			cc_signal TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_cc_fingerprints_work_id ON cc_fingerprints(work_id);
		CREATE INDEX IF NOT EXISTS idx_cc_fingerprints_creator_id ON cc_fingerprints(creator_id);
	`

	insertSQL = `
		INSERT INTO cc_fingerprints (id, fingerprint, work_id, creator_id, cc_signal)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			fingerprint = EXCLUDED.fingerprint,
			cc_signal = EXCLUDED.cc_signal
	`

	deleteSQL = `DELETE FROM cc_fingerprints WHERE id = $1`

	loadAllSQL = `
		SELECT id, fingerprint, work_id, creator_id, cc_signal
		FROM cc_fingerprints
	`

	getByWorkIDSQL = `
		SELECT id, fingerprint, work_id, creator_id, cc_signal
		FROM cc_fingerprints
		WHERE work_id = $1
	`
)

// implements FingerprintStore using PostgreSQL
type PostgresFingerprintStore struct {
	db *pgxpool.Pool
}

// creates a new PostgreSQL fingerprint store
func NewPostgresFingerprintStore(db *pgxpool.Pool) *PostgresFingerprintStore {
	return &PostgresFingerprintStore{db: db}
}

// creates the required tables if they don't exist
func (s *PostgresFingerprintStore) Initialize(ctx context.Context) error {
	_, err := s.db.Exec(ctx, createTableSQL)
	return err
}

// saves a fingerprint record
func (s *PostgresFingerprintStore) Store(ctx context.Context, record *FingerprintRecord) error {
	var ccSignal *string
	if record.CCSignal != "" {
		str := string(record.CCSignal)
		ccSignal = &str
	}

	_, err := s.db.Exec(ctx, insertSQL,
		record.ID,
		int64(record.Fingerprint), //nolint:gosec // fingerprint is 64-bit, same width as int64
		record.WorkID,
		record.CreatorID,
		ccSignal,
	)
	return err
}

// removes a fingerprint record
func (s *PostgresFingerprintStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, deleteSQL, id)
	return err
}

// loads all fingerprint records
func (s *PostgresFingerprintStore) LoadAll(ctx context.Context) ([]*FingerprintRecord, error) {
	rows, err := s.db.Query(ctx, loadAllSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to load fingerprints: %w", err)
	}

	defer rows.Close()
	var records []*FingerprintRecord

	for rows.Next() {
		var record FingerprintRecord
		var fp int64
		var ccSignal *string

		err := rows.Scan(&record.ID, &fp, &record.WorkID, &record.CreatorID, &ccSignal)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fingerprint: %w", err)
		}

		record.Fingerprint = Fingerprint(fp) //nolint:gosec // int64 and uint64 have same width

		if ccSignal != nil {
			record.CCSignal = CCSignal(*ccSignal)
		}

		records = append(records, &record)
	}

	return records, rows.Err()
}

// retrieves fingerprint by work ID
func (s *PostgresFingerprintStore) GetByWorkID(ctx context.Context, workID string) (*FingerprintRecord, error) {
	var record FingerprintRecord
	var fp int64
	var ccSignal *string

	err := s.db.QueryRow(ctx, getByWorkIDSQL, workID).Scan(
		&record.ID, &fp, &record.WorkID, &record.CreatorID, &ccSignal,
	)
	if err != nil {
		return nil, nil // not found
	}

	record.Fingerprint = Fingerprint(fp) //nolint:gosec // int64 and uint64 have same width
	if ccSignal != nil {
		record.CCSignal = CCSignal(*ccSignal)
	}

	return &record, nil
}
