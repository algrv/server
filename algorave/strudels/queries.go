package strudels

const (
	queryCreate = `
		INSERT INTO user_strudels (
			user_id, title, code, is_public, description, tags, categories, conversation_history
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, title, code, is_public, use_in_training, description, tags, categories, conversation_history, created_at, updated_at
	`

	queryList = `
		SELECT id, user_id, title, code, is_public, use_in_training, description, tags, categories, conversation_history, created_at, updated_at
		FROM user_strudels
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	queryListPublic = `
		SELECT id, user_id, title, code, is_public, use_in_training, description, tags, categories, conversation_history, created_at, updated_at
		FROM user_strudels
		WHERE is_public = true
		ORDER BY created_at DESC
		LIMIT $1
	`

	queryGet = `
		SELECT id, user_id, title, code, is_public, use_in_training, description, tags, categories, conversation_history, created_at, updated_at
		FROM user_strudels
		WHERE id = $1 AND user_id = $2
	`

	queryUpdate = `
		UPDATE user_strudels
		SET title = COALESCE($1, title),
		    code = COALESCE($2, code),
		    is_public = COALESCE($3, is_public),
		    description = COALESCE($4, description),
		    tags = COALESCE($5, tags),
		    categories = COALESCE($6, categories),
		    conversation_history = COALESCE($7, conversation_history),
		    updated_at = NOW()
		WHERE id = $8 AND user_id = $9
		RETURNING id, user_id, title, code, is_public, use_in_training, description, tags, categories, conversation_history, created_at, updated_at
	`

	queryDelete = `
		DELETE FROM user_strudels
		WHERE id = $1 AND user_id = $2
	`

	queryListTrainableWithoutEmbedding = `
		SELECT us.id, us.user_id, us.title, us.code, us.is_public, us.use_in_training, us.description, us.tags, us.categories, us.conversation_history, us.created_at, us.updated_at
		FROM user_strudels us
		INNER JOIN users u ON us.user_id = u.id
		WHERE us.use_in_training = true
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
		RETURNING id, user_id, title, code, is_public, use_in_training, description, tags, categories, conversation_history, created_at, updated_at
	`

	queryAdminGetStrudel = `
		SELECT id, user_id, title, code, is_public, use_in_training, description, tags, categories, conversation_history, created_at, updated_at
		FROM user_strudels
		WHERE id = $1
	`
)
