package retriever

const (
	vectorSearchQuery = `
		SELECT
			id::text,
			page_name,
			page_url,
			section_title,
			content,
			similarity
		FROM search_docs($1, $2)
	`

	searchExamplesQuery = `
		SELECT
			id::text,
			title,
			description,
			code,
			tags,
			user_id::text,
			'' as url,
			similarity
		FROM search_user_strudels($1, $2)
	`

	bm25SearchDocsQuery = `
		SELECT
			id::text,
			page_name,
			page_url,
			section_title,
			content,
			ts_rank(content_tsvector, websearch_to_tsquery('english', $1)) as rank
		FROM doc_embeddings
		WHERE content_tsvector @@ websearch_to_tsquery('english', $1)
		ORDER BY rank DESC
		LIMIT $2
	`

	bm25SearchExamplesQuery = `
		SELECT
			us.id::text,
			us.user_id::text,
			us.title,
			us.description,
			us.code,
			us.tags,
			'' as url,
			ts_rank(us.searchable_tsvector, websearch_to_tsquery('english', $1)) as rank
		FROM user_strudels us
		INNER JOIN users u ON us.user_id = u.id
		WHERE us.searchable_tsvector @@ websearch_to_tsquery('english', $1)
		  AND us.cc_signal IS NOT NULL
		  AND us.cc_signal != 'no-ai'
		  AND us.use_in_training = true
		  AND us.is_public = true
		  AND u.training_consent = true
		ORDER BY rank DESC
		LIMIT $2
	`

	fetchSpecialChunkQuery = `
		SELECT
			id::text,
			page_name,
			page_url,
			section_title,
			content,
			0.0 as similarity
		FROM doc_embeddings
		WHERE page_name = $1 AND section_title = $2
		LIMIT 1
	`
)
