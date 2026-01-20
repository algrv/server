-- Create RAG tables for Algojams hybrid retrieval system
-- Generated: 2024-12-24

-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- ============================================================================
-- DOCUMENTATION CHUNKS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS doc_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_name TEXT NOT NULL,
    page_url TEXT NOT NULL,
    section_title TEXT,
    content TEXT NOT NULL,
    embedding extensions.vector(1536),
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

COMMENT ON TABLE doc_embeddings IS 'Stores chunked documentation with embeddings for semantic search';
COMMENT ON COLUMN doc_embeddings.section_title IS 'PAGE_SUMMARY for summary chunks, otherwise section name';

-- vector similarity search index
CREATE INDEX IF NOT EXISTS doc_embeddings_embedding_idx 
ON doc_embeddings 
USING ivfflat (embedding extensions.vector_cosine_ops)
WITH (lists = 100);

-- index on page_name for filtering and grouping
CREATE INDEX IF NOT EXISTS doc_embeddings_page_name_idx 
ON doc_embeddings(page_name);

-- index on created_at for maintenance queries
CREATE INDEX IF NOT EXISTS doc_embeddings_created_at_idx 
ON doc_embeddings(created_at DESC);

-- index on section_title for filtering PAGE_SUMMARY chunks
CREATE INDEX IF NOT EXISTS doc_embeddings_section_title_idx
ON doc_embeddings(section_title);

-- ============================================================================
-- EXAMPLE STRUDELS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS example_strudels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    description TEXT,
    code TEXT NOT NULL,
    tags TEXT[],
    embedding extensions.vector(1536),
    url TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

COMMENT ON TABLE example_strudels IS 'Stores example Strudel code with embeddings of descriptions';
COMMENT ON COLUMN example_strudels.embedding IS 'Embedding of title + description + tags (NOT code)';

-- vector similarity search index
CREATE INDEX IF NOT EXISTS example_strudels_embedding_idx 
ON example_strudels
USING ivfflat (embedding extensions.vector_cosine_ops)
WITH (lists = 100);

-- index on tags for filtering
CREATE INDEX IF NOT EXISTS example_strudels_tags_idx
ON example_strudels USING GIN(tags);

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- function to search doc_embeddings by similarity
CREATE OR REPLACE FUNCTION search_docs(
    query_embedding extensions.vector(1536),
    match_count int DEFAULT 5
)
RETURNS TABLE (
    id UUID,
    page_name TEXT,
    page_url TEXT,
    section_title TEXT,
    content TEXT,
    similarity FLOAT
)
LANGUAGE plpgsql STABLE
AS $$
BEGIN
    -- Set search path to include extensions schema where vector operators live
    PERFORM set_config('search_path', 'extensions, public', true);
    
    RETURN QUERY
    SELECT
        d.id,
        d.page_name,
        d.page_url,
        d.section_title,
        d.content,
        1 - (d.embedding <=> query_embedding) AS similarity
    FROM doc_embeddings d
    ORDER BY d.embedding <=> query_embedding
    LIMIT match_count;
END;
$$;

-- function to search example_strudels by similarity
CREATE OR REPLACE FUNCTION search_examples(
    query_embedding extensions.vector(1536),
    match_count int DEFAULT 3
)
RETURNS TABLE (
    id UUID,
    title TEXT,
    description TEXT,
    code TEXT,
    tags TEXT[],
    url TEXT,
    similarity FLOAT
)
LANGUAGE plpgsql STABLE
AS $$
BEGIN
    -- Set search path to include extensions schema where vector operators live
    PERFORM set_config('search_path', 'extensions, public', true);
    
    RETURN QUERY
    SELECT
        e.id,
        e.title,
        e.description,
        e.code,
        e.tags,
        e.url,
        1 - (e.embedding <=> query_embedding) AS similarity
    FROM example_strudels e
    ORDER BY e.embedding <=> query_embedding
    LIMIT match_count;
END;
$$;