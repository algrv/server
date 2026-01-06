package attribution

import "time"

type Attribution struct {
	ID                    string
	SourceStrudelID       string
	SourceStrudelTitle    string
	RequestingUserID      *string
	RequestingDisplayName *string
	SimilarityScore       *float32
	CreatedAt             time.Time
}

type AttributionStats struct {
	TotalUses      int
	UniqueStrudels int
	LastUsedAt     *time.Time
}

// per-strudel stats
type StrudelStats struct {
	TotalUses   int        `json:"total_uses"`
	UniqueUsers int        `json:"unique_users"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
}

type StrudelUse struct {
	ID                    string     `json:"id"`
	TargetStrudelID       *string    `json:"target_strudel_id,omitempty"`
	TargetStrudelTitle    *string    `json:"target_strudel_title,omitempty"`
	RequestingUserID      *string    `json:"requesting_user_id,omitempty"`
	RequestingDisplayName *string    `json:"requesting_display_name,omitempty"`
	SimilarityScore       *float32   `json:"similarity_score,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
}

type StrudelStatsResponse struct {
	Stats      StrudelStats `json:"stats"`
	RecentUses []StrudelUse `json:"recent_uses"`
}
