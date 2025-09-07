"""
Hybrid LLM Router Classifier Service 
Combines rule-based classification with ML models for enhanced accuracy
"""

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field
import numpy as np
import os
import re
import ssl
import logging
from typing import Dict, List, Optional, Any
import warnings

# Suppress warnings
warnings.filterwarnings("ignore")
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Initialize FastAPI app
app = FastAPI(
    title="Hybrid LLM Router Classifier",
    description="Rule-based + ML prompt classification",
    version="2.0.0"
)

# Global variables
sentence_model = None
category_embeddings = None
category_labels = []

# Models
class ClassificationRequest(BaseModel):
    prompt: str = Field(..., description="The prompt to classify")

class HybridResponse(BaseModel):
    primary_use_case: str = Field(..., description="Main category of the prompt")
    complexity_score: float = Field(..., ge=0.0, le=1.0, description="Complexity score")
    creativity_score: float = Field(..., ge=0.0, le=1.0, description="Creativity score")
    token_count_estimate: int = Field(..., description="Estimated token count")
    urgency_level: float = Field(..., ge=0.0, le=1.0, description="Time sensitivity")
    output_length_estimate: int = Field(..., description="Estimated response length")
    interaction_style: str = Field(..., description="Communication style")
    domain_confidence: float = Field(..., ge=0.0, le=1.0, description="Classification confidence")
    difficulty: str = Field(..., description="Difficulty: easy, medium, hard")
    classification_method: str = Field(..., description="Method used: rule-based, ml-based, or hybrid")
    ml_confidence: Optional[float] = Field(None, description="ML model confidence if available")

# Reference categories for ML training
REFERENCE_CATEGORIES = {
    "coding": [
        "Write a Python function to sort a list of numbers",
        "Help me debug this JavaScript code that's throwing an error",
        "Create a REST API endpoint using Flask",
        "Explain how to optimize this SQL query for better performance",
        "Build a React component for user authentication"
    ],
    "creative_writing": [
        "Write a short story about a time traveler who gets stuck in the past",
        "Create a poem about the beauty of nature in autumn",
        "Compose a creative dialogue between two characters meeting for the first time",
        "Write a fictional blog post from the perspective of an AI discovering emotions",
        "Craft a compelling opening paragraph for a mystery novel"
    ],
    "analysis": [
        "Analyze the pros and cons of remote work versus office work",
        "Compare and contrast the economic policies of two different countries",
        "Evaluate the effectiveness of different marketing strategies for startups",
        "Assess the environmental impact of electric vehicles versus gasoline cars",
        "Review the key findings from this research paper and summarize the implications"
    ],
    "math": [
        "Solve this calculus problem step by step: find the derivative of x^3 + 2x^2 - 5x + 1",
        "Calculate the probability of rolling a sum of 7 with two dice",
        "Help me understand the Pythagorean theorem with practical examples",
        "Compute the compound interest for an investment of $10,000 at 5% annually",
        "Explain linear regression and provide a simple mathematical example"
    ],
    "question": [
        "What are the key factors that influence customer purchasing decisions?",
        "How does machine learning differ from traditional programming approaches?",
        "Why is cybersecurity becoming increasingly important for small businesses?",
        "When is the best time to implement agile methodology in software development?",
        "Which programming language should I choose for data science projects?"
    ],
    "chat": [
        "Hi there! How are you doing today?",
        "Hello! I'd love to have a casual conversation about your interests",
        "Hey, what's your favorite way to spend a weekend?",
        "Good morning! Can we chat about current events?",
        "Greetings! I'm looking for someone to brainstorm ideas with"
    ],
    "general": [
        "Please provide general information about climate change",
        "Give me an overview of how the internet works",
        "Explain the basics of personal finance management",
        "Tell me about the history of artificial intelligence",
        "Describe the main components of a healthy diet"
    ]
}

