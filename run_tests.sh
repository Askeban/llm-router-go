#!/bin/bash

# Enhanced LLM Router System Test Runner
# This script sets up the environment and runs comprehensive tests

echo "🚀 Enhanced LLM Router System Test Runner"
echo "=========================================="

# Set environment variables
export CLASSIFIER_URL="http://localhost:5000"
export DATABASE_PATH="./test_data.db"
export MODEL_PROFILES_PATH="./internal/models.json"
export PORT="8080"

# Create directories
mkdir -p configs python/classifier_service python/ingestor_service

# Check if Python is available
if ! command -v python3 &> /dev/null; then
    echo "❌ Python 3 is required but not installed"
    exit 1
fi

# Check if Go is available  
if ! command -v go &> /dev/null; then
    echo "❌ Go is required but not installed"
    exit 1
fi

echo "✅ Prerequisites check passed"

# Install Python dependencies for classifier
echo "📦 Installing classifier dependencies..."
cd python/classifier_service
if [ -f "requirements.txt" ]; then
    pip3 install -r requirements.txt > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        echo "✅ Classifier dependencies installed"
    else
        echo "⚠️ Some classifier dependencies may have failed to install"
    fi
else
    echo "⚠️ Classifier requirements.txt not found, installing basic dependencies"
    pip3 install fastapi uvicorn sentence-transformers torch scikit-learn numpy tiktoken > /dev/null 2>&1
fi
cd ../..

# Install Python dependencies for ingestor
echo "📦 Installing ingestor dependencies..."
cd python/ingestor_service
if [ -f "requirements.txt" ]; then
    pip3 install -r requirements.txt > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        echo "✅ Ingestor dependencies installed"
    else
        echo "⚠️ Some ingestor dependencies may have failed to install"
    fi
else
    echo "⚠️ Ingestor requirements.txt not found, installing basic dependencies"
    pip3 install fastapi uvicorn pandas numpy aiohttp pydantic > /dev/null 2>&1
fi
cd ../..

# Install Go dependencies
echo "📦 Installing Go dependencies..."
go mod tidy > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✅ Go dependencies installed"
else
    echo "⚠️ Some Go dependencies may have failed to install"
fi

# Build Go application
echo "🔨 Building Go application..."
go build -o router cmd/server/main.go
if [ $? -eq 0 ]; then
    echo "✅ Go application built successfully"
else
    echo "❌ Failed to build Go application"
    exit 1
fi

echo ""
echo "🧪 Starting comprehensive test suite..."
echo "This will:"
echo "  1. Start the Enhanced Classifier Service (Python)"
echo "  2. Start the Enhanced Ingestor Service (Python)" 
echo "  3. Start the Enhanced Router Service (Go)"
echo "  4. Run comprehensive tests with sample data"
echo "  5. Generate detailed test report"
echo ""
echo "⚠️ This will take 2-3 minutes and use ports 5000, 8001, and 8080"
echo ""

read -p "Continue? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "👋 Test cancelled by user"
    exit 0
fi

# Run the comprehensive test suite
echo "🔥 Running comprehensive tests..."
python3 test_suite.py

echo ""
echo "✅ Test suite completed!"
echo "📊 Check the output above for detailed results"
echo ""
echo "🔧 Manual testing commands:"
echo "  # Start services individually:"
echo "  cd python/classifier_service && python3 enhanced_app.py"
echo "  cd python/ingestor_service && python3 main_service.py"  
echo "  ./router"
echo ""
echo "  # Test individual endpoints:"
echo "  curl -X POST http://localhost:8080/route/enhanced \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"prompt\":\"Write a Python function\",\"preference\":\"balanced\"}'"
echo ""
echo "  curl -X POST http://localhost:5000/classify/advanced \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"prompt\":\"Help me with coding\"}'"
echo ""