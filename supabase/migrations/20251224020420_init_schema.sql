-- enable pgvector extension
-- Note: On Supabase hosted, enable this extension via Dashboard -> Database -> Extensions first
CREATE EXTENSION IF NOT EXISTS vector WITH SCHEMA extensions;

-- create doc_embeddings table
CREATE TABLE IF NOT EXISTS doc_embeddings 
    (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_name TEXT NOT NULL,
    page_url TEXT NOT NULL,
    section_title TEXT,
    content TEXT NOT NULL,
    embedding extensions.vector(1536),
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

-- create vector similarity search index
CREATE INDEX IF NOT EXISTS doc_embeddings_embedding_idx 
ON doc_embeddings 
USING ivfflat (embedding extensions.vector_cosine_ops)
WITH (lists = 100);

-- create index on page_name for faster lookups
CREATE INDEX IF NOT EXISTS doc_embeddings_page_name_idx 
ON doc_embeddings(page_name);

-- create index on created_at for maintenance queries
CREATE INDEX IF NOT EXISTS doc_embeddings_created_at_idx 
ON doc_embeddings(created_at DESC);

-- verify the table was created
SELECT 
    tablename, 
    schemaname 
FROM pg_tables 
WHERE tablename = 'doc_embeddings';

-- check if vector extension is enabled
SELECT * FROM pg_extension WHERE extname = 'vector';