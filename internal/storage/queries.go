package storage

const (
	getChunkCountQuery     = "SELECT COUNT(*) FROM doc_embeddings"
	deleteAllExamplesQuery = "DELETE FROM example_strudels"
	deleteAllChunksQuery   = "DELETE FROM doc_embeddings"
	getExampleCountQuery   = "SELECT COUNT(*) FROM example_strudels"

	insertChunkQuery = `
		INSERT INTO doc_embeddings (page_name, page_url, section_title, content, embedding, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	insertExampleQuery = `
		INSERT INTO example_strudels (title, description, code, tags, embedding, url)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
)
