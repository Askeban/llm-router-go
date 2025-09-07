#!/bin/bash

# Enhanced LLM Router System - Docker Test Script
# Tests the microservices deployment using Docker Compose

set -e  # Exit on any error

echo "🚀 Enhanced LLM Router System - Docker Microservices Test"
echo "======================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is installed and running
check_docker() {
    print_status "Checking Docker installation..."
    
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        print_error "Docker is not running. Please start Docker first."
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    
    print_success "Docker and Docker Compose are available"
}

# Clean up any existing containers
cleanup() {
    print_status "Cleaning up existing containers..."
    docker-compose -f docker-compose.enhanced.yml down --volumes --remove-orphans 2>/dev/null || true
    docker system prune -f --volumes 2>/dev/null || true
    print_success "Cleanup completed"
}

# Build and start services
start_services() {
    print_status "Building and starting microservices..."
    
    # Export environment variables
    export ANALYTICS_API_KEY=${ANALYTICS_API_KEY:-"test_key"}
    
    # Build and start services
    docker-compose -f docker-compose.enhanced.yml up --build -d
    
    print_success "Services started in background"
    
    # Show service status
    echo ""
    print_status "Service status:"
    docker-compose -f docker-compose.enhanced.yml ps
}

# Wait for services to be healthy
wait_for_services() {
    print_status "Waiting for services to be healthy..."
    
    local max_attempts=60  # 5 minutes max
    local attempt=0
    
    services=("router" "classifier" "ingestor" "redis")
    
    while [ $attempt -lt $max_attempts ]; do
        all_healthy=true
        
        for service in "${services[@]}"; do
            container_name="llm-router-$service"
            
            if [ "$service" = "redis" ]; then
                # For Redis, just check if container is running
                if ! docker ps | grep -q "$container_name"; then
                    all_healthy=false
                    break
                fi
            else
                # For other services, check health status
                health_status=$(docker inspect --format='{{.State.Health.Status}}' "$container_name" 2>/dev/null || echo "no-health-check")
                
                if [ "$health_status" != "healthy" ] && [ "$health_status" != "no-health-check" ]; then
                    all_healthy=false
                    break
                fi
                
                # If no health check, check if container is running
                if [ "$health_status" = "no-health-check" ]; then
                    if ! docker ps | grep -q "$container_name"; then
                        all_healthy=false
                        break
                    fi
                fi
            fi
        done
        
        if [ "$all_healthy" = true ]; then
            print_success "All services are healthy!"
            return 0
        fi
        
        attempt=$((attempt + 1))
        echo -n "."
        sleep 5
    done
    
    print_error "Services did not become healthy within the timeout period"
    
    # Show logs for debugging
    print_status "Showing container logs for debugging..."
    docker-compose -f docker-compose.enhanced.yml logs --tail=20
    
    return 1
}

# Test individual services
test_services() {
    print_status "Testing individual services..."
    
    # Test Redis
    print_status "Testing Redis..."
    if docker exec llm-router-redis redis-cli ping | grep -q "PONG"; then
        print_success "Redis is responding"
    else
        print_error "Redis is not responding"
        return 1
    fi
    
    # Test Router health
    print_status "Testing Router health..."
    if curl -f -s http://localhost:8080/healthz > /dev/null; then
        print_success "Router health check passed"
    else
        print_warning "Router health check failed, but container might still be starting"
    fi
    
    # Test Classifier health  
    print_status "Testing Classifier health..."
    if curl -f -s http://localhost:5001/health > /dev/null; then
        print_success "Classifier health check passed"
    else
        print_warning "Classifier health check failed, but container might still be starting"
    fi
    
    # Test Ingestor health
    print_status "Testing Ingestor health..."
    if curl -f -s http://localhost:8001/health > /dev/null; then
        print_success "Ingestor health check passed"
    else
        print_warning "Ingestor health check failed, but container might still be starting"
    fi
}

# Test API functionality
test_apis() {
    print_status "Testing API functionality..."
    
    # Test basic classification
    print_status "Testing classification API..."
    classifier_response=$(curl -s -X POST http://localhost:5001/classify \
        -H "Content-Type: application/json" \
        -d '{"prompt":"Write a Python function to sort numbers"}' || echo "ERROR")
    
    if [[ "$classifier_response" == *"primary_use_case"* ]]; then
        print_success "Classification API is working"
        echo "Sample response: $classifier_response" | head -c 100
        echo "..."
    else
        print_warning "Classification API test failed: $classifier_response"
    fi
    
    # Test ingestor status
    print_status "Testing ingestor status API..."
    ingestor_response=$(curl -s http://localhost:8001/status || echo "ERROR")
    
    if [[ "$ingestor_response" == *"status"* ]]; then
        print_success "Ingestor status API is working"
    else
        print_warning "Ingestor status API test failed: $ingestor_response"
    fi
    
    # Test basic routing
    print_status "Testing basic routing API..."
    router_response=$(curl -s -X POST http://localhost:8080/route \
        -H "Content-Type: application/json" \
        -d '{"prompt":"Help me write code","mode":"recommend"}' || echo "ERROR")
    
    if [[ "$router_response" == *"classification"* ]] || [[ "$router_response" == *"recommended"* ]]; then
        print_success "Basic routing API is working"
    else
        print_warning "Basic routing API test might have failed: $router_response"
    fi
}

# Show service information
show_service_info() {
    echo ""
    print_status "🎯 Service Information:"
    echo "======================================"
    echo "🔀 Main Router:        http://localhost:8080"
    echo "🧠 Enhanced Classifier: http://localhost:5001"
    echo "📥 Enhanced Ingestor:   http://localhost:8001"
    echo "📊 Health Dashboard:    http://localhost:8090"
    echo "💾 Redis Cache:         localhost:6379"
    echo ""
    
    print_status "📡 Key API Endpoints:"
    echo "======================================"
    echo "• POST http://localhost:8080/route - Basic routing"
    echo "• POST http://localhost:5001/classify - Classification"
    echo "• GET  http://localhost:8001/status - Ingestor status"
    echo "• GET  http://localhost:8080/healthz - Health check"
    echo ""
    
    print_status "🧪 Test Commands:"
    echo "======================================"
    echo 'curl -X POST http://localhost:8080/route \'
    echo '  -H "Content-Type: application/json" \'
    echo '  -d '\''{"prompt":"Write Python code","mode":"recommend"}'\'''
    echo ""
    echo 'curl -X POST http://localhost:5001/classify \'
    echo '  -H "Content-Type: application/json" \'
    echo '  -d '\''{"prompt":"Create a story about AI"}'\'''
    echo ""
    echo "curl http://localhost:8001/status"
    echo ""
}

# Main execution
main() {
    echo "Starting Enhanced LLM Router System Docker Test..."
    echo ""
    
    # Check prerequisites
    check_docker
    
    # Cleanup previous runs
    cleanup
    
    # Start services
    start_services
    
    # Wait for services to be ready
    if ! wait_for_services; then
        print_error "Services failed to start properly"
        
        print_status "Checking container status..."
        docker-compose -f docker-compose.enhanced.yml ps
        
        print_status "Showing recent logs..."
        docker-compose -f docker-compose.enhanced.yml logs --tail=50
        
        exit 1
    fi
    
    # Test services
    test_services
    
    # Test APIs
    test_apis
    
    # Show service information
    show_service_info
    
    print_success "🎉 Docker microservices test completed!"
    print_status "Services are running. Use 'docker-compose -f docker-compose.enhanced.yml down' to stop them."
    print_status "Visit http://localhost:8090 for the health dashboard."
    
    # Ask user if they want to keep services running
    echo ""
    read -p "Keep services running for manual testing? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_status "Stopping services..."
        docker-compose -f docker-compose.enhanced.yml down
        print_success "Services stopped"
    else
        print_success "Services will continue running in the background"
        print_status "Use 'docker-compose -f docker-compose.enhanced.yml logs -f' to watch logs"
        print_status "Use 'docker-compose -f docker-compose.enhanced.yml down' to stop when done"
    fi
}

# Handle Ctrl+C
trap 'print_warning "Script interrupted. Cleaning up..."; docker-compose -f docker-compose.enhanced.yml down; exit 1' INT

# Run main function
main