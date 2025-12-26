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
			url,
			similarity
		FROM search_examples($1, $2)
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
			id,
			title,
			description,
			code,
			tags,
			url,
			ts_rank(searchable_tsvector, websearch_to_tsquery('english', $1)) as rank
		FROM example_strudels
		WHERE searchable_tsvector @@ websearch_to_tsquery('english', $1)
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
