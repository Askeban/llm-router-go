# LLM Router v4 - Hybrid ML Classification System

A comprehensive microservices-based LLM routing system that uses hybrid ML classification to intelligently recommend the best model for any given prompt.

## 🚀 Key Features

- **🤖 Hybrid ML Classification**: Combines rule-based patterns with sentence transformer ML models
- **🐳 Docker Microservices**: Complete containerized architecture with Redis caching
- **📊 Smart Model Routing**: AI-powered model recommendation based on prompt analysis
- **⚡ High Performance**: Sub-30ms classification with intelligent fallback mechanisms
- **🔄 Real-time Health Monitoring**: Comprehensive service health dashboard

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────┐
│   User Request  │───▶│  Router (Go)     │───▶│  Classifier (Python)│
│                 │    │  Port: 8080      │    │  Port: 5001         │
└─────────────────┘    └──────────────────┘    └─────────────────────┘
                                │                          │
                                ▼                          ▼
                    ┌──────────────────┐         ┌─────────────────────┐
                    │ Model Rankings   │         │ Hybrid ML Engine    │
                    │ & Recommendations│         │ • Rule-based        │
                    └──────────────────┘         │ • Sentence Trans.   │
                                                 │ • all-MiniLM-L6-v2  │
                                                 └─────────────────────┘
```

## 🛠️ Services

| Service | Port | Description |
|---------|------|-------------|
| **Router** | 8080 | Main Go service for routing and recommendations |
| **Classifier** | 5001 | Hybrid Python ML classifier service |
| **Ingestor** | 8001 | Model data management service |
| **Redis** | 6379 | Caching layer for performance |
| **Dashboard** | 8090 | Health monitoring dashboard |

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- 4GB+ RAM (for ML models)

### 1. Start All Services
```bash
# Clone the repository
git clone <repository-url>
cd llm-router-go

# Start the entire system
docker-compose -f docker-compose.enhanced.yml up -d

# Check service health
docker-compose -f docker-compose.enhanced.yml ps
```

### 2. Verify System Health
```bash
# Check main router
curl http://localhost:8080/healthz
# Response: {"ok":true}

# Check ML classifier
curl http://localhost:5001/info  
# Response: {"ml_model_available":true, "ml_model":"all-MiniLM-L6-v2", ...}

# Check ingestor
curl http://localhost:8001/status
# Response: {"status":"operational","total_models":3, ...}

# Visit health dashboard
open http://localhost:8090
```

## 📡 API Reference

### 🎯 Main Router API (Port 8080)

#### Get Model Recommendation
```bash
POST /route
Content-Type: application/json

{
  "prompt": "Write a Python function to sort a list",
  "mode": "recommend"
}
```

**Response:**
```json
{
  "classification": {
    "category": "coding",
    "difficulty": "easy", 
    "ms": 24
  },
  "recommended_model": {
    "id": "gpt-3.5-turbo",
    "provider": "openai",
    "display_name": "GPT-3.5 Turbo"
  },
  "ranking": [...]
}
```

#### Generate with Best Model
```bash
POST /route  
Content-Type: application/json

{
  "prompt": "Explain quantum computing",
  "mode": "generate",
  "constraints": {"max_tokens": 500}
}
```

#### Get Available Models
```bash
GET /metrics/models
```

#### Get Model Metrics
```bash
GET /metrics/{model_id}
```

### 🧠 Classifier API (Port 5001)

#### Classify Prompt
```bash
POST /classify
Content-Type: application/json

{
  "prompt": "Create a beautiful poem about nature"
}
```

**Response:**
```json
{
  "primary_use_case": "creative_writing",
  "complexity_score": 0.15,
  "creativity_score": 0.9,
  "domain_confidence": 0.95,
  "classification_method": "hybrid",
  "ml_confidence": 0.87,
  "difficulty": "easy"
}
```

#### Get Classifier Info
```bash
GET /info
```

**Response:**
```json
{
  "service": "Hybrid LLM Router Classifier",
  "version": "2.0.0",
  "ml_model_available": true,
  "ml_model": "all-MiniLM-L6-v2",
  "categories": ["coding", "creative_writing", "analysis", "math", "question", "chat", "general"],
  "classification_methods": ["rule-based", "ml-based", "hybrid"]
}
```

### 📊 Ingestor API (Port 8001)

#### Get Service Status
```bash
GET /status
```

**Response:**
```json
{
  "status": "operational",
  "total_models": 3,
  "active_models": 3,
  "last_sync": "2025-09-07T17:44:53.609014",
  "data_sources": ["models.json", "default_data"]
}
```

#### Get All Models
```bash
GET /models
```

#### Get Specific Model
```bash
GET /models/{model_id}
```

#### Update Models (Admin)
```bash
POST /update
Content-Type: application/json

{
  "models": [
    {
      "id": "new-model",
      "provider": "openai", 
      "display_name": "New Model",
      "status": "active"
    }
  ]
}
```

## 🤖 ML Classification Details

### Classification Categories
- **coding**: Programming, debugging, technical implementation
- **creative_writing**: Stories, poems, creative content
- **analysis**: Research, comparison, evaluation
- **math**: Calculations, formulas, mathematical problems  
- **question**: Direct questions and queries
- **chat**: Conversational, casual communication
- **general**: General information and explanations

### Hybrid Classification Logic
1. **Rule-based Classification**: Keyword pattern matching with confidence scoring
2. **ML Classification**: Semantic similarity using sentence transformers  
3. **Intelligent Selection**: Chooses best approach based on confidence levels
4. **Graceful Fallback**: Falls back to rules if ML models fail

### Example Classification Flows

**High Rule Confidence (>0.7):**
```
"Write Python code" → Rule-based → coding (confidence: 0.95)
```

**High ML Confidence (>0.8):**  
```
"Implement neural networks" → ML-based → coding (confidence: 0.88)
```

**Agreement Between Both:**
```
"Create a story" → Hybrid → creative_writing (boosted confidence: 0.97)
```

## 🔧 Configuration

### Environment Variables
```bash
# Router
PORT=8080
DATABASE_PATH=/data/router.db
MODEL_PROFILES_PATH=/root/internal/models.json
CLASSIFIER_URL=http://classifier:5000

# Classifier  
REDIS_HOST=redis
REDIS_PORT=6379
CACHE_TTL=3600
LOG_LEVEL=INFO

# Ingestor
MODELS_JSON_PATH=/app/data/models.json
ANALYTICS_API_KEY=your-api-key
OUTPUT_PATH=/app/data/enhanced_models.json
```

### Performance Tuning
- **Redis TTL**: Adjust cache expiration (default: 3600s)
- **Batch Size**: Modify ML model batch processing  
- **Worker Count**: Scale classification workers
- **Model Loading**: Pre-load models for faster startup

## 🧪 Testing

### Run Health Checks
```bash
# Test all services
curl http://localhost:8080/healthz  # Router
curl http://localhost:5001/health   # Classifier  
curl http://localhost:8001/health   # Ingestor
curl http://localhost:6379          # Redis

# Test classification
curl -X POST http://localhost:5001/classify \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Help me debug this Python error"}'

# Test end-to-end routing
curl -X POST http://localhost:8080/route \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Solve this math equation: 2x + 5 = 15", "mode": "recommend"}'
```

### Load Testing
```bash
# Basic load test
for i in {1..10}; do
  curl -X POST http://localhost:8080/route \
    -H "Content-Type: application/json" \
    -d '{"prompt": "Test prompt '$i'", "mode": "recommend"}' &
done
```

## 🚦 Monitoring & Troubleshooting

### Health Dashboard
Visit http://localhost:8090 for:
- Service status overview
- Real-time health metrics
- Container resource usage
- Network connectivity status

### Log Analysis
```bash
# View service logs
docker-compose -f docker-compose.enhanced.yml logs router
docker-compose -f docker-compose.enhanced.yml logs classifier  
docker-compose -f docker-compose.enhanced.yml logs ingestor

# Follow live logs
docker-compose -f docker-compose.enhanced.yml logs -f classifier
```

### Common Issues

**ML Model Not Loading:**
```bash
# Check classifier logs
docker-compose logs classifier | grep "ML model"

# Expected: "✅ Successfully loaded LOCAL ML model: all-MiniLM-L6-v2"
# If failed: Model will fallback to rule-based only
```

**Port Conflicts:**
```bash
# Check if ports are in use
lsof -i :8080  # Router
lsof -i :5001  # Classifier  
lsof -i :8001  # Ingestor
```

**Performance Issues:**
```bash
# Check container resources
docker stats

# Scale specific services
docker-compose -f docker-compose.enhanced.yml up -d --scale classifier=2
```

## 🔄 Development

### Local Development Setup
```bash
# Install Go dependencies
go mod download

# Install Python dependencies
cd python/classifier_service && pip install -r requirements.txt
cd python/ingestor_service && pip install -r simple_requirements.txt

# Build router locally
go build -o router cmd/server/main.go
```

### Adding New Classification Categories
1. Update `REFERENCE_CATEGORIES` in `python/classifier_service/hybrid_app.py`
2. Add keyword patterns in `classify_with_rules()`
3. Rebuild classifier service
4. Test new category classification

### Model Updates
1. Update `internal/models.json` with new model definitions
2. Restart ingestor service to reload models
3. Verify through `/models` endpoint

## 📈 Performance Metrics

### Benchmark Results
- **Classification Latency**: 15-30ms (hybrid mode)
- **Fallback Latency**: 3-5ms (rule-only mode)  
- **Throughput**: 100+ requests/second
- **Memory Usage**: ~500MB (with ML models loaded)
- **Model Loading**: ~10 seconds startup time

### Scaling Guidelines
- **Single Instance**: Up to 100 RPS
- **Load Balanced**: 500+ RPS with multiple classifier instances
- **Enterprise**: 1000+ RPS with dedicated Redis cluster

## 🤝 Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)  
5. Open Pull Request

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- **Sentence Transformers** for ML-powered semantic classification
- **FastAPI** for high-performance Python services
- **Gin** for efficient Go HTTP routing
- **Docker** for containerized deployment
- **Redis** for high-speed caching

---

**Built with ❤️ for intelligent LLM routing**