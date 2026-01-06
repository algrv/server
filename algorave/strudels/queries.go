package strudels

const (
	queryCreate = `
		INSERT INTO user_strudels (
			user_id, title, code, is_public, allow_training, ai_assist_count, forked_from, description, tags, categories, conversation_history
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, user_id, title, code, is_public, allow_training, use_in_training, ai_assist_count, forked_from, description, tags, categories, conversation_history, created_at, updated_at
	`

	queryGetPublic = `
		SELECT id, user_id, title, code, is_public, allow_training, use_in_training, ai_assist_count, forked_from, description, tags, categories, conversation_history, created_at, updated_at
		FROM user_strudels
		WHERE id = $1 AND is_public = true
	`

	queryGet = `
		SELECT id, user_id, title, code, is_public, allow_training, use_in_training, ai_assist_count, forked_from, description, tags, categories, conversation_history, created_at, updated_at
		FROM user_strudels
		WHERE id = $1 AND user_id = $2
	`

	queryUpdate = `
		UPDATE user_strudels
		SET title = COALESCE($1, title),
		    code = COALESCE($2, code),
		    is_public = COALESCE($3, is_public),
		    allow_training = COALESCE($4, allow_training),
		    ai_assist_count = COALESCE($5, ai_assist_count),
		    description = COALESCE($6, description),
		    tags = COALESCE($7, tags),
		    categories = COALESCE($8, categories),
		    conversation_history = COALESCE($9, conversation_history),
		    updated_at = NOW()
		WHERE id = $10 AND user_id = $11
		RETURNING id, user_id, title, code, is_public, allow_training, use_in_training, ai_assist_count, forked_from, description, tags, categories, conversation_history, created_at, updated_at
	`

	queryDelete = `
		DELETE FROM user_strudels
		WHERE id = $1 AND user_id = $2
	`

	queryListTrainableWithoutEmbedding = `
		SELECT us.id, us.user_id, us.title, us.code, us.is_public, us.allow_training, us.use_in_training, us.ai_assist_count, us.forked_from, us.description, us.tags, us.categories, us.conversation_history, us.created_at, us.updated_at
		FROM user_strudels us
		INNER JOIN users u ON us.user_id = u.id
		WHERE us.allow_training = true
		  AND us.use_in_training = true
		  AND us.is_public = true
		  AND us.embedding IS NULL
		  AND u.training_consent = true
		ORDER BY us.created_at DESC
		LIMIT $1
	`

	queryUpdateEmbedding = `
		UPDATE user_strudels
		SET embedding = $1
		WHERE id = $2
	`

	queryAdminSetUseInTraining = `
		UPDATE user_strudels
		SET use_in_training = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, user_id, title, code, is_public, allow_training, use_in_training, ai_assist_count, forked_from, description, tags, categories, conversation_history, created_at, updated_at
	`

	queryAdminGetStrudel = `
		SELECT id, user_id, title, code, is_public, allow_training, use_in_training, ai_assist_count, forked_from, description, tags, categories, conversation_history, created_at, updated_at
		FROM user_strudels
		WHERE id = $1
	`

	queryListPublicTags = `
		SELECT DISTINCT unnest(tags) as tag
		FROM user_strudels
		WHERE is_public = true AND array_length(tags, 1) > 0
		ORDER BY tag
	`

	queryListUserTags = `
		SELECT DISTINCT unnest(tags) as tag
		FROM user_strudels
		WHERE user_id = $1 AND array_length(tags, 1) > 0
		ORDER BY tag
	`

	// strudel_messages queries (AI conversation history for saved strudels)
	queryAddStrudelMessage = `
		INSERT INTO strudel_messages (strudel_id, user_id, role, content, is_actionable, is_code_response, clarifying_questions, display_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, strudel_id, user_id, role, content, is_actionable, is_code_response, clarifying_questions, display_name, created_at
	`

	queryGetStrudelMessages = `
		SELECT id, strudel_id, user_id, role, content, is_actionable, is_code_response, clarifying_questions, display_name, created_at
		FROM strudel_messages
		WHERE strudel_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
)
