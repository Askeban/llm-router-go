# 🚀 LLM Router Complete User Flow Testing Guide

This guide provides step-by-step instructions to test the complete user journey from frontend registration through API key generation to getting model recommendations.

## 📋 Prerequisites

1. **LLM Router Server Running** with authentication enabled
2. **Enhanced API Server Running** (fallback)
3. **curl** or **Postman** for API testing
4. **jq** for JSON parsing (optional but recommended)

## 🛠️ Server Setup

### 1. Start Authentication Server (Port 8080)
```bash
# In terminal 1
cd /Users/Sauransh.Singh/Downloads/llm-router-go
export ANALYTICS_API_KEY="aa_hvPVoBMuwefckQlniBWCrpQUmPdNSift"
go run . # or your preferred server binary
```

### 2. Start Enhanced Server (Port 8083) - Fallback
```bash
# In terminal 2
cd /Users/Sauransh.Singh/Downloads/llm-router-go
export ANALYTICS_API_KEY="aa_hvPVoBMuwefckQlniBWCrpQUmPdNSift"
go run cmd/enhanced-server/main.go
```

## 🧪 Complete User Flow Testing

### Step 1: User Registration 👤

**Endpoint:** `POST http://localhost:8080/auth/register`

**Command:**
```bash
curl -X POST "http://localhost:8080/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "testpassword123",
    "name": "Test User"
  }' | jq .
```

**Expected Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "user_id": 123,
  "email": "test@example.com",
  "name": "Test User"
}
```

**❗ Save the `access_token` - you'll need it for the next steps!**

---

### Step 2: API Key Generation 🔑

**Endpoint:** `POST http://localhost:8080/dashboard/api-keys`

**Command:**
```bash
# Replace YOUR_ACCESS_TOKEN with the token from Step 1
export USER_TOKEN="YOUR_ACCESS_TOKEN"

curl -X POST "http://localhost:8080/dashboard/api-keys" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -d '{
    "name": "Test API Key",
    "description": "API key for testing user flow"
  }' | jq .
```

**Expected Response:**
```json
{
  "api_key": "sk_live_...",
  "key_id": "ak_...",
  "name": "Test API Key",
  "description": "API key for testing user flow",
  "created_at": "2025-09-11T10:00:00Z",
  "permissions": ["read", "recommend"]
}
```

**❗ Save the `api_key` - you'll need it for model recommendations!**

---

### Step 3: Model Recommendations Using API Key 🤖

**Endpoint:** `POST http://localhost:8080/v1/recommend`

**Command:**
```bash
# Replace YOUR_API_KEY with the API key from Step 2
export API_KEY="YOUR_API_KEY"

curl -X POST "http://localhost:8080/v1/recommend" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_KEY" \
  -d '{
    "prompt": "Write a Python function to calculate fibonacci numbers with optimizations",
    "budget_priority": "cost_optimized",
    "max_recommendations": 5,
    "constraints": {
      "max_cost_per_request": 0.01,
      "response_time_priority": "fast"
    }
  }' | jq .
```

**Expected Response:**
```json
{
  "recommendations": [
    {
      "model_name": "claude-3-haiku-20240307",
      "provider": "anthropic",
      "estimated_cost": 0.00025,
      "performance_score": 0.92,
      "reasoning": "Excellent for coding tasks with low cost",
      "pricing": {
        "input_cost": 0.00025,
        "output_cost": 0.00125
      }
    },
    {
      "model_name": "gpt-3.5-turbo",
      "provider": "openai", 
      "estimated_cost": 0.0003,
      "performance_score": 0.89,
      "reasoning": "Good coding performance, cost-effective"
    }
  ],
  "total_cost": 0.00025,
  "classification": {
    "task_type": "text",
    "category": "coding",
    "complexity": "simple"
  },
  "metadata": {
    "request_id": "req_123...",
    "processing_time": "45ms",
    "model_count": 5
  }
}
```

---

## 🔄 Alternative: Enhanced Server Direct Testing

If the authentication server isn't available, you can test the core recommendation engine directly:

### Direct API Testing (No Auth Required)

**Endpoint:** `POST http://localhost:8083/api/v2/recommend/smart`

**Command:**
```bash
curl -X POST "http://localhost:8083/api/v2/recommend/smart" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Generate a photorealistic image of a sunset over mountains",
    "budget_priority": "performance_focused",
    "max_recommendations": 3
  }' | jq .
```

---

## 🧪 Advanced Testing Scenarios

### Scenario 1: Agentic Pipeline Workflow
```bash
curl -X POST "http://localhost:8080/v1/recommend" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_KEY" \
  -d '{
    "prompt": "Plan and implement a customer support ticket system: analyze requirements, design database schema, implement REST APIs, create frontend, write tests, deploy to production",
    "budget_priority": "balanced",
    "workflow_type": "agentic_pipeline",
    "max_recommendations": 10
  }' | jq .
```

### Scenario 2: Enterprise Constrained Models
```bash
curl -X POST "http://localhost:8080/v1/recommend" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_KEY" \
  -d '{
    "prompt": "Analyze quarterly financial data and create executive summary",
    "budget_priority": "cost_optimized",
    "constraints": {
      "allowed_providers": ["openai", "anthropic"],
      "allowed_models": ["gpt-4", "gpt-4o", "gpt-5", "claude-4", "claude-4-opus"],
      "max_cost_per_request": 0.50,
      "enterprise_mode": true
    },
    "max_recommendations": 5
  }' | jq .
```

### Scenario 3: Image Generation
```bash
curl -X POST "http://localhost:8080/v1/recommend" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_KEY" \
  -d '{
    "prompt": "Create a professional logo for a tech startup focusing on AI and sustainability",
    "budget_priority": "performance_focused",
    "task_type": "image_generation",
    "max_recommendations": 5
  }' | jq .
```

---

## 🐛 Troubleshooting

### Common Issues and Solutions

#### 1. **Server Not Running**
```bash
# Check if servers are running
lsof -ti:8080  # Auth server
lsof -ti:8083  # Enhanced server

# If no output, servers are not running - start them using setup commands above
```

#### 2. **Authentication Failed (401)**
- Check if access token is correct and not expired
- Ensure `Authorization: Bearer TOKEN` header format is correct
- Try refreshing the token or re-registering

#### 3. **API Key Invalid**
- Verify API key format starts with `sk_live_` or similar
- Check if API key was created successfully in Step 2
- Ensure correct Authorization header format

#### 4. **No Recommendations Returned**
- Check server logs for classification errors
- Try simpler prompts first
- Verify Analytics AI integration is working
- Check if models are loaded properly

#### 5. **Connection Refused**
```bash
# Check server logs
curl -s http://localhost:8080/healthz
curl -s http://localhost:8083/api/v2/status

# If servers are down, restart them
```

---

## 📊 Success Metrics to Verify

### ✅ Registration Success
- [ ] User created with valid email/password
- [ ] Access token received
- [ ] Token can be used for subsequent requests

### ✅ API Key Success  
- [ ] API key generated successfully
- [ ] Key has proper format (sk_live_...)
- [ ] Key can authenticate API requests

### ✅ Recommendation Success
- [ ] Models returned for different prompt types
- [ ] Cost estimates provided
- [ ] Performance scores included
- [ ] Classification working correctly

### ✅ Performance Verification
- [ ] Response time < 500ms for recommendations
- [ ] Cost savings > 70% compared to single premium model
- [ ] Classification accuracy > 85%
- [ ] Models appropriate for task type

---

## 🚀 Quick Test Script

Run this automated test to verify everything works:

```bash
# Make the test script executable and run it
chmod +x test-user-flow.sh
./test-user-flow.sh
```

This will test the complete flow automatically and provide a detailed report of what's working and what needs attention.

---

## 📈 Expected Results

**For Coding Tasks:**
- Should recommend models like Claude-3-Haiku, GPT-3.5-Turbo, CodeLlama
- Cost savings: 80-90% vs GPT-4
- Response time: <100ms

**For Image Generation:**
- Should recommend DALL-E, Midjourney, Stable Diffusion models
- Cost optimization based on quality requirements
- Response time: <200ms

**For Agentic Pipelines:**
- Should provide model sequences for multi-step workflows  
- Cost savings: 70-85% vs using premium models throughout
- Performance score: >85% accuracy

**For Enterprise Constraints:**
- Should respect provider/model limitations
- Stay within budget constraints
- Maintain high performance within limits

---

**Happy Testing! 🎉**

Need help? Check the server logs or contact the development team.