# Embedding Microservice

A Python Flask microservice that provides multilingual sentence embeddings using the `all-MiniLM-L6-v2` model from sentence-transformers.

## Features

- **Multilingual Support**: Works with 100+ languages
- **Fast Inference**: Optimized for quick embedding generation
- **Batch Processing**: Efficient batch embedding generation
- **ML-Based Classification**: Content quality assessment using DistilBERT
- **Health Checks**: Built-in health monitoring
- **Docker Ready**: Containerized for easy deployment

## Quick Start

### Local Development

```bash
# Install dependencies
pip install -r requirements.txt

# Start the service
python app.py
```

### Docker Deployment

```bash
# Build and run with docker-compose
docker-compose up --build

# Or build and run manually
docker build -t embedding-service .
docker run -p 8001:8001 embedding-service
```

## API Endpoints

### Health Check
```bash
GET /health
```

### Single Text Embedding
```bash
POST /embed
Content-Type: application/json

{
  "text": "This is a sample text to embed"
}
```

### Batch Text Embeddings
```bash
POST /embed/batch
Content-Type: application/json

{
  "texts": [
    "First text to embed",
    "Second text to embed", 
    "Third text to embed"
  ]
}
```

### Similarity Calculation
```bash
POST /similarity
Content-Type: application/json

{
  "embedding1": [0.1, 0.2, 0.3, ...],
  "embedding2": [0.4, 0.5, 0.6, ...]
}
```

### Content Classification
```bash
POST /classify
Content-Type: application/json

{
  "text": "This is article content to classify..."
}
```

### Generate and Store Article Embedding
```bash
POST /articles/{article_id}/embedding
Content-Type: application/json

{
  "text": "Article title and description to embed..."
}
```

### Batch Generate and Store Embeddings
```bash
POST /articles/batch/embedding
Content-Type: application/json

{
  "articles": [
    {
      "id": "uuid-1",
      "text": "First article content..."
    },
    {
      "id": "uuid-2", 
      "text": "Second article content..."
    }
  ]
}
```

### Batch Content Classification
```bash
POST /classify/batch
Content-Type: application/json

{
  "texts": [
    "First article content",
    "Second article content",
    "Third article content"
  ]
}
```

## Integration with Go Backend

The Go backend uses this service through the `embedding.Client`:

```go
client := embedding.NewClient("http://localhost:8001")

// Generate embeddings
embeddings, err := client.GetEmbedding("Article title and description")

// Classify content quality
classification, err := client.ClassifyContent("Article title and content...")
```

## Model Information

### Embedding Model
- **Model**: `all-MiniLM-L6-v2`
- **Dimension**: 384
- **Languages**: 100+ (including English, Spanish, French, German, Chinese, etc.)
- **Size**: ~90MB
- **Performance**: Fast inference, good semantic understanding

### Classification Model
- **Model**: `distilbert-base-uncased-finetuned-sst-2-english`
- **Type**: Sentiment analysis (used as content quality proxy)
- **Size**: ~67MB
- **Performance**: Lightweight, fast classification
- **Method**: Combines sentiment analysis with length/structure heuristics

## Environment Variables

- `PORT`: Service port (default: 8001)
- `DEBUG`: Enable debug mode (default: false)

## Health Monitoring

The service includes health checks for Docker and Kubernetes deployments:
- `/health` endpoint for application health
- Docker healthcheck in Dockerfile
- Kubernetes-ready health probes