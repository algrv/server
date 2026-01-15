package attribution

import (
	"context"

	"codeberg.org/algorave/server/internal/logger"
	"codeberg.org/algorave/server/internal/retriever"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

// records that examples were used as RAG context
// runs asynchronously to not block the agent response
func (s *Service) RecordAttributions(
	_ context.Context,
	examples []retriever.ExampleResult,
	requestingUserID string,
	targetStrudelID *string,
) {
	go func() {
		for _, ex := range examples {
			if ex.UserID == "" || ex.ID == "" {
				continue
			}

			// don't record self-attribution
			if ex.UserID == requestingUserID {
				continue
			}

			_, err := s.db.Exec(
				context.Background(),
				queryRecordAttribution,
				ex.ID,
				targetStrudelID,
				requestingUserID,
				ex.Similarity,
			)

			if err != nil {
				logger.Warn("failed to record attribution", "error", err, "source_strudel_id", ex.ID)
			}
		}
	}()
}

// gets attribution stats for a user's strudels
func (s *Service) GetUserAttributionStats(ctx context.Context, userID string) (*AttributionStats, error) {
	var stats AttributionStats

	err := s.db.QueryRow(ctx, queryGetUserAttributionStats, userID).Scan(
		&stats.TotalUses,
		&stats.UniqueStrudels,
		&stats.LastUsedAt,
	)

	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// gets recent attributions for a user (their strudels being used)
func (s *Service) GetRecentAttributions(ctx context.Context, userID string, limit int) ([]Attribution, error) {
	rows, err := s.db.Query(ctx, queryGetRecentAttributions, userID, limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var attributions []Attribution

	for rows.Next() {
		var a Attribution
		err := rows.Scan(
			&a.ID,
			&a.SourceStrudelID,
			&a.SourceStrudelTitle,
			&a.RequestingUserID,
			&a.RequestingDisplayName,
			&a.SimilarityScore,
			&a.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		attributions = append(attributions, a)
	}

	return attributions, rows.Err()
}

// gets stats for a specific strudel
func (s *Service) GetStrudelStats(ctx context.Context, strudelID string) (*StrudelStats, error) {
	var stats StrudelStats

	err := s.db.QueryRow(ctx, queryGetStrudelStats, strudelID).Scan(
		&stats.TotalUses,
		&stats.UniqueUsers,
		&stats.LastUsedAt,
	)

	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// gets recent uses of a specific strudel (unique target strudels)
func (s *Service) GetStrudelRecentUses(ctx context.Context, strudelID string, limit int) ([]StrudelUse, error) {
	rows, err := s.db.Query(ctx, queryGetStrudelRecentUses, strudelID, limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var uses []StrudelUse

	for rows.Next() {
		var u StrudelUse
		err := rows.Scan(
			&u.ID,
			&u.TargetStrudelID,
			&u.TargetStrudelTitle,
			&u.RequestingUserID,
			&u.RequestingDisplayName,
			&u.SimilarityScore,
			&u.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		uses = append(uses, u)
	}

	return uses, rows.Err()
}

// gets fork count for a strudel
func (s *Service) GetStrudelForkCount(ctx context.Context, strudelID string) (int, error) {
	var count int
	err := s.db.QueryRow(ctx, queryGetStrudelForkCount, strudelID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// gets full stats response for a strudel
func (s *Service) GetStrudelStatsResponse(ctx context.Context, strudelID string) (*StrudelStatsResponse, error) {
	stats, err := s.GetStrudelStats(ctx, strudelID)
	if err != nil {
		return nil, err
	}

	// get fork count
	forkCount, err := s.GetStrudelForkCount(ctx, strudelID)
	if err != nil {
		return nil, err
	}
	stats.ForkCount = forkCount

	uses, err := s.GetStrudelRecentUses(ctx, strudelID, 5)
	if err != nil {
		return nil, err
	}

	if uses == nil {
		uses = []StrudelUse{}
	}

	return &StrudelStatsResponse{
		Stats:      *stats,
		RecentUses: uses,
	}, nil
}
