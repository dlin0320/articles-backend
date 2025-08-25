-- Initialize database with pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Create index on vector column for performance (will be applied after GORM migration)
-- This will be executed when the embedding column is added
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS articles_embedding_idx ON articles USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);