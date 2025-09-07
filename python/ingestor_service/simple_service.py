"""
Simple LLM Router Ingestor Service - Basic version for testing
Provides ingestor functionality without ML dependencies
"""

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field
import json
import os
import logging
from typing import Dict, List, Any, Optional
from datetime import datetime

# Initialize FastAPI app
app = FastAPI(
    title="Simple LLM Router Ingestor",
    description="Basic data ingestor service for testing",
    version="1.0.0"
)

# Models
class ModelInfo(BaseModel):
    id: str
    provider: str
    display_name: str
    status: str = "active"
    last_updated: str

class IngestorStatus(BaseModel):
    status: str
    total_models: int
    last_sync: Optional[str]
    active_models: int
    data_sources: List[str]

class ModelUpdateRequest(BaseModel):
    models: List[Dict[str, Any]]

# Global data store
models_data = []
last_sync_time = None

def load_models_data():
    """Load initial models data"""
    global models_data
    
    # Try to load from the mounted models.json file
    models_path = os.environ.get("MODELS_JSON_PATH", "/app/data/models.json")
    
    try:
        with open(models_path, 'r') as f:
            data = json.load(f)
            # Convert to our format
            models_data = []
            for model in data:
                models_data.append({
                    "id": model.get("id", "unknown"),
                    "provider": model.get("provider", "unknown"),
                    "display_name": model.get("display_name", model.get("id", "Unknown")),
                    "status": "active",
                    "last_updated": datetime.now().isoformat()
                })
        logging.info(f"Loaded {len(models_data)} models from {models_path}")
        
    except FileNotFoundError:
        logging.warning(f"Models file not found at {models_path}, using default models")
        # Default models
        models_data = [
            {
                "id": "gpt-4",
                "provider": "openai",
                "display_name": "GPT-4",
                "status": "active",
                "last_updated": datetime.now().isoformat()
            },
            {
                "id": "gpt-3.5-turbo",
                "provider": "openai", 
                "display_name": "GPT-3.5 Turbo",
                "status": "active",
                "last_updated": datetime.now().isoformat()
            },
            {
                "id": "claude-3-sonnet",
                "provider": "anthropic",
                "display_name": "Claude 3 Sonnet", 
                "status": "active",
                "last_updated": datetime.now().isoformat()
            }
        ]
    except Exception as e:
        logging.error(f"Error loading models: {e}")
        models_data = []

@app.on_event("startup")
async def startup_event():
    """Initialize the service"""
    global last_sync_time
    logging.info("Starting Simple Ingestor Service")
    load_models_data()
    last_sync_time = datetime.now().isoformat()

@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {"status": "healthy", "mode": "simple"}

@app.get("/status", response_model=IngestorStatus)
async def get_status():
    """Get ingestor status"""
    active_models = len([m for m in models_data if m.get("status") == "active"])
    
    return IngestorStatus(
        status="operational",
        total_models=len(models_data),
        last_sync=last_sync_time,
        active_models=active_models,
        data_sources=["models.json", "default_data"]
    )

@app.get("/models")
async def get_models():
    """Get all models"""
    return {
        "models": models_data,
        "count": len(models_data),
        "last_updated": last_sync_time
    }

@app.post("/update")
async def update_models(request: ModelUpdateRequest):
    """Update models data"""
    global models_data, last_sync_time
    
    try:
        # Simple update - replace existing data
        models_data = []
        for model_data in request.models:
            models_data.append({
                "id": model_data.get("id", "unknown"),
                "provider": model_data.get("provider", "unknown"), 
                "display_name": model_data.get("display_name", model_data.get("id", "Unknown")),
                "status": model_data.get("status", "active"),
                "last_updated": datetime.now().isoformat()
            })
        
        last_sync_time = datetime.now().isoformat()
        
        return {
            "success": True,
            "updated_models": len(models_data),
            "timestamp": last_sync_time
        }
        
    except Exception as e:
        logging.error(f"Update failed: {e}")
        raise HTTPException(status_code=500, detail=f"Update failed: {str(e)}")

@app.get("/models/{model_id}")
async def get_model(model_id: str):
    """Get specific model by ID"""
    for model in models_data:
        if model["id"] == model_id:
            return model
    
    raise HTTPException(status_code=404, detail="Model not found")

if __name__ == "__main__":
    import uvicorn
    port = int(os.environ.get("PORT", 8001))
    uvicorn.run(app, host="0.0.0.0", port=port)