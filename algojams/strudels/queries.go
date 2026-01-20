package strudels

const (
	queryCreate = `
		INSERT INTO user_strudels (
			user_id, title, code, is_public, license, cc_signal, ai_assist_count, forked_from, description, tags, categories, conversation_history
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, user_id, title, code, is_public, license, cc_signal, use_in_training, ai_assist_count, forked_from, description, tags, categories, conversation_history, created_at, updated_at
	`

	queryGetPublic = `
		SELECT s.id, s.user_id, u.name, s.title, s.code, s.is_public, s.license, s.cc_signal, s.use_in_training, s.ai_assist_count, s.forked_from, s.description, s.tags, s.categories, s.conversation_history, s.created_at, s.updated_at
		FROM user_strudels s
		LEFT JOIN users u ON s.user_id = u.id
		WHERE s.id = $1 AND s.is_public = true
	`

	queryGet = `
		SELECT id, user_id, title, code, is_public, license, cc_signal, use_in_training, ai_assist_count, forked_from, description, tags, categories, conversation_history, created_at, updated_at
		FROM user_strudels
		WHERE id = $1 AND user_id = $2
	`

	queryUpdate = `
		UPDATE user_strudels
		SET title = COALESCE($1, title),
		    code = COALESCE($2, code),
		    is_public = COALESCE($3, is_public),
		    license = COALESCE($4, license),
		    cc_signal = COALESCE($5, cc_signal),
		    ai_assist_count = COALESCE($6, ai_assist_count),
		    description = COALESCE($7, description),
		    tags = COALESCE($8, tags),
		    categories = COALESCE($9, categories),
		    conversation_history = COALESCE($10, conversation_history),
		    updated_at = NOW()
		WHERE id = $11 AND user_id = $12
		RETURNING id, user_id, title, code, is_public, license, cc_signal, use_in_training, ai_assist_count, forked_from, description, tags, categories, conversation_history, created_at, updated_at
	`

	queryDelete = `
		DELETE FROM user_strudels
		WHERE id = $1 AND user_id = $2
	`

	queryListTrainableWithoutEmbedding = `
		SELECT us.id, us.user_id, us.title, us.code, us.is_public, us.license, us.cc_signal, us.use_in_training, us.ai_assist_count, us.forked_from, us.description, us.tags, us.categories, us.conversation_history, us.created_at, us.updated_at
		FROM user_strudels us
		INNER JOIN users u ON us.user_id = u.id
		WHERE us.cc_signal IS NOT NULL
		  AND us.cc_signal != 'no-ai'
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
		RETURNING id, user_id, title, code, is_public, license, cc_signal, use_in_training, ai_assist_count, forked_from, description, tags, categories, conversation_history, created_at, updated_at
	`

	queryAdminGetStrudel = `
		SELECT id, user_id, title, code, is_public, license, cc_signal, use_in_training, ai_assist_count, forked_from, description, tags, categories, conversation_history, created_at, updated_at
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

	queryGetParentCCSignal = `
		SELECT cc_signal FROM user_strudels WHERE id = $1
	`

	queryGetStrudelForkedFrom = `
		SELECT forked_from FROM user_strudels WHERE id = $1
	`

	// paste detection: check if code matches any strudel owned by user
	queryUserOwnsStrudelWithCode = `
		SELECT EXISTS(
			SELECT 1 FROM user_strudels
			WHERE user_id = $1 AND code = $2
		)
	`

	// paste detection: check if code matches any public strudel that allows AI (for fork validation)
	// requires explicit permissive signal - NULL/missing defaults to no-ai (restrictive)
	queryPublicStrudelExistsWithCodeAllowsAI = `
		SELECT EXISTS(
			SELECT 1 FROM user_strudels
			WHERE is_public = true
			  AND code = $1
			  AND cc_signal IN ('cc-cr', 'cc-dc', 'cc-ec', 'cc-op')
		)
	`

	// paste detection: check if code matches any public strudel with no-ai CC signal
	queryPublicStrudelExistsWithCodeNoAI = `
		SELECT EXISTS(
			SELECT 1 FROM user_strudels
			WHERE is_public = true
			  AND code = $1
			  AND cc_signal = 'no-ai'
		)
	`

	// strudel_messages queries (AI conversation history for saved strudels)
	queryAddStrudelMessage = `
		INSERT INTO strudel_messages (strudel_id, user_id, role, content, is_actionable, is_code_response, clarifying_questions, strudel_references, doc_references, display_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, strudel_id, user_id, role, content, is_actionable, is_code_response, clarifying_questions, strudel_references, doc_references, display_name, created_at
	`

	queryGetStrudelMessages = `
		SELECT id, strudel_id, user_id, role, content, is_actionable, is_code_response, clarifying_questions, strudel_references, doc_references, display_name, created_at
		FROM strudel_messages
		WHERE strudel_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	// fingerprint protection: get all no-ai strudels with sufficient content
	// used at startup to populate the LSH index for similarity detection
	queryListNoAIStrudels = `
		SELECT id, user_id, code, cc_signal
		FROM user_strudels
		WHERE cc_signal = 'no-ai'
		  AND LENGTH(code) >= $1
	`
)
