FROM python:3.12-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements first for better caching
COPY requirements.txt .

# Install Python dependencies
RUN pip install --extra-index-url https://download.pytorch.org/whl/cpu -r requirements.txt

# Copy application code
COPY *.py .

# Download model during build (optional - can also download on first run)
RUN python -c "from sentence_transformers import SentenceTransformer; SentenceTransformer('all-MiniLM-L6-v2')"

# Expose port
EXPOSE 8001

# Health check disabled for debugging
# HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 \
#     CMD python -c "import requests; requests.get('http://localhost:8001/health')"

# Run the application
CMD ["python", "app.py"]