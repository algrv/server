package users

const (
	queryFindOrCreateByProvider = `
		INSERT INTO users (provider, provider_id, email, name, avatar_url)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (provider, provider_id)
		DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			avatar_url = EXCLUDED.avatar_url,
			updated_at = NOW()
		RETURNING id, email, provider, provider_id, name, avatar_url, tier, is_admin, training_consent, ai_features_enabled, created_at, updated_at
	`

	queryFindByID = `
		SELECT id, email, provider, provider_id, name, avatar_url, tier, is_admin, training_consent, ai_features_enabled, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	queryUpdateProfile = `
		UPDATE users
		SET name = $1, avatar_url = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, email, provider, provider_id, name, avatar_url, tier, is_admin, training_consent, ai_features_enabled, created_at, updated_at
	`

	queryUpdateTrainingConsent = `
		UPDATE users
		SET training_consent = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, email, provider, provider_id, name, avatar_url, tier, is_admin, training_consent, ai_features_enabled, created_at, updated_at
	`

	queryUpdateAIFeaturesEnabled = `
		UPDATE users
		SET ai_features_enabled = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, email, provider, provider_id, name, avatar_url, tier, is_admin, training_consent, ai_features_enabled, created_at, updated_at
	`

	queryGetUserDailyUsage = `
		SELECT get_user_usage_today($1)
	`

	queryGetSessionDailyUsage = `
		SELECT get_session_usage_today($1)
	`

	queryLogUsage = `
		INSERT INTO usage_logs (user_id, session_id, provider, model, input_tokens, output_tokens, is_byok)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
)
