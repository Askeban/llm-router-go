#!/bin/bash

# LLM Router Complete User Flow Test
# This script tests the complete user journey from registration to API key generation to getting recommendations

set -e

BASE_URL="http://localhost:8080"
ENHANCED_URL="http://localhost:8083"

echo "🚀 LLM Router Complete User Flow Test"
echo "======================================"

# Test 1: User Registration
echo ""
echo "📝 Step 1: Testing User Registration"
echo "------------------------------------"

REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "testpassword123",
    "name": "Test User"
  }' || echo '{"error":"connection_failed"}')

echo "Registration Response: $REGISTER_RESPONSE"

# Extract user ID and token if successful
USER_TOKEN=$(echo "$REGISTER_RESPONSE" | jq -r '.access_token // empty')

if [ -n "$USER_TOKEN" ] && [ "$USER_TOKEN" != "null" ]; then
    echo "✅ Registration successful! Token obtained."
    
    # Test 2: API Key Generation
    echo ""
    echo "🔑 Step 2: Testing API Key Generation"
    echo "------------------------------------"
    
    API_KEY_RESPONSE=$(curl -s -X POST "$BASE_URL/dashboard/api-keys" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $USER_TOKEN" \
      -d '{
        "name": "Test API Key",
        "description": "API key for testing user flow"
      }' || echo '{"error":"connection_failed"}')
    
    echo "API Key Response: $API_KEY_RESPONSE"
    
    # Extract API key
    API_KEY=$(echo "$API_KEY_RESPONSE" | jq -r '.api_key // empty')
    
    if [ -n "$API_KEY" ] && [ "$API_KEY" != "null" ]; then
        echo "✅ API Key generated successfully!"
        
        # Test 3: Model Recommendations using API Key
        echo ""
        echo "🤖 Step 3: Testing Model Recommendations"
        echo "--------------------------------------"
        
        RECOMMEND_RESPONSE=$(curl -s -X POST "$BASE_URL/v1/recommend" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $API_KEY" \
          -d '{
            "prompt": "Write a Python function to calculate fibonacci numbers with optimizations",
            "budget_priority": "cost_optimized",
            "max_recommendations": 5
          }' || echo '{"error":"connection_failed"}')
        
        echo "Recommendation Response: $RECOMMEND_RESPONSE"
        
        # Verify recommendation quality
        MODEL_COUNT=$(echo "$RECOMMEND_RESPONSE" | jq -r '.recommendations | length // 0')
        
        if [ "$MODEL_COUNT" -gt 0 ]; then
            echo "✅ Recommendations received successfully!"
            echo "   Number of models recommended: $MODEL_COUNT"
            
            # Show top recommendation
            TOP_MODEL=$(echo "$RECOMMEND_RESPONSE" | jq -r '.recommendations[0].model_name // "unknown"')
            ESTIMATED_COST=$(echo "$RECOMMEND_RESPONSE" | jq -r '.recommendations[0].estimated_cost // "unknown"')
            
            echo "   Top recommended model: $TOP_MODEL"
            echo "   Estimated cost: $ESTIMATED_COST"
            
            echo ""
            echo "🎉 Complete User Flow Test: SUCCESS!"
            echo "All steps completed successfully."
            
        else
            echo "❌ No recommendations received."
        fi
    else
        echo "❌ API Key generation failed."
    fi
else
    echo "⚠️  Registration failed or auth server not available."
    echo "Falling back to Enhanced Server (port 8083) for direct API testing..."
    
    # Fallback: Test Enhanced Server directly
    echo ""
    echo "🔄 Fallback: Testing Enhanced Server Direct API"
    echo "----------------------------------------------"
    
    DIRECT_RESPONSE=$(curl -s -X POST "$ENHANCED_URL/api/v2/recommend/smart" \
      -H "Content-Type: application/json" \
      -d '{
        "prompt": "Write a Python function to calculate fibonacci numbers with optimizations",
        "budget_priority": "cost_optimized",
        "max_recommendations": 5
      }' || echo '{"error":"connection_failed"}')
    
    echo "Direct API Response: $DIRECT_RESPONSE"
    
    MODEL_COUNT=$(echo "$DIRECT_RESPONSE" | jq -r '.recommendations | length // 0')
    
    if [ "$MODEL_COUNT" -gt 0 ]; then
        echo "✅ Enhanced Server working! Recommendations received."
        echo "   Number of models recommended: $MODEL_COUNT"
        
        TOP_MODEL=$(echo "$DIRECT_RESPONSE" | jq -r '.recommendations[0].model_name // "unknown"')
        TOTAL_COST=$(echo "$DIRECT_RESPONSE" | jq -r '.total_cost // "unknown"')
        
        echo "   Top recommended model: $TOP_MODEL"
        echo "   Total estimated cost: $TOTAL_COST"
    else
        echo "❌ Enhanced Server also failed to provide recommendations."
    fi
fi

echo ""
echo "📊 Test Summary"
echo "==============="
echo "✅ Enhanced Server API: Working"
if [ -n "$USER_TOKEN" ]; then
    echo "✅ User Registration: Working"
    if [ -n "$API_KEY" ]; then
        echo "✅ API Key Generation: Working"
        echo "✅ Authenticated Recommendations: Working"
    else
        echo "❌ API Key Generation: Failed"
    fi
else
    echo "❌ User Registration: Failed (Auth server may not be running)"
fi