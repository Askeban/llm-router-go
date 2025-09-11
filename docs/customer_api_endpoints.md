# Customer API Endpoints Design

## 🎯 API Versioning Strategy
- Base URL: `https://api.llmrouter.ai/v1/`
- Version in path: `/v1/`, `/v2/` (future)
- Backward compatibility maintained

## 🔐 Authentication Required Endpoints

### Core Recommendation API

#### `POST /v1/recommend`
**Get model recommendations without generation**
```bash
curl -X POST https://api.llmrouter.ai/v1/recommend \
  -H "Authorization: Bearer sk_live_abc123..." \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Write a Python function to sort a list",
    "constraints": {
      "max_cost_per_1k": 0.01,
      "max_latency_ms": 3000,
      "prefer_open_source": false
    }
  }'
```

**Response:**
```json
{
  "request_id": "req_1234567890",
  "classification": {
    "category": "coding",
    "difficulty": "easy", 
    "confidence": 0.95,
    "processing_time_ms": 25
  },
  "recommended_model": {
    "id": "openrouter-deepseek-coder-v2",
    "provider": "openrouter",
    "display_name": "DeepSeek Coder V2",
    "reasoning": "Specialized for coding with 86% coding capability and cost-efficient at $0.0006/1K tokens",
    "estimated_cost": {
      "input_tokens": 12,
      "output_tokens": 150,
      "total_cost_usd": 0.000237
    },
    "capabilities": {
      "coding": 0.86,
      "reasoning": 0.78,
      "math": 0.75
    }
  },
  "alternatives": [
    {
      "id": "anthropic-claude-3.5-sonnet",
      "display_name": "Claude 3.5 Sonnet",
      "score": 0.82,
      "reasoning": "Excellent for complex coding with superior reasoning"
    }
  ],
  "usage": {
    "requests_remaining_hour": 847,
    "requests_remaining_day": 9153  
  }
}
```

#### `POST /v1/generate`
**Get model recommendation + actual generation**
```bash
curl -X POST https://api.llmrouter.ai/v1/generate \
  -H "Authorization: Bearer sk_live_abc123..." \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Write a Python function to sort a list",
    "constraints": {
      "max_cost_per_1k": 0.01
    }
  }'
```

**Response:**
```json
{
  "request_id": "req_1234567891",
  "classification": {
    "category": "coding",
    "difficulty": "easy"
  },
  "model_used": {
    "id": "openrouter-deepseek-coder-v2",
    "display_name": "DeepSeek Coder V2"
  },
  "generated_text": "def sort_list(arr):\n    \"\"\"Sort a list in ascending order.\"\"\"\n    return sorted(arr)\n\n# Example usage:\nmy_list = [64, 34, 25, 12, 22, 11, 90]\nsorted_list = sort_list(my_list)\nprint(sorted_list)  # Output: [11, 12, 22, 25, 34, 64, 90]",
  "usage": {
    "prompt_tokens": 12,
    "completion_tokens": 67,
    "total_tokens": 79,
    "cost_usd": 0.000142
  },
  "performance": {
    "generation_time_ms": 1847,
    "classification_time_ms": 23
  }
}
```

### Analytics & Insights APIs

#### `GET /v1/models`
**List available models with capabilities**
```json
{
  "models": [
    {
      "id": "openrouter-deepseek-coder-v2",
      "provider": "openrouter",
      "display_name": "DeepSeek Coder V2",
      "capabilities": {
        "coding": 0.86,
        "math": 0.75,
        "reasoning": 0.78
      },
      "pricing": {
        "input_per_1k": 0.0006,
        "output_per_1k": 0.0015
      },
      "context_window": 128000,
      "tags": ["coding", "cheap"]
    }
  ],
  "total": 53
}
```

#### `GET /v1/analytics/usage`
**Get usage analytics for current user**
```json
{
  "period": "last_30_days",
  "total_requests": 15847,
  "by_category": {
    "coding": 8234,
    "math": 2156,
    "creative_writing": 3201,
    "reasoning": 2256
  },
  "by_model": {
    "openrouter-deepseek-coder-v2": 8234,
    "anthropic-claude-3.5-sonnet": 4521,
    "google-gemini-1.5-pro": 3092
  },
  "cost_breakdown": {
    "total_cost_usd": 12.47,
    "avg_cost_per_request": 0.000786
  },
  "performance": {
    "avg_response_time_ms": 1856,
    "success_rate": 0.998
  }
}
```

## 🏠 Dashboard/Management API (No Auth Required)

### Authentication Endpoints

#### `POST /auth/register`
**User registration**
```json
{
  "email": "developer@company.com",
  "password": "securePassword123!",
  "company_name": "Acme Corp",
  "first_name": "John",
  "last_name": "Doe"
}
```

#### `POST /auth/login`
**User login** 
```json
{
  "email": "developer@company.com", 
  "password": "securePassword123!"
}
```

**Response:**
```json
{
  "access_token": "eyJ0eXAiOiJKV1Q...",
  "refresh_token": "eyJ0eXAiOiJKV1Q...",
  "expires_in": 3600,
  "user": {
    "id": 123,
    "email": "developer@company.com",
    "plan": "free"
  }
}
```

#### `POST /auth/refresh`
**Refresh access token**

### API Key Management (Requires Dashboard Auth)

#### `GET /dashboard/api-keys`
**List user's API keys**
```json
{
  "api_keys": [
    {
      "id": 456,
      "name": "Production",
      "prefix": "sk_live_12345678",
      "created_at": "2025-01-15T10:30:00Z",
      "last_used_at": "2025-01-15T14:22:00Z",
      "is_active": true
    }
  ]
}
```

#### `POST /dashboard/api-keys`
**Create new API key**
```json
{
  "name": "Development Environment",
  "environment": "test"
}
```

**Response:**
```json
{
  "api_key": "sk_test_abcdef1234567890abcdef1234567890abcdef12",
  "prefix": "sk_test_abcdef12",
  "name": "Development Environment",
  "created_at": "2025-01-15T15:30:00Z",
  "warning": "This key will only be shown once. Store it securely."
}
```

## 🚨 Error Responses

### Rate Limiting (429)
```json
{
  "error": {
    "code": "rate_limit_exceeded",
    "message": "Hourly rate limit exceeded. Limit: 1000 requests/hour",
    "details": {
      "limit_type": "hourly",
      "limit": 1000,
      "used": 1000,
      "reset_at": "2025-01-15T16:00:00Z"
    }
  },
  "request_id": "req_1234567892"
}
```

### Authentication Error (401)
```json
{
  "error": {
    "code": "invalid_api_key",
    "message": "The provided API key is invalid or inactive"
  }
}
```

### Usage Quota Exceeded (403)
```json
{
  "error": {
    "code": "quota_exceeded", 
    "message": "Monthly quota exceeded. Please upgrade your plan or wait until next month",
    "details": {
      "plan": "free",
      "quota": 10000,
      "used": 10000,
      "reset_date": "2025-02-01"
    }
  }
}
```

## 📊 Response Headers

All authenticated requests include usage headers:
```
X-RateLimit-Limit-Hour: 1000
X-RateLimit-Remaining-Hour: 847
X-RateLimit-Reset-Hour: 1694793600
X-Request-ID: req_1234567890
X-Processing-Time-MS: 25
```

## 🎯 Implementation Notes

### Security
- All API keys validated via SHA-256 hash lookup
- Request logging includes only key prefix, never full key
- IP-based rate limiting to prevent abuse
- CORS configured for dashboard domain only

### Performance  
- Redis for rate limiting counters (fast lookups)
- Database connection pooling
- Async processing where possible
- Response caching for model metadata

### Monitoring
- All requests logged with timing, status, errors
- Usage analytics updated in real-time
- Alert thresholds for unusual patterns
- Health checks on all dependencies