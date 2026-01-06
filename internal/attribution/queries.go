package attribution

const (
	queryRecordAttribution = `
		INSERT INTO rag_attributions (source_strudel_id, target_strudel_id, requesting_user_id, similarity_score)
		VALUES ($1, $2, $3, $4)
	`

	queryGetUserAttributionStats = `
		SELECT
			COUNT(*) as total_uses,
			COUNT(DISTINCT ra.source_strudel_id) as unique_strudels,
			MAX(ra.created_at) as last_used_at
		FROM rag_attributions ra
		INNER JOIN user_strudels us ON ra.source_strudel_id = us.id
		WHERE us.user_id = $1
	`

	queryGetRecentAttributions = `
		SELECT
			ra.id,
			ra.source_strudel_id,
			us.title as source_strudel_title,
			ra.requesting_user_id,
			u.display_name as requesting_display_name,
			ra.similarity_score,
			ra.created_at
		FROM rag_attributions ra
		INNER JOIN user_strudels us ON ra.source_strudel_id = us.id
		LEFT JOIN users u ON ra.requesting_user_id = u.id
		WHERE us.user_id = $1
		ORDER BY ra.created_at DESC
		LIMIT $2
	`

	// per-strudel queries
	queryGetStrudelStats = `
		SELECT
			COUNT(*) as total_uses,
			COUNT(DISTINCT ra.requesting_user_id) as unique_users,
			MAX(ra.created_at) as last_used_at
		FROM rag_attributions ra
		WHERE ra.source_strudel_id = $1
	`

	queryGetStrudelRecentUses = `
		SELECT DISTINCT ON (target.id)
			ra.id,
			ra.target_strudel_id,
			target.title as target_strudel_title,
			ra.requesting_user_id,
			u.display_name as requesting_display_name,
			ra.similarity_score,
			ra.created_at
		FROM rag_attributions ra
		LEFT JOIN user_strudels target ON ra.target_strudel_id = target.id
		LEFT JOIN users u ON ra.requesting_user_id = u.id
		WHERE ra.source_strudel_id = $1
			AND ra.target_strudel_id IS NOT NULL
			AND target.is_public = true
		ORDER BY target.id, ra.created_at DESC
		LIMIT $2
	`
)
