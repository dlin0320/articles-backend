-- Database cleanup script for integration tests
-- This script truncates all tables in the correct order to avoid foreign key constraints

-- Clear all data in dependency order (only truncate tables that exist)
DO $$
BEGIN
    -- Truncate recommendations table if it exists
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'recommendations') THEN
        TRUNCATE TABLE recommendations CASCADE;
        RAISE NOTICE 'Truncated recommendations table';
    END IF;
    
    -- Truncate ratings table if it exists
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'ratings') THEN
        TRUNCATE TABLE ratings CASCADE;
        RAISE NOTICE 'Truncated ratings table';
    END IF;
    
    -- Truncate articles table if it exists
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'articles') THEN
        TRUNCATE TABLE articles CASCADE;
        RAISE NOTICE 'Truncated articles table';
    END IF;
    
    -- Truncate users table if it exists
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users') THEN
        TRUNCATE TABLE users CASCADE;
        RAISE NOTICE 'Truncated users table';
    END IF;
END $$;

-- Verify cleanup
DO $$
BEGIN
    RAISE NOTICE 'Cleanup completed. Table row counts:';
    
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users') THEN
        RAISE NOTICE 'users: % rows', (SELECT COUNT(*) FROM users);
    ELSE
        RAISE NOTICE 'users: table does not exist';
    END IF;
    
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'articles') THEN
        RAISE NOTICE 'articles: % rows', (SELECT COUNT(*) FROM articles);
    ELSE
        RAISE NOTICE 'articles: table does not exist';
    END IF;
    
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'ratings') THEN
        RAISE NOTICE 'ratings: % rows', (SELECT COUNT(*) FROM ratings);
    ELSE
        RAISE NOTICE 'ratings: table does not exist';
    END IF;
    
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'recommendations') THEN
        RAISE NOTICE 'recommendations: % rows', (SELECT COUNT(*) FROM recommendations);
    ELSE
        RAISE NOTICE 'recommendations: table does not exist';
    END IF;
END $$;