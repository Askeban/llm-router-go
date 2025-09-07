FROM python:3.9-slim

WORKDIR /app

# Install system dependencies and SSL certificates
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    wget \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates \
    && c_rehash

# Copy requirements first for better caching
COPY requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Download required NLTK data and models (if needed)
RUN python -c "import nltk; nltk.download('punkt')" || true

# Set SSL environment variables and pre-download model
ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt
ENV SSL_CERT_DIR=/etc/ssl/certs
ENV REQUESTS_CA_BUNDLE=/etc/ssl/certs/ca-certificates.crt
ENV CURL_CA_BUNDLE=/etc/ssl/certs/ca-certificates.crt

# Pre-download sentence transformer model with comprehensive SSL handling
RUN export CURL_CA_BUNDLE="" && export REQUESTS_CA_BUNDLE="" && export SSL_VERIFY=false && \
    python -c "import ssl, os; ssl._create_default_https_context = ssl._create_unverified_context; os.environ['CURL_CA_BUNDLE']=''; os.environ['REQUESTS_CA_BUNDLE']=''; from sentence_transformers import SentenceTransformer; SentenceTransformer('all-MiniLM-L6-v2', trust_remote_code=True)" || echo "Model download failed, will use fallback"

# Copy application code
COPY . .

# Create directories for cache
RUN mkdir -p /app/semantic_cache /app/rules_cache /app/feedback_cache

# Environment variables
ENV PYTHONPATH=/app
ENV PYTHONUNBUFFERED=1
ENV PORT=5000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
  CMD curl -f http://localhost:5000/health || exit 1

# Expose port
EXPOSE 5000

# Command to run the application
CMD ["python", "hybrid_app.py"]