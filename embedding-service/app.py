#!/usr/bin/env python3
"""
Multilingual Sentence Embedding Microservice
Uses all-MiniLM-L6-v2 model for generating embeddings
"""

from flask import Flask, request, jsonify
from sentence_transformers import SentenceTransformer
from transformers import pipeline
import numpy as np
import logging
import os
import re
from uuid import UUID
from sqlalchemy.exc import SQLAlchemyError

from database import init_database, get_db_session, health_check as db_health_check
from models import Article

app = Flask(__name__)
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Load the multilingual sentence transformer model
MODEL_NAME = "all-MiniLM-L6-v2"
CLASSIFIER_MODEL_NAME = "distilbert-base-uncased-finetuned-sst-2-english"
model = None
classifier = None

def load_model():
    """Load the sentence transformer model and classifier"""
    global model, classifier
    try:
        logger.info(f"Loading embedding model: {MODEL_NAME}")
        model = SentenceTransformer(MODEL_NAME)
        logger.info("Embedding model loaded successfully")
        
        logger.info(f"Loading classifier model: {CLASSIFIER_MODEL_NAME}")
        classifier = pipeline("text-classification", 
                            model=CLASSIFIER_MODEL_NAME,
                            return_all_scores=True)
        logger.info("Classifier model loaded successfully")
    except Exception as e:
        logger.error(f"Failed to load models: {e}")
        raise

def initialize():
    """Initialize the model and database"""
    load_model()
    try:
        init_database()
        logger.info("Database initialized successfully")
    except Exception as e:
        logger.error(f"Failed to initialize database: {e}")
        raise

# Initialize on startup
with app.app_context():
    initialize()

@app.route('/health', methods=['GET'])
def health_check():
    """Health check endpoint"""
    db_healthy = db_health_check()
    return jsonify({
        "status": "healthy" if db_healthy else "unhealthy",
        "embedding_model": MODEL_NAME,
        "classifier_model": CLASSIFIER_MODEL_NAME,
        "embedding_model_loaded": model is not None,
        "classifier_loaded": classifier is not None,
        "database_healthy": db_healthy
    })

@app.route('/embed', methods=['POST'])
def generate_embedding():
    """Generate embedding for given text"""
    try:
        data = request.get_json()
        
        if not data or 'text' not in data:
            return jsonify({"error": "Missing 'text' field in request"}), 400
        
        text = data['text']
        if not text or not text.strip():
            return jsonify({"error": "Empty text provided"}), 400
        
        # Generate embedding
        logger.info(f"Generating embedding for text: {text[:50]}...")
        embedding = model.encode([text.strip()])[0]
        
        # Convert to list for JSON serialization
        embedding_list = embedding.tolist()
        
        return jsonify({
            "text": text,
            "embedding": embedding_list,
            "dimension": len(embedding_list)
        })
        
    except Exception as e:
        logger.error(f"Error generating embedding: {e}")
        return jsonify({"error": "Internal server error"}), 500

@app.route('/embed/batch', methods=['POST'])
def generate_batch_embeddings():
    """Generate embeddings for multiple texts"""
    try:
        data = request.get_json()
        
        if not data or 'texts' not in data:
            return jsonify({"error": "Missing 'texts' field in request"}), 400
        
        texts = data['texts']
        if not isinstance(texts, list) or not texts:
            return jsonify({"error": "Empty or invalid texts list"}), 400
        
        # Filter empty texts
        clean_texts = [text.strip() for text in texts if text and text.strip()]
        if not clean_texts:
            return jsonify({"error": "No valid texts provided"}), 400
        
        logger.info(f"Generating embeddings for {len(clean_texts)} texts")
        
        # Generate embeddings in batch (more efficient)
        embeddings = model.encode(clean_texts)
        
        # Convert to list for JSON serialization
        embeddings_list = [emb.tolist() for emb in embeddings]
        
        return jsonify({
            "texts": clean_texts,
            "embeddings": embeddings_list,
            "count": len(embeddings_list),
            "dimension": len(embeddings_list[0]) if embeddings_list else 0
        })
        
    except Exception as e:
        logger.error(f"Error generating batch embeddings: {e}")
        return jsonify({"error": "Internal server error"}), 500

@app.route('/similarity', methods=['POST'])
def calculate_similarity():
    """Calculate cosine similarity between two embeddings"""
    try:
        data = request.get_json()
        
        if not data or 'embedding1' not in data or 'embedding2' not in data:
            return jsonify({"error": "Missing embedding1 or embedding2 fields"}), 400
        
        emb1 = np.array(data['embedding1'])
        emb2 = np.array(data['embedding2'])
        
        if emb1.shape != emb2.shape:
            return jsonify({"error": "Embedding dimensions don't match"}), 400
        
        # Calculate cosine similarity
        dot_product = np.dot(emb1, emb2)
        norm1 = np.linalg.norm(emb1)
        norm2 = np.linalg.norm(emb2)
        
        if norm1 == 0 or norm2 == 0:
            similarity = 0.0
        else:
            similarity = dot_product / (norm1 * norm2)
        
        return jsonify({
            "similarity": float(similarity)
        })
        
    except Exception as e:
        logger.error(f"Error calculating similarity: {e}")
        return jsonify({"error": "Internal server error"}), 500

@app.route('/classify', methods=['POST'])
def classify_content():
    """Classify if content is article-worthy using ML model"""
    try:
        data = request.get_json()
        
        if not data or 'text' not in data:
            return jsonify({"error": "Missing 'text' field in request"}), 400
        
        text = data['text']
        if not text or not text.strip():
            return jsonify({"error": "Empty text provided"}), 400
        
        # Clean and prepare text for classification
        cleaned_text = clean_text_for_classification(text)
        
        logger.info(f"Classifying content quality for text: {cleaned_text[:50]}...")
        
        # Use the classifier to determine content quality
        # We'll use sentiment analysis as a proxy for content quality
        # Positive sentiment often correlates with well-written article content
        result = classifier(cleaned_text)
        
        # Calculate article-worthiness based on multiple factors
        article_score = calculate_article_score(text, result)
        is_article = article_score > 0.5
        
        return jsonify({
            "text": text[:100] + "..." if len(text) > 100 else text,
            "is_article": is_article,
            "confidence": float(article_score),
            "classification_details": {
                "sentiment_scores": result,
                "text_length": len(text),
                "cleaned_text_length": len(cleaned_text),
                "word_count": len(cleaned_text.split())
            }
        })
        
    except Exception as e:
        logger.error(f"Error classifying content: {e}")
        return jsonify({"error": "Internal server error"}), 500

@app.route('/classify/batch', methods=['POST'])
def classify_batch_content():
    """Classify multiple texts for article-worthiness"""
    try:
        data = request.get_json()
        
        if not data or 'texts' not in data:
            return jsonify({"error": "Missing 'texts' field in request"}), 400
        
        texts = data['texts']
        if not isinstance(texts, list) or not texts:
            return jsonify({"error": "Empty or invalid texts list"}), 400
        
        # Filter and clean texts
        clean_texts = []
        original_indices = []
        for i, text in enumerate(texts):
            if text and text.strip():
                clean_texts.append(clean_text_for_classification(text.strip()))
                original_indices.append(i)
        
        if not clean_texts:
            return jsonify({"error": "No valid texts provided"}), 400
        
        logger.info(f"Batch classifying {len(clean_texts)} texts")
        
        # Classify all texts
        results = []
        for i, cleaned_text in enumerate(clean_texts):
            original_text = texts[original_indices[i]]
            classification_result = classifier(cleaned_text)
            article_score = calculate_article_score(original_text, classification_result)
            
            results.append({
                "text": original_text[:100] + "..." if len(original_text) > 100 else original_text,
                "is_article": article_score > 0.5,
                "confidence": float(article_score),
                "index": original_indices[i]
            })
        
        return jsonify({
            "results": results,
            "count": len(results),
            "processed": len(clean_texts)
        })
        
    except Exception as e:
        logger.error(f"Error in batch classification: {e}")
        return jsonify({"error": "Internal server error"}), 500

def clean_text_for_classification(text):
    """Clean text for better classification results"""
    # Remove excessive whitespace and newlines
    text = re.sub(r'\s+', ' ', text.strip())
    
    # Remove URLs
    text = re.sub(r'http[s]?://(?:[a-zA-Z]|[0-9]|[$-_@.&+]|[!*\\(\\),]|(?:%[0-9a-fA-F][0-9a-fA-F]))+', '', text)
    
    # Remove email addresses
    text = re.sub(r'\S+@\S+', '', text)
    
    # Truncate very long text for the classifier (BERT has token limits)
    if len(text) > 500:
        # Take first 400 chars and last 100 chars to preserve context
        text = text[:400] + " " + text[-100:]
    
    return text.strip()

def calculate_article_score(original_text, sentiment_result):
    """Calculate article-worthiness score based on multiple factors"""
    base_score = 0.0
    
    # Factor 1: Text length (longer text is more likely to be an article)
    text_length = len(original_text.strip())
    if text_length > 1000:
        base_score += 0.3
    elif text_length > 500:
        base_score += 0.2
    elif text_length > 200:
        base_score += 0.1
    elif text_length < 50:
        base_score -= 0.2  # Very short text is less likely to be an article
    
    # Factor 2: Word count
    word_count = len(original_text.split())
    if word_count > 200:
        base_score += 0.2
    elif word_count > 100:
        base_score += 0.1
    elif word_count < 20:
        base_score -= 0.1
    
    # Factor 3: Sentiment analysis results (well-written content tends to be more neutral/positive)
    if sentiment_result and len(sentiment_result) > 0:
        # Get the confidence of the positive sentiment
        for result in sentiment_result[0]:  # sentiment_result is a list of lists
            if result['label'] == 'POSITIVE':
                # Higher positive sentiment confidence suggests better written content
                positive_confidence = result['score']
                base_score += (positive_confidence - 0.5) * 0.3  # Scale between -0.15 to +0.15
            elif result['label'] == 'NEGATIVE':
                # Very negative sentiment might indicate poor quality
                negative_confidence = result['score']
                if negative_confidence > 0.8:
                    base_score -= 0.1
    
    # Factor 4: Basic structure indicators
    structure_indicators = ['.', '!', '?', ':', ';', ',']
    structure_count = sum(original_text.count(indicator) for indicator in structure_indicators)
    if structure_count > 10:
        base_score += 0.1
    elif structure_count > 5:
        base_score += 0.05
    
    # Ensure score is between 0 and 1
    return max(0.0, min(1.0, base_score + 0.5))  # Add 0.5 as base to be more generous

@app.route('/articles/<article_id>/embedding', methods=['POST'])
def generate_and_store_embedding(article_id):
    """Generate embedding for an article and store it in the database"""
    try:
        # Validate UUID
        try:
            article_uuid = UUID(article_id)
        except ValueError:
            return jsonify({"error": "Invalid article ID format"}), 400
        
        data = request.get_json()
        if not data or 'text' not in data:
            return jsonify({"error": "Missing 'text' field in request"}), 400
        
        text = data.get('text', '').strip()
        if not text:
            return jsonify({"error": "Empty text provided"}), 400
        
        logger.info(f"Generating and storing embedding for article {article_id}")
        
        # Generate embedding
        embedding = model.encode([text])[0]
        embedding_list = embedding.tolist()
        
        # Store in database
        with get_db_session() as session:
            # Find the article
            article = session.query(Article).filter(Article.id == article_uuid).first()
            if not article:
                return jsonify({"error": "Article not found"}), 404
            
            # Update embedding fields
            article.embedding = embedding_list
            article.embedding_status = 'success'
            
            session.commit()
            
        logger.info(f"Successfully stored embedding for article {article_id}")
        
        return jsonify({
            "article_id": article_id,
            "embedding_dimension": len(embedding_list),
            "embedding_status": "success",
            "message": "Embedding generated and stored successfully"
        })
        
    except SQLAlchemyError as e:
        logger.error(f"Database error generating embedding for article {article_id}: {e}")
        return jsonify({"error": "Database error"}), 500
    except Exception as e:
        logger.error(f"Error generating embedding for article {article_id}: {e}")
        return jsonify({"error": "Internal server error"}), 500

@app.route('/articles/batch/embedding', methods=['POST'])
def generate_batch_embeddings_and_store():
    """Generate embeddings for multiple articles and store them in the database"""
    try:
        data = request.get_json()
        if not data or 'articles' not in data:
            return jsonify({"error": "Missing 'articles' field in request"}), 400
        
        articles_data = data['articles']
        if not isinstance(articles_data, list) or not articles_data:
            return jsonify({"error": "Empty or invalid articles list"}), 400
        
        logger.info(f"Processing batch embedding for {len(articles_data)} articles")
        
        results = []
        successful_updates = 0
        
        with get_db_session() as session:
            for article_data in articles_data:
                try:
                    article_id = article_data.get('id')
                    text = article_data.get('text', '').strip()
                    
                    if not article_id or not text:
                        results.append({
                            "article_id": article_id,
                            "status": "error",
                            "message": "Missing article ID or text"
                        })
                        continue
                    
                    # Validate UUID
                    try:
                        article_uuid = UUID(article_id)
                    except ValueError:
                        results.append({
                            "article_id": article_id,
                            "status": "error", 
                            "message": "Invalid article ID format"
                        })
                        continue
                    
                    # Generate embedding
                    embedding = model.encode([text])[0]
                    embedding_list = embedding.tolist()
                    
                    # Find and update article
                    article = session.query(Article).filter(Article.id == article_uuid).first()
                    if not article:
                        results.append({
                            "article_id": article_id,
                            "status": "error",
                            "message": "Article not found"
                        })
                        continue
                    
                    # Update embedding fields
                    article.embedding = embedding_list
                    article.embedding_status = 'success'
                    
                    successful_updates += 1
                    results.append({
                        "article_id": article_id,
                        "status": "success",
                        "embedding_dimension": len(embedding_list)
                    })
                    
                except Exception as e:
                    logger.error(f"Error processing article {article_id}: {e}")
                    results.append({
                        "article_id": article_data.get('id', 'unknown'),
                        "status": "error",
                        "message": str(e)
                    })
            
            # Commit all successful updates
            session.commit()
        
        logger.info(f"Batch embedding completed: {successful_updates}/{len(articles_data)} successful")
        
        return jsonify({
            "total_articles": len(articles_data),
            "successful_updates": successful_updates,
            "results": results
        })
        
    except SQLAlchemyError as e:
        logger.error(f"Database error in batch embedding: {e}")
        return jsonify({"error": "Database error"}), 500
    except Exception as e:
        logger.error(f"Error in batch embedding: {e}")
        return jsonify({"error": "Internal server error"}), 500

if __name__ == '__main__':
    port = int(os.getenv('PORT', 8001))
    debug = os.getenv('DEBUG', 'false').lower() == 'true'
    
    logger.info(f"Starting embedding service on port {port}")
    app.run(host='0.0.0.0', port=port, debug=debug)