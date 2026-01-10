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
			u.name as requesting_display_name,
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
	// combines RAG attribution stats with fork stats for comprehensive usage data
	queryGetStrudelStats = `
		WITH rag_stats AS (
			SELECT
				COUNT(*) as total_uses,
				COUNT(DISTINCT ra.requesting_user_id) as unique_users,
				MAX(ra.created_at) as last_used_at
			FROM rag_attributions ra
			WHERE ra.source_strudel_id = $1
		),
		fork_stats AS (
			SELECT
				COUNT(DISTINCT us.user_id) as unique_forkers,
				MAX(us.created_at) as last_forked_at
			FROM user_strudels us
			WHERE us.forked_from = $1
		)
		SELECT
			COALESCE(r.total_uses, 0) as total_uses,
			COALESCE(r.unique_users, 0) + COALESCE(f.unique_forkers, 0) as unique_users,
			GREATEST(r.last_used_at, f.last_forked_at) as last_used_at
		FROM rag_stats r, fork_stats f
	`

	// combines RAG attributions and forks into "inspired" list, only public strudels shown
	queryGetStrudelRecentUses = `
		WITH combined AS (
			-- RAG attributions
			SELECT
				ra.id::text,
				ra.target_strudel_id,
				target.title as target_strudel_title,
				ra.requesting_user_id,
				u.name as requesting_display_name,
				ra.similarity_score,
				ra.created_at,
				target.is_public
			FROM rag_attributions ra
			LEFT JOIN user_strudels target ON ra.target_strudel_id = target.id
			LEFT JOIN users u ON ra.requesting_user_id = u.id
			WHERE ra.source_strudel_id = $1
				AND ra.target_strudel_id IS NOT NULL

			UNION ALL

			-- Forks
			SELECT
				us.id::text,
				us.id as target_strudel_id,
				us.title as target_strudel_title,
				us.user_id as requesting_user_id,
				u.name as requesting_display_name,
				NULL::float as similarity_score,
				us.created_at,
				us.is_public
			FROM user_strudels us
			LEFT JOIN users u ON us.user_id = u.id
			WHERE us.forked_from = $1
		)
		SELECT DISTINCT ON (target_strudel_id)
			id,
			target_strudel_id,
			target_strudel_title,
			requesting_user_id,
			requesting_display_name,
			similarity_score,
			created_at
		FROM combined
		WHERE is_public = true
		ORDER BY target_strudel_id, created_at DESC
		LIMIT $2
	`

	queryGetStrudelForkCount = `
		SELECT COUNT(*) FROM user_strudels WHERE forked_from = $1
	`
)