def initialize_ml_model():
    """Initialize sentence transformer model with SSL handling"""
    global sentence_model, category_embeddings, category_labels
    
    try:
        # Configure SSL context to be more permissive
        import ssl
        ssl._create_default_https_context = ssl._create_unverified_context
        
        # Also set environment variables to disable SSL verification
        os.environ['CURL_CA_BUNDLE'] = ''
        os.environ['REQUESTS_CA_BUNDLE'] = ''
        os.environ['SSL_VERIFY'] = 'false'
        
        logger.info("Attempting to load sentence transformer model...")
        from sentence_transformers import SentenceTransformer
        
        # Try loading from local directory first, then fallback to remote
        local_model_path = './all-MiniLM-L6-v2'
        try:
            sentence_model = SentenceTransformer(local_model_path)
            logger.info("✅ Successfully loaded LOCAL ML model: all-MiniLM-L6-v2")
        except Exception as e:
            logger.info(f"Local model not found ({e}), trying remote download...")
            sentence_model = SentenceTransformer('all-MiniLM-L6-v2', trust_remote_code=True)
            logger.info("✅ Successfully loaded REMOTE ML model: all-MiniLM-L6-v2")
        
        # Prepare category embeddings
        all_examples = []
        category_labels = []
        
        for category, examples in REFERENCE_CATEGORIES.items():
            for example in examples:
                all_examples.append(example)
                category_labels.append(category)
        
        # Generate embeddings
        category_embeddings = sentence_model.encode(
            all_examples, 
            convert_to_numpy=True, 
            normalize_embeddings=True
        )
        logger.info(f"✅ Generated embeddings for {len(all_examples)} reference examples")
        return True
        
    except Exception as e:
        logger.warning(f"❌ Failed to load ML model: {e}")
        logger.info("📋 Falling back to rule-based classification only")
        sentence_model = None
        category_embeddings = None
        category_labels = []
        return False

def estimate_tokens(text: str) -> int:
    """Estimate token count"""
    return int(len(text.split()) * 1.3)

def classify_with_rules(prompt: str) -> tuple:
    """Rule-based classification"""
    prompt_lower = prompt.lower()
    
    # Define keyword patterns
    patterns = {
        "coding": ['function', 'class', 'code', 'program', 'script', 'debug', 'algorithm', 'python', 'javascript', 'api', 'database', 'sql'],
        "creative_writing": ['story', 'poem', 'creative', 'write', 'imagine', 'fictional', 'character', 'plot', 'narrative', 'dialogue'],
        "analysis": ['analyze', 'compare', 'evaluate', 'assess', 'review', 'examine', 'research', 'study', 'pros and cons'],
        "math": ['calculate', 'solve', 'equation', 'formula', 'mathematical', 'probability', 'statistics', 'derivative', 'integral'],
        "question": ['what', 'how', 'why', 'when', 'where', 'which'],
        "chat": ['hey', 'hi', 'hello', 'chat', 'talk', 'conversation'],
        "general": ['explain', 'tell me', 'describe', 'overview', 'information']
    }
    
    # Calculate scores for each category
    scores = {}
    for category, keywords in patterns.items():
        score = sum(1 for keyword in keywords if keyword in prompt_lower)
        if category == "question" and any(prompt_lower.startswith(word) for word in keywords):
            score += 2  # Boost for question starters
        scores[category] = score
    
    # Find best category
    best_category = max(scores.keys(), key=lambda x: scores[x])
    max_score = scores[best_category]
    
    # Calculate confidence based on score distribution
    total_score = sum(scores.values())
    if total_score == 0:
        confidence = 0.5
        best_category = "general"
    else:
        confidence = min(max_score / total_score, 1.0)
        # Apply minimum confidence threshold
        confidence = max(confidence, 0.3)
    
    return best_category, confidence

def classify_with_ml(prompt: str) -> tuple:
    """ML-based classification using sentence transformers"""
    if sentence_model is None or category_embeddings is None:
        return None, 0.0
    
    try:
        # Generate prompt embedding
        prompt_embedding = sentence_model.encode(
            [prompt], 
            convert_to_numpy=True, 
            normalize_embeddings=True
        )[0]
        
        # Calculate similarities
        similarities = category_embeddings @ prompt_embedding
        
        # Find best match
        best_idx = np.argmax(similarities)
        best_category = category_labels[best_idx]
        confidence = float(similarities[best_idx])
        
        return best_category, confidence
        
    except Exception as e:
        logger.error(f"ML classification failed: {e}")
        return None, 0.0

def hybrid_classify(prompt: str) -> tuple:
    """Hybrid classification combining rules and ML"""
    
    # Get rule-based result
    rule_category, rule_confidence = classify_with_rules(prompt)
    
    # Get ML result if available
    ml_category, ml_confidence = classify_with_ml(prompt)
    
    if ml_category is None:
        # ML not available, use rule-based only
        return rule_category, rule_confidence, "rule-based", None
    
    # Both available - use hybrid approach
    if ml_confidence > 0.8:
        # High ML confidence, trust ML
        return ml_category, ml_confidence, "ml-based", ml_confidence
    elif rule_confidence > 0.7 and ml_confidence < 0.6:
        # High rule confidence, low ML confidence - trust rules
        return rule_category, rule_confidence, "rule-based", ml_confidence
    elif rule_category == ml_category:
        # Both agree - high confidence
        combined_confidence = (rule_confidence + ml_confidence) / 2
        return rule_category, min(combined_confidence + 0.1, 1.0), "hybrid", ml_confidence
    else:
        # Disagreement - use higher confidence
        if ml_confidence > rule_confidence:
            return ml_category, ml_confidence, "ml-based", ml_confidence
        else:
            return rule_category, rule_confidence, "rule-based", ml_confidence

def calculate_other_metrics(prompt: str, category: str) -> dict:
    """Calculate complexity, creativity, and other metrics"""
    token_count = estimate_tokens(prompt)
    
    # Complexity based on length and technical terms
    base_complexity = min(token_count / 100.0, 0.5)
    tech_terms = ['algorithm', 'optimization', 'implementation', 'architecture', 'methodology']
    tech_bonus = sum(0.1 for term in tech_terms if term in prompt.lower())
    complexity = min(base_complexity + tech_bonus, 1.0)
    
    # Creativity based on category and creative indicators
    creative_categories = {"creative_writing": 0.9, "chat": 0.6, "general": 0.4}
    base_creativity = creative_categories.get(category, 0.3)
    creative_words = ['imagine', 'creative', 'story', 'unique', 'original']
    creative_bonus = sum(0.1 for word in creative_words if word in prompt.lower())
    creativity = min(base_creativity + creative_bonus, 1.0)
    
    # Other metrics
    urgency = 0.2  # Default low urgency
    output_length = min(token_count * 2, 1000)
    
    # Interaction style
    if any(word in prompt.lower() for word in ['hey', 'hi', 'chat', 'talk']):
        interaction_style = "conversational"
    elif len(prompt) > 100:
        interaction_style = "formal"
    else:
        interaction_style = "direct"
    
    # Difficulty
    if complexity > 0.7:
        difficulty = "hard"
    elif complexity > 0.4:
        difficulty = "medium"
    else:
        difficulty = "easy"
    
    return {
        "complexity_score": complexity,
        "creativity_score": creativity,
        "token_count_estimate": token_count,
        "urgency_level": urgency,
        "output_length_estimate": output_length,
        "interaction_style": interaction_style,
        "difficulty": difficulty
    }

@app.on_event("startup")
async def startup_event():
    """Initialize the service"""
    logger.info("🚀 Starting Hybrid Classifier Service")
    ml_available = initialize_ml_model()
    if ml_available:
        logger.info("✅ Hybrid mode: Rule-based + ML classification")
    else:
        logger.info("📋 Fallback mode: Rule-based classification only")

@app.get("/health")
async def health_check():
    """Health check endpoint"""
    mode = "hybrid" if sentence_model is not None else "rule-based"
    return {
        "status": "healthy", 
        "mode": mode,
        "ml_available": sentence_model is not None
    }

@app.post("/classify", response_model=HybridResponse)
async def classify_text(request: ClassificationRequest):
    """Main hybrid classification endpoint"""
    try:
        prompt = request.prompt.strip()
        if not prompt:
            raise HTTPException(status_code=400, detail="Empty prompt")
        
        # Perform hybrid classification
        category, confidence, method, ml_conf = hybrid_classify(prompt)
        
        # Calculate other metrics
        metrics = calculate_other_metrics(prompt, category)
        
        return HybridResponse(
            primary_use_case=category,
            domain_confidence=confidence,
            classification_method=method,
            ml_confidence=ml_conf,
            **metrics
        )
        
    except Exception as e:
        logger.error(f"Classification error: {e}")
        raise HTTPException(status_code=500, detail="Classification failed")

@app.get("/info")
async def get_info():
    """Get service information"""
    return {
        "service": "Hybrid LLM Router Classifier",
        "version": "2.0.0",
        "ml_model_available": sentence_model is not None,
        "ml_model": "all-MiniLM-L6-v2" if sentence_model else None,
        "categories": list(REFERENCE_CATEGORIES.keys()),
        "classification_methods": ["rule-based", "ml-based", "hybrid"]
    }

if __name__ == "__main__":
    import uvicorn
    port = int(os.environ.get("PORT", 5000))
    uvicorn.run(app, host="0.0.0.0", port=port)