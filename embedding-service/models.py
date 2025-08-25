"""
SQLAlchemy models for the embedding service.
These models should match the GORM models in the Go service.
"""

from sqlalchemy import Column, String, Text, Integer, DateTime, Float
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.sql import func
from pgvector.sqlalchemy import Vector
import uuid

Base = declarative_base()

class Article(Base):
    """Article model matching the Go GORM Article struct"""
    __tablename__ = 'articles'

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    user_id = Column(UUID(as_uuid=True), nullable=False, index=True)
    url = Column(String(2048), nullable=False)
    title = Column(String(500))
    description = Column(Text)
    image_url = Column(String(2048))
    content = Column(Text)
    word_count = Column(Integer, default=0)
    metadata_status = Column(String(20), default='pending', index=True)
    retry_count = Column(Integer, default=0)
    confidence_score = Column(Float, default=0.0)
    classifier_used = Column(String(50))
    
    # Vector embedding fields
    embedding = Column(Vector(384), index=True)  # 384-dimensional vector for all-MiniLM-L6-v2
    embedding_status = Column(String(20), default='pending')
    
    # Timestamps
    created_at = Column(DateTime(timezone=True), server_default=func.now(), index=True)
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

    def __repr__(self):
        return f"<Article(id={self.id}, title='{self.title[:50]}...', embedding_status='{self.embedding_status}')>"

class User(Base):
    """User model for reference (forward declaration match)"""
    __tablename__ = 'users'
    
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    email = Column(String(255), unique=True, nullable=False)
    
class Rating(Base):
    """Rating model for reference (forward declaration match)"""
    __tablename__ = 'ratings'
    
    user_id = Column(UUID(as_uuid=True), primary_key=True)
    article_id = Column(UUID(as_uuid=True), primary_key=True)
    score = Column(Integer, nullable=False)  # 1-5 rating
    created_at = Column(DateTime(timezone=True), server_default=func.now())
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())