-- Create vector indexes for optimal similarity search performance
-- This should be run after GORM has created the tables

-- Create IVFFlat index for embedding similarity search
-- IVFFlat is good for datasets with 1K+ vectors
-- The lists parameter should be roughly sqrt(total_rows)
CREATE INDEX CONCURRENTLY IF NOT EXISTS articles_embedding_cosine_idx 
ON articles 
USING ivfflat (embedding vector_cosine_ops) 
WITH (lists = 100);

-- Create index for embedding status to quickly find articles with embeddings
CREATE INDEX CONCURRENTLY IF NOT EXISTS articles_embedding_status_idx 
ON articles (embedding_status)
WHERE embedding_status = 'success';

-- Create composite index for user filtering + embedding status
CREATE INDEX CONCURRENTLY IF NOT EXISTS articles_user_embedding_idx 
ON articles (user_id, embedding_status)
WHERE embedding_status = 'success';

-- Create index for metadata status filtering
CREATE INDEX CONCURRENTLY IF NOT EXISTS articles_metadata_status_idx 
ON articles (metadata_status)
WHERE metadata_status = 'success';

-- Analyze table to update statistics
ANALYZE articles;