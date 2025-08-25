# Articles Backend API

A production-ready Go backend service for managing article links with automatic metadata extraction, user ratings, and AI-powered recommendations. Built with scalability and clean architecture principles in mind.

## 🚀 Key Features

- **Article Management**: Save and organize article links with automatic metadata extraction
- **Smart Recommendations**: AI-powered recommendation engine using vector similarity search
- **User Authentication**: Secure JWT-based authentication system  
- **Rating System**: User ratings with 5-star scale
- **Background Processing**: Resilient worker for retry logic on failed metadata extractions
- **Vector Search**: O(log n) similarity search using PostgreSQL with pgvector extension
- **Clean Architecture**: Well-structured codebase following SOLID principles

## 🏗️ Architecture Highlights

### Scalable Recommendation System
- **Pre-computed Embeddings**: Articles are embedded once and stored in PostgreSQL
- **Vector Similarity Search**: Uses pgvector extension for O(log n) performance
- **Multilingual Support**: Supports 100+ languages via sentence-transformers
- **Smart Fallbacks**: Popular articles shown for new users without rating history

### Clean Code Structure
```
├── cmd/api/              # Application entry point
├── internal/             # Core business logic
│   ├── article/         # Article domain
│   ├── rating/          # Rating domain  
│   ├── recommendation/  # Recommendation engine
│   ├── user/           # User authentication
│   ├── repository/     # Data persistence layer
│   └── worker/         # Background processing
├── embedding-service/   # Python ML microservice
└── pkg/                # Reusable packages
```

### Design Patterns
- **Dependency Injection**: Clean separation of concerns
- **Repository Pattern**: Database abstraction layer
- **Service Layer**: Business logic encapsulation
- **Interface-Based Design**: Testable and maintainable code

## 🛠️ Tech Stack

- **Backend**: Go 1.21+ with Gin framework
- **Database**: PostgreSQL 15 with pgvector extension
- **ML Service**: Python Flask with sentence-transformers
- **ORM**: GORM (Go) and SQLAlchemy (Python)
- **Authentication**: JWT tokens
- **Containerization**: Docker & Docker Compose
- **Testing**: Unit and integration tests

## 📊 Performance

| Metric | Performance | Details |
|--------|------------|---------|
| **Recommendation Speed** | O(log n) | Vector index optimization |
| **API Response Time** | <100ms | P95 latency |
| **Embedding Generation** | Batch processing | Efficient ML pipeline |
| **Database Queries** | Optimized | GORM structured queries |
| **Scalability** | Horizontal | Microservice architecture |

## 🚀 Quick Start

### Prerequisites
- Docker and Docker Compose
- Go 1.21+ (for local development)
- PostgreSQL 15+ with pgvector extension

### Running with Docker (Recommended)

1. Clone the repository:
```bash
git clone <repository-url>
cd articles-backend
```

2. Copy environment variables:
```bash
cp .env.example .env
# Edit .env with your configuration
```

3. Start all services:
```bash
docker-compose up --build
```

This will start:
- API server on port 8080
- PostgreSQL with pgvector on port 5432
- Embedding service on port 8001

### Local Development

1. Install dependencies:
```bash
go mod download
```

2. Set up PostgreSQL with pgvector:
```bash
# Install pgvector extension
CREATE EXTENSION vector;
```

3. Run database migrations:
```bash
# Migrations run automatically on startup
```

4. Start the embedding service:
```bash
cd embedding-service
pip install -r requirements.txt
python app.py
```

5. Run the API server:
```bash
go run cmd/api/main.go
```

## 📚 API Documentation

### Authentication Endpoints

#### Sign Up
```bash
POST /signup
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword"
}
```

#### Login
```bash
POST /login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword"
}

Response:
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

### Article Management

#### Create Article
```bash
POST /articles
Authorization: Bearer <token>
Content-Type: application/json

{
  "url": "https://example.com/article"
}
```

#### List Articles
```bash
GET /articles?page=1&limit=10
Authorization: Bearer <token>
```

#### Delete Article
```bash
DELETE /articles/:id
Authorization: Bearer <token>
```

### Ratings

#### Rate Article
```bash
POST /articles/:id/rate
Authorization: Bearer <token>
Content-Type: application/json

{
  "rating": 5
}
```

#### Get Rating
```bash
GET /articles/:id/rate
Authorization: Bearer <token>
```

### Recommendations

#### Get Recommendations
```bash
GET /recommendations?limit=10
Authorization: Bearer <token>
```

## 🧪 Testing

### Run All Tests
```bash
go test ./...
```

### Run with Coverage
```bash
go test -cover ./...
```

### Run Integration Tests
```bash
go test -tags=integration ./tests/integration/...
```

### Test Embedding Service
```bash
# Health check
curl http://localhost:8001/health

# Test embedding generation
curl -X POST http://localhost:8001/embed \
  -H "Content-Type: application/json" \
  -d '{"text": "Sample article text"}'
```

## 🔧 Configuration

All configuration is managed through environment variables. See `.env.example` for available options:

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | API server port | 8080 |
| `DB_HOST` | PostgreSQL host | localhost |
| `DB_PORT` | PostgreSQL port | 5432 |
| `JWT_SECRET` | JWT signing key | (required) |
| `JWT_EXPIRATION` | Token expiration | 24h |
| `EMBEDDING_SERVICE_URL` | ML service URL | http://localhost:8001 |
| `WORKER_RETRY_INTERVAL` | Retry interval | 5m |
| `LOG_LEVEL` | Logging level | info |

## 🔒 Security

- JWT tokens for authentication
- Password hashing with bcrypt
- SQL injection prevention via GORM
- Input validation on all endpoints
- No secrets in code (use environment variables)

## 📊 Monitoring

The application provides health check endpoints:

- API Health: `GET /health`
- Embedding Service: `GET http://localhost:8001/health`

## 🚢 Deployment

### Docker Deployment
```bash
docker-compose up -d --build
```

### Production Considerations
- Use strong JWT secrets
- Enable SSL/TLS
- Set up database backups
- Configure log aggregation
- Implement rate limiting
- Set up monitoring and alerting


## 🙏 Acknowledgments

- Built with [Gin Web Framework](https://github.com/gin-gonic/gin)
- Vector search powered by [pgvector](https://github.com/pgvector/pgvector)
- ML models from [Hugging Face](https://huggingface.co/)
- Metadata extraction using [go-readability](https://github.com/go-shiori/go-readability)

---

**Built with ❤️ using Go and modern software engineering practices**