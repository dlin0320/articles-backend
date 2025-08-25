"""
Database configuration and connection management for the embedding service.
"""

import os
import logging
from sqlalchemy import create_engine, text
from sqlalchemy.orm import sessionmaker, Session
from sqlalchemy.pool import StaticPool
from contextlib import contextmanager
from typing import Generator

from models import Base

logger = logging.getLogger(__name__)

# Database configuration
DATABASE_URL = os.getenv('DATABASE_URL', 'postgresql://postgres:postgres@localhost:5432/articles')

# Create SQLAlchemy engine
engine = create_engine(
    DATABASE_URL,
    echo=os.getenv('SQLALCHEMY_ECHO', 'false').lower() == 'true',
    pool_pre_ping=True,  # Enable connection health checks
    pool_recycle=3600,   # Recycle connections every hour
)

# Create session factory
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)

def init_database():
    """Initialize database connection and verify pgvector extension"""
    try:
        # Test connection and verify pgvector extension
        with engine.connect() as conn:
            result = conn.execute(text("SELECT extname FROM pg_extension WHERE extname = 'vector'"))
            if result.fetchone() is None:
                logger.error("pgvector extension is not installed in the database")
                raise RuntimeError("pgvector extension is not available")
            else:
                logger.info("pgvector extension is available")
        
        # Create tables if they don't exist (should match GORM schema)
        logger.info("Database connection initialized successfully")
        return True
        
    except Exception as e:
        logger.error(f"Failed to initialize database: {e}")
        raise

@contextmanager
def get_db_session() -> Generator[Session, None, None]:
    """Get database session with automatic cleanup"""
    session = SessionLocal()
    try:
        yield session
        session.commit()
    except Exception as e:
        session.rollback()
        logger.error(f"Database session error: {e}")
        raise
    finally:
        session.close()

def get_session() -> Session:
    """Get database session (for dependency injection)"""
    return SessionLocal()

def health_check() -> bool:
    """Check database health"""
    try:
        with engine.connect() as conn:
            conn.execute(text("SELECT 1"))
        return True
    except Exception as e:
        logger.error(f"Database health check failed: {e}")
        return False