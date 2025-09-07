"""
Enhanced Multi-Layer Data Ingestor Service
Combines static, scraped, and real-time data for intelligent model quantification
"""

import asyncio
import json
import os
import glob
import logging
import redis
import numpy as np
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any, Tuple
from dataclasses import dataclass, asdict
from pathlib import Path
import aiohttp
import yaml
from fastapi import FastAPI, HTTPException, BackgroundTasks
from pydantic import BaseModel, Field

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Initialize FastAPI
app = FastAPI(
    title="Enhanced LLM Router Ingestor",
    description="Multi-layer intelligent data consolidation service",
    version="2.0.0"
)

# Data Models
@dataclass
class CategoryScore:
    score: float
    confidence: float
    contributing_scores: Dict[str, Dict[str, Any]]
    last_updated: str

@dataclass
class DataProvenance:
    source: str
    last_updated: str
    data_quality: float

@dataclass
class EnhancedModel:
    model_id: str
    provider: str
    display_name: str
    static_data: Dict[str, Any]
    category_scores: Dict[str, CategoryScore]
    data_provenance: Dict[str, DataProvenance]
    performance_metadata: Dict[str, Any]
    last_consolidated: str
    overall_quality: float

# Configuration
class Config:
    def __init__(self, config_path: str = "config.yaml"):
        try:
            with open(config_path, 'r') as f:
                self.config = yaml.safe_load(f)
        except FileNotFoundError:
            logger.warning(f"Config file {config_path} not found, using defaults")
            self.config = {}
        except Exception as e:
            logger.error(f"Failed to load config: {e}")
            self.config = {}
    
    def get(self, path: str, default=None):
        if not self.config:
            return default
        keys = path.split('.')
        value = self.config
        for key in keys:
            if isinstance(value, dict) and key in value:
                value = value[key]
            else:
                return default
        return value

# Global instances
config = Config()
redis_client = None

# Classification Categories Configuration
CLASSIFICATION_CATEGORIES = {
    "coding": {
        "static_indicators": ["api_alias", "tags"],
        "benchmark_weights": {
            "humaneval": 1.0,
            "livecodebench": 0.8,
            "artificial_analysis_coding_index": 0.9
        },
        "formula": "weighted_average"
    },
    "math": {
        "static_indicators": ["capabilities.math"],
        "benchmark_weights": {
            "gsm8k": 1.0,
            "artificial_analysis_math_index": 0.9,
            "math": 0.7
        },
        "formula": "weighted_average"
    },
    "reasoning": {
        "static_indicators": ["context_window", "capabilities.reasoning"],
        "benchmark_weights": {
            "mmlu": 1.0,
            "mmlu_pro": 1.1,
            "arc_challenge": 0.8,
            "artificial_analysis_intelligence_index": 0.9,
            "gpqa": 0.7
        },
        "formula": "weighted_average_with_context"
    },
    "creative_writing": {
        "static_indicators": ["context_window", "cost_in_per_1k"],
        "benchmark_weights": {
            "hellaswag": 0.6,
            "truthfulqa": 0.4
        },
        "formula": "interpolated_with_bonuses"
    },
    "general": {
        "static_indicators": [],
        "benchmark_weights": {
            "mmlu": 0.8,
            "hellaswag": 0.6
        },
        "formula": "balanced_average"
    },
    "question": {
        "static_indicators": ["avg_latency_ms"],
        "benchmark_weights": {
            "mmlu": 0.8,
            "truthfulqa": 0.9,
            "artificial_analysis_intelligence_index": 0.7
        },
        "formula": "accuracy_with_latency_penalty"
    },
    "chat": {
        "static_indicators": ["cost_in_per_1k", "avg_latency_ms"],
        "benchmark_weights": {
            "hellaswag": 0.7,
            "truthfulqa": 0.6
        },
        "formula": "conversational_with_efficiency"
    }
}

class DataSourceManager:
    """Base class for data source management"""
    
    def __init__(self, source_name: str):
        self.source_name = source_name
        self.last_update = None
        self.data_quality = 0.0
    
    async def fetch_data(self) -> Dict[str, Any]:
        raise NotImplementedError
    
    def calculate_quality(self, data: Dict[str, Any]) -> float:
        """Calculate data quality score based on completeness and freshness"""
        if not data:
            return 0.0
        
        completeness = self._calculate_completeness(data)
        freshness = self._calculate_freshness()
        consistency = self._calculate_consistency(data)
        
        return (completeness * 0.5 + freshness * 0.3 + consistency * 0.2)
    
    def _calculate_completeness(self, data: Dict[str, Any]) -> float:
        """Calculate data completeness score"""
        required_fields = self._get_required_fields()
        present_fields = 0
        
        for model_id, model_data in data.items():
            for field in required_fields:
                if self._has_field(model_data, field):
                    present_fields += 1
        
        total_expected = len(data) * len(required_fields)
        return present_fields / total_expected if total_expected > 0 else 0.0
    
    def _calculate_freshness(self) -> float:
        """Calculate data freshness score"""
        if not self.last_update:
            return 0.0
        
        age = datetime.now() - self.last_update
        max_age = timedelta(days=30)  # Data older than 30 days is considered stale
        
        if age > max_age:
            return 0.0
        
        return max(0.0, 1.0 - (age.total_seconds() / max_age.total_seconds()))
    
    def _calculate_consistency(self, data: Dict[str, Any]) -> float:
        """Calculate data consistency score"""
        # Simple heuristic: check for reasonable score ranges
        scores = []
        for model_data in data.values():
            if isinstance(model_data, dict):
                for key, value in model_data.items():
                    if isinstance(value, (int, float)) and 0 <= value <= 100:
                        scores.append(value)
        
        if not scores:
            return 0.5  # Neutral if no numeric scores found
        
        # Check if scores fall within reasonable ranges
        mean_score = np.mean(scores)
        std_score = np.std(scores)
        
        # Good consistency if standard deviation is reasonable
        return max(0.0, min(1.0, 1.0 - (std_score / 50.0)))
    
    def _get_required_fields(self) -> List[str]:
        return []
    
    def _has_field(self, data: Dict[str, Any], field: str) -> bool:
        """Check if field exists in nested data structure"""
        if '.' in field:
            keys = field.split('.')
            current = data
            for key in keys:
                if isinstance(current, dict) and key in current:
                    current = current[key]
                else:
                    return False
            return current is not None
        return field in data

class StaticDataManager(DataSourceManager):
    """Manages static model data from models.json"""
    
    def __init__(self):
        super().__init__("static")
        # Check for environment variable first, then config, then default
        self.file_path = os.getenv('MODELS_JSON_PATH', 
                                  config.get('data_sources.static.path', 'internal/models.json'))
    
    async def fetch_data(self) -> Dict[str, Any]:
        """Load static model data"""
        try:
            with open(self.file_path, 'r') as f:
                models_list = json.load(f)
            
            # Convert list to dict keyed by model ID
            data = {}
            for model in models_list:
                if 'id' in model:
                    data[model['id']] = model
            
            self.last_update = datetime.now()
            self.data_quality = self.calculate_quality(data)
            
            logger.info(f"Loaded {len(data)} models from static data")
            return data
            
        except Exception as e:
            logger.error(f"Failed to load static data: {e}")
            return {}
    
    def _get_required_fields(self) -> List[str]:
        return ['id', 'provider', 'display_name']

class BenchmarkDataManager(DataSourceManager):
    """Manages scraped benchmark data"""
    
    def __init__(self):
        super().__init__("benchmarks")
        # Check for environment variable first, then config, then default  
        self.data_dir = os.getenv('SCRAPED_DATA_DIR',
                                 config.get('data_sources.scraped.directory', 'configs'))
        self.file_pattern = config.get('data_sources.scraped.pattern', 'trun*.json')
    
    async def fetch_data(self) -> Dict[str, Any]:
        """Load benchmark data from multiple trun*.json files with proper parsing"""
        try:
            data = {}
            pattern = os.path.join(self.data_dir, self.file_pattern)
            
            for file_path in glob.glob(pattern):
                with open(file_path, 'r') as f:
                    file_data = json.load(f)
                    
                    # Parse the structured trun*.json format
                    if 'output' in file_data:
                        output = file_data['output']
                        
                        # Extract models from different provider profiles
                        provider_keys = [
                            'openai_models_profile', 'anthropic_models_profile', 
                            'google_models_profile', 'meta_nvidia_models_profile',
                            'mistral_ai_models_profile', 'xai_models_profile',
                            'other_notable_models_profile'
                        ]
                        
                        # Also check for alternative naming patterns
                        alt_keys = [
                            'openai_models', 'anthropic_models', 'google_models',
                            'meta_models', 'mistral_models', 'xai_models'
                        ]
                        
                        all_keys = provider_keys + alt_keys
                        
                        for key in all_keys:
                            if key in output and isinstance(output[key], list):
                                for model_info in output[key]:
                                    if isinstance(model_info, dict):
                                        # Extract model identifier
                                        model_id = None
                                        
                                        # Try different ID field names
                                        id_fields = ['model_name', 'api_alias', 'id', 'name']
                                        for id_field in id_fields:
                                            if id_field in model_info:
                                                model_id = model_info[id_field]
                                                if isinstance(model_id, str):
                                                    # For api_alias, take first alias if comma-separated
                                                    if ',' in model_id:
                                                        model_id = model_id.split(',')[0].strip()
                                                    break
                                        
                                        if model_id:
                                            # Structure the model data with rich benchmark info
                                            structured_data = {
                                                'source': 'benchmark',
                                                'provider': key.replace('_models_profile', '').replace('_models', ''),
                                                'last_updated': datetime.now().isoformat()
                                            }
                                            
                                            # Map key fields
                                            field_mapping = {
                                                'model_name': 'display_name',
                                                'api_alias': 'api_name',
                                                'context_window_tokens': 'context_window',
                                                'pricing_details': 'pricing',
                                                'benchmark_highlights': 'benchmarks',
                                                'benchmark_scores': 'benchmarks',
                                                'best_use_cases': 'use_cases',
                                                'capabilities_and_modalities': 'capabilities',
                                                'modalities': 'modalities',
                                                'availability_status': 'status'
                                            }
                                            
                                            for orig_key, new_key in field_mapping.items():
                                                if orig_key in model_info:
                                                    structured_data[new_key] = model_info[orig_key]
                                            
                                            # Parse pricing if it's a string
                                            if 'pricing' in structured_data and isinstance(structured_data['pricing'], str):
                                                pricing_text = structured_data['pricing']
                                                structured_data['pricing_parsed'] = self._parse_pricing(pricing_text)
                                            
                                            # Parse benchmarks if it's a string  
                                            if 'benchmarks' in structured_data and isinstance(structured_data['benchmarks'], str):
                                                benchmark_text = structured_data['benchmarks']
                                                structured_data['benchmarks_parsed'] = self._parse_benchmarks(benchmark_text)
                                            
                                            data[model_id] = structured_data
                    
                    # Fallback for simple JSON structure
                    else:
                        data.update(file_data)
            
            self.last_update = datetime.now()
            self.data_quality = self.calculate_quality(data)
            
            logger.info(f"Loaded benchmark data for {len(data)} models from {len(glob.glob(pattern))} files")
            return data
            
        except Exception as e:
            logger.error(f"Failed to load benchmark data: {e}")
            return {}
    
    def _parse_pricing(self, pricing_text: str) -> Dict[str, Any]:
        """Parse pricing information from text"""
        try:
            pricing = {}
            # Extract costs using regex
            import re
            
            # Look for patterns like "$1.25/1M" or "$0.03/1K"
            cost_pattern = r'\$(\d+\.?\d*)/(\d+[KM]?)\s*(\w+\s*)?tokens?'
            matches = re.findall(cost_pattern, pricing_text, re.IGNORECASE)
            
            for match in matches:
                cost, unit, token_type = match
                cost = float(cost)
                
                # Normalize to per 1K tokens
                if 'M' in unit.upper():
                    cost = cost / 1000  # Convert from per 1M to per 1K
                
                token_key = token_type.lower().strip() if token_type else 'input'
                if 'input' in token_key or not token_key:
                    pricing['input_cost_per_1k'] = cost
                elif 'output' in token_key:
                    pricing['output_cost_per_1k'] = cost
            
            return pricing
        except Exception as e:
            logger.warning(f"Failed to parse pricing: {e}")
            return {}
    
    def _parse_benchmarks(self, benchmark_text: str) -> Dict[str, float]:
        """Parse benchmark scores from text"""
        try:
            benchmarks = {}
            import re
            
            # Look for patterns like "GPQA Diamond: 89.4%" or "SWE Bench: 74.9%"
            benchmark_pattern = r'([A-Za-z0-9\s\-\']+):\s*(\d+\.?\d*)%?'
            matches = re.findall(benchmark_pattern, benchmark_text)
            
            for match in matches:
                benchmark_name, score = match
                benchmark_name = benchmark_name.strip()
                score = float(score)
                
                # Normalize score to 0-1 range if it appears to be a percentage
                if score > 1:
                    score = score / 100
                
                benchmarks[benchmark_name] = score
            
            return benchmarks
        except Exception as e:
            logger.warning(f"Failed to parse benchmarks: {e}")
            return {}
    
    def _get_required_fields(self) -> List[str]:
        return ['benchmarks']

class AnalyticsAPIManager(DataSourceManager):
    """Manages real-time Analytics AI data"""
    
    def __init__(self):
        super().__init__("analytics")
        # Use the correct API endpoint from documentation
        self.api_url = 'https://artificialanalysis.ai/api/v2/data/llms/models'
        self.api_key = os.getenv('ANALYTICS_API_KEY')
        self.timeout = 30
    
    async def fetch_data(self) -> Dict[str, Any]:
        """Fetch real-time data from Analytics AI API"""
        if not self.api_key:
            logger.warning("Analytics API key not found, skipping real-time data")
            return {}
        
        try:
            headers = {'x-api-key': self.api_key}
            timeout = aiohttp.ClientTimeout(total=self.timeout)
            
            async with aiohttp.ClientSession(timeout=timeout) as session:
                async with session.get(self.api_url, headers=headers) as response:
                    if response.status == 200:
                        api_data = await response.json()
                        
                        # Transform API data to our format
                        data = {}
                        for model in api_data.get('data', []):
                            model_name = self._normalize_model_name(model.get('name', ''))
                            data[model_name] = {
                                'evaluations': model.get('evaluations', {}),
                                'pricing': model.get('pricing', {}),
                                'performance': {
                                    'tokens_per_second': model.get('median_output_tokens_per_second', 0),
                                    'time_to_first_token': model.get('median_time_to_first_token_seconds', 0)
                                },
                                'metadata': {
                                    'source': 'analytics_ai',
                                    'last_updated': datetime.now().isoformat()
                                }
                            }
                        
                        self.last_update = datetime.now()
                        self.data_quality = self.calculate_quality(data)
                        
                        logger.info(f"Fetched real-time data for {len(data)} models")
                        return data
                    else:
                        logger.error(f"Analytics API returned status {response.status}")
                        return {}
                        
        except Exception as e:
            logger.error(f"Failed to fetch Analytics AI data: {e}")
            return {}
    
    def _normalize_model_name(self, name: str) -> str:
        """Normalize model name for matching"""
        name = name.lower()
        # Remove common provider prefixes
        for prefix in ['openai/', 'anthropic/', 'google/', 'meta/']:
            name = name.replace(prefix, '')
        # Remove special characters
        import re
        name = re.sub(r'[^a-z0-9]', '', name)
        return name
    
    def _get_required_fields(self) -> List[str]:
        return ['evaluations']

class ModelMatcher:
    """Intelligent model matching across data sources"""
    
    def __init__(self):
        self.similarity_threshold = config.get('processing.semantic_matching.similarity_threshold', 0.8)
    
    def match_models(self, static_data: Dict, benchmark_data: Dict, analytics_data: Dict) -> Dict[str, Dict]:
        """Match models across all data sources"""
        matched_models = {}
        used_benchmark_keys = set()
        used_analytics_keys = set()
        
        # First pass: Start with static data as the foundation
        for model_id, static_model in static_data.items():
            benchmark_match = self._find_benchmark_match(model_id, static_model, benchmark_data)
            analytics_match = self._find_analytics_match(model_id, static_model, analytics_data)
            
            matches = {
                'static': static_model,
                'benchmarks': benchmark_match,
                'analytics': analytics_match
            }
            
            matched_models[model_id] = matches
            
            # Track used keys
            if benchmark_match:
                for bench_key, bench_data in benchmark_data.items():
                    if bench_data == benchmark_match:
                        used_benchmark_keys.add(bench_key)
                        break
            
            if analytics_match:
                for analytics_key, analytics_model in analytics_data.items():
                    if analytics_model == analytics_match:
                        used_analytics_keys.add(analytics_key)
                        break
        
        # Second pass: Add standalone benchmark models (like GPT-5) that don't match static data
        for bench_key, bench_data in benchmark_data.items():
            if bench_key not in used_benchmark_keys:
                # Create a synthetic static entry based on benchmark data
                synthetic_static = self._create_synthetic_static(bench_data, bench_key)
                analytics_match = self._find_analytics_match_for_benchmark(bench_key, bench_data, analytics_data)
                
                matches = {
                    'static': synthetic_static,
                    'benchmarks': bench_data,
                    'analytics': analytics_match
                }
                
                matched_models[bench_key] = matches
                
                if analytics_match:
                    for analytics_key, analytics_model in analytics_data.items():
                        if analytics_model == analytics_match:
                            used_analytics_keys.add(analytics_key)
                            break
        
        return matched_models
    
    def _find_benchmark_match(self, model_id: str, static_model: Dict, benchmark_data: Dict) -> Optional[Dict]:
        """Find matching benchmark data for a model"""
        # Try exact ID match first
        if model_id in benchmark_data:
            return benchmark_data[model_id]
        
        # Try display name match
        display_name = static_model.get('display_name', '').lower()
        for bench_id, bench_data in benchmark_data.items():
            if bench_id.lower() in display_name or display_name in bench_id.lower():
                return bench_data
        
        return None
    
    def _find_analytics_match(self, model_id: str, static_model: Dict, analytics_data: Dict) -> Optional[Dict]:
        """Find matching analytics data for a model"""
        display_name = static_model.get('display_name', '').lower()
        normalized_display = self._normalize_for_matching(display_name)
        
        # Try various matching strategies
        for analytics_key, analytics_model in analytics_data.items():
            if (analytics_key == model_id.lower() or 
                analytics_key in normalized_display or 
                normalized_display in analytics_key):
                return analytics_model
        
        return None
    
    def _normalize_for_matching(self, name: str) -> str:
        """Normalize name for matching"""
        import re
        name = name.lower()
        name = re.sub(r'[^a-z0-9]', '', name)
        return name
    
    def _create_synthetic_static(self, bench_data: Dict, model_key: str) -> Dict:
        """Create synthetic static data from benchmark data"""
        # Extract provider from model key or benchmark data
        provider = "unknown"
        if model_key.lower().startswith(('gpt', 'openai')):
            provider = "openai"
        elif model_key.lower().startswith(('claude', 'anthropic')):
            provider = "anthropic"
        elif model_key.lower().startswith(('gemini', 'google')):
            provider = "google"
        elif model_key.lower().startswith(('llama', 'meta')):
            provider = "meta"
        
        # Get provider from benchmark data if available
        if 'provider' in bench_data:
            provider = bench_data['provider']
        
        # Create synthetic static data with reasonable defaults
        synthetic = {
            'id': model_key.lower().replace(' ', '-'),
            'provider': provider,
            'display_name': bench_data.get('display_name', model_key),
            'api_alias': model_key.lower().replace(' ', '-'),
            'open_source': False
        }
        
        # Map benchmark fields to static fields
        if 'context_window' in bench_data:
            synthetic['context_window'] = bench_data['context_window']
        elif 'context_window_tokens' in bench_data:
            synthetic['context_window'] = bench_data['context_window_tokens']
        
        # Parse pricing if available
        if 'pricing_parsed' in bench_data:
            pricing = bench_data['pricing_parsed']
            if 'input_cost_per_1k' in pricing:
                synthetic['cost_in_per_1k'] = pricing['input_cost_per_1k']
            if 'output_cost_per_1k' in pricing:
                synthetic['cost_out_per_1k'] = pricing['output_cost_per_1k']
        
        # Set reasonable defaults for missing fields
        synthetic.setdefault('context_window', 4096)
        synthetic.setdefault('cost_in_per_1k', 0.01)
        synthetic.setdefault('cost_out_per_1k', 0.02)
        synthetic.setdefault('avg_latency_ms', 2000)
        
        return synthetic
    
    def _find_analytics_match_for_benchmark(self, bench_key: str, bench_data: Dict, analytics_data: Dict) -> Optional[Dict]:
        """Find analytics match for a benchmark model"""
        # Normalize benchmark model name for matching
        normalized_bench = self._normalize_for_matching(bench_key)
        
        # Try various matching strategies
        for analytics_key, analytics_model in analytics_data.items():
            normalized_analytics = self._normalize_for_matching(analytics_key)
            
            if (normalized_analytics == normalized_bench or 
                normalized_analytics in normalized_bench or 
                normalized_bench in normalized_analytics):
                return analytics_model
        
        return None

class CategoryCalculator:
    """Calculate category scores based on multi-source data"""
    
    def calculate_scores(self, model_data: Dict) -> Dict[str, CategoryScore]:
        """Calculate category scores for a model"""
        category_scores = {}
        
        for category, category_config in CLASSIFICATION_CATEGORIES.items():
            score_data = self._calculate_category_score(model_data, category, category_config)
            if score_data:
                category_scores[category] = score_data
        
        return category_scores
    
    def _calculate_category_score(self, model_data: Dict, category: str, config: Dict) -> Optional[CategoryScore]:
        """Calculate score for a specific category"""
        contributing_scores = {}
        total_weighted_score = 0.0
        total_weight = 0.0
        
        # Process benchmark scores
        for benchmark, weight in config['benchmark_weights'].items():
            score = self._extract_benchmark_score(model_data, benchmark)
            if score is not None:
                # Apply recency and quality factors
                recency_factor = self._calculate_recency_factor(model_data, benchmark)
                quality_factor = self._get_quality_factor(model_data, benchmark)
                
                adjusted_score = score * recency_factor * quality_factor
                weighted_score = adjusted_score * weight
                
                total_weighted_score += weighted_score
                total_weight += weight
                
                contributing_scores[benchmark] = {
                    'score': score,
                    'weight': weight,
                    'source': self._get_score_source(model_data, benchmark),
                    'date': self._get_score_date(model_data, benchmark),
                    'adjusted_score': adjusted_score
                }
        
        if total_weight == 0:
            return None
        
        # Calculate final score based on formula
        final_score = self._apply_formula(
            total_weighted_score / total_weight,
            model_data,
            category,
            config['formula']
        )
        
        # Calculate confidence based on data availability
        confidence = min(total_weight / sum(config['benchmark_weights'].values()), 1.0)
        
        return CategoryScore(
            score=final_score,
            confidence=confidence,
            contributing_scores=contributing_scores,
            last_updated=datetime.now().isoformat()
        )
    
    def _extract_benchmark_score(self, model_data: Dict, benchmark: str) -> Optional[float]:
        """Extract score for a specific benchmark from model data"""
        # Try benchmarks data first
        if 'benchmarks' in model_data and model_data['benchmarks']:
            bench_data = model_data['benchmarks']
            if 'benchmarks' in bench_data and benchmark in bench_data['benchmarks']:
                score_info = bench_data['benchmarks'][benchmark]
                if isinstance(score_info, dict):
                    return score_info.get('score')
                return score_info
        
        # Try analytics data
        if 'analytics' in model_data and model_data['analytics']:
            analytics = model_data['analytics']
            if 'evaluations' in analytics and benchmark in analytics['evaluations']:
                return analytics['evaluations'][benchmark]
        
        return None
    
    def _calculate_recency_factor(self, model_data: Dict, benchmark: str) -> float:
        """Calculate recency factor for a score"""
        score_date = self._get_score_date(model_data, benchmark)
        if not score_date:
            return 0.8  # Default factor for unknown dates
        
        try:
            date_obj = datetime.fromisoformat(score_date.replace('Z', '+00:00'))
            age = datetime.now() - date_obj.replace(tzinfo=None)
            
            # Exponential decay: factor decreases over time
            days_old = age.days
            if days_old <= 30:
                return 1.0
            elif days_old <= 90:
                return 0.9
            elif days_old <= 180:
                return 0.8
            else:
                return 0.7
                
        except Exception:
            return 0.8
    
    def _get_quality_factor(self, model_data: Dict, benchmark: str) -> float:
        """Get quality factor based on data source"""
        source = self._get_score_source(model_data, benchmark)
        
        quality_factors = {
            'static': 1.0,
            'benchmarks': 0.9,
            'analytics': 0.95
        }
        
        return quality_factors.get(source, 0.8)
    
    def _get_score_source(self, model_data: Dict, benchmark: str) -> str:
        """Determine the source of a score"""
        if 'benchmarks' in model_data and model_data['benchmarks']:
            bench_data = model_data['benchmarks']
            if 'benchmarks' in bench_data and benchmark in bench_data['benchmarks']:
                return 'benchmarks'
        
        if 'analytics' in model_data and model_data['analytics']:
            analytics = model_data['analytics']
            if 'evaluations' in analytics and benchmark in analytics['evaluations']:
                return 'analytics'
        
        return 'unknown'
    
    def _get_score_date(self, model_data: Dict, benchmark: str) -> Optional[str]:
        """Get the date of a score"""
        # Try benchmarks data first
        if 'benchmarks' in model_data and model_data['benchmarks']:
            bench_data = model_data['benchmarks']
            if 'benchmarks' in bench_data and benchmark in bench_data['benchmarks']:
                score_info = bench_data['benchmarks'][benchmark]
                if isinstance(score_info, dict):
                    return score_info.get('date')
        
        # Try analytics metadata
        if 'analytics' in model_data and model_data['analytics']:
            analytics = model_data['analytics']
            if 'metadata' in analytics:
                return analytics['metadata'].get('last_updated')
        
        return None
    
    def _apply_formula(self, base_score: float, model_data: Dict, category: str, formula: str) -> float:
        """Apply category-specific formula to calculate final score"""
        static_data = model_data.get('static', {})
        
        if formula == "weighted_average":
            return base_score
        
        elif formula == "weighted_average_with_context":
            # Boost for larger context windows in reasoning tasks
            context_window = static_data.get('context_window', 4096)
            context_boost = min(0.1, (context_window - 4096) / 100000)  # Up to 10% boost
            return min(100.0, base_score + (base_score * context_boost))
        
        elif formula == "interpolated_with_bonuses":
            # Creative writing benefits from larger context and cost efficiency
            context_window = static_data.get('context_window', 4096)
            cost_per_1k = static_data.get('cost_in_per_1k', 0.01)
            
            context_bonus = min(10.0, (context_window - 4000) / 1000)  # Up to 10 points
            cost_bonus = max(0.0, 5.0 - (cost_per_1k * 500))  # Up to 5 points for cheap models
            
            return min(100.0, base_score + context_bonus + cost_bonus)
        
        elif formula == "accuracy_with_latency_penalty":
            # Question answering penalized by high latency
            latency_ms = static_data.get('avg_latency_ms', 2000)
            latency_penalty = max(0.0, (latency_ms - 1000) / 10000 * base_score)  # Up to score% penalty
            return max(0.0, base_score - latency_penalty)
        
        elif formula == "conversational_with_efficiency":
            # Chat benefits from low cost and low latency
            cost_per_1k = static_data.get('cost_in_per_1k', 0.01)
            latency_ms = static_data.get('avg_latency_ms', 2000)
            
            cost_factor = 1.0 + max(0.0, (0.005 - cost_per_1k) / 0.005 * 0.2)  # Up to 20% boost
            speed_factor = 1.0 + max(0.0, (2000 - latency_ms) / 2000 * 0.1)  # Up to 10% boost
            
            return min(100.0, base_score * cost_factor * speed_factor)
        
        else:  # balanced_average and fallback
            return base_score

# Global instances
static_manager = StaticDataManager()
benchmark_manager = BenchmarkDataManager()
analytics_manager = AnalyticsAPIManager()
model_matcher = ModelMatcher()
category_calculator = CategoryCalculator()

# Global storage for consolidated models (fallback when Redis is not available)
global_consolidated_models: Dict[str, EnhancedModel] = {}

# API Models
class ConsolidationStatus(BaseModel):
    status: str
    total_models: int
    data_sources_status: Dict[str, Any]
    last_consolidation: Optional[str]
    data_quality: float

class ModelScoreResponse(BaseModel):
    model_id: str
    category_scores: Dict[str, Any]
    data_quality: float
    last_updated: str

class CategoryRankingResponse(BaseModel):
    category: str
    rankings: List[Dict[str, Any]]
    total_models: int

# API Endpoints
@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {"status": "healthy", "service": "enhanced_ingestor", "version": "2.0.0"}

@app.get("/status", response_model=ConsolidationStatus)
async def get_status():
    """Get comprehensive status of data consolidation"""
    try:
        # Get cached consolidated data count
        consolidated_count = 0
        if redis_client:
            try:
                keys = redis_client.keys("model:*")
                consolidated_count = len(keys)
            except Exception as redis_error:
                logger.warning(f"Redis error, falling back to global storage count: {redis_error}")
        
        # Fallback to global variable count
        if consolidated_count == 0:
            global global_consolidated_models
            consolidated_count = len(global_consolidated_models)
        
        # Get data source statuses
        data_sources_status = {
            "static": {
                "quality": static_manager.data_quality,
                "last_update": static_manager.last_update.isoformat() if static_manager.last_update else None
            },
            "benchmarks": {
                "quality": benchmark_manager.data_quality,
                "last_update": benchmark_manager.last_update.isoformat() if benchmark_manager.last_update else None
            },
            "analytics": {
                "quality": analytics_manager.data_quality,
                "last_update": analytics_manager.last_update.isoformat() if analytics_manager.last_update else None
            }
        }
        
        # Calculate overall data quality
        qualities = [ds["quality"] for ds in data_sources_status.values()]
        avg_quality = sum(qualities) / len(qualities) if qualities else 0.0
        
        return ConsolidationStatus(
            status="operational",
            total_models=consolidated_count,
            data_sources_status=data_sources_status,
            last_consolidation=datetime.now().isoformat(),
            data_quality=avg_quality
        )
        
    except Exception as e:
        logger.error(f"Status check failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/consolidate")
async def trigger_consolidation(background_tasks: BackgroundTasks):
    """Trigger data consolidation process"""
    background_tasks.add_task(consolidate_data)
    return {"message": "Data consolidation started", "timestamp": datetime.now().isoformat()}

@app.get("/models/{model_id}/scores", response_model=ModelScoreResponse)
async def get_model_scores(model_id: str):
    """Get category scores for a specific model"""
    try:
        # Try Redis first
        if redis_client:
            cached_data = redis_client.get(f"model:{model_id}")
            if cached_data:
                model_data = json.loads(cached_data)
                return ModelScoreResponse(
                    model_id=model_id,
                    category_scores=model_data.get('category_scores', {}),
                    data_quality=model_data.get('overall_quality', 0.0),
                    last_updated=model_data.get('last_consolidated', '')
                )
        
        # Fallback to global variable
        global global_consolidated_models
        if model_id in global_consolidated_models:
            model_data = global_consolidated_models[model_id]
            return ModelScoreResponse(
                model_id=model_id,
                category_scores=model_data.category_scores,
                data_quality=model_data.overall_quality,
                last_updated=model_data.last_consolidated
            )
        
        raise HTTPException(status_code=404, detail="Model not found")
        
    except Exception as e:
        logger.error(f"Failed to get model scores: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/rankings/{category}", response_model=CategoryRankingResponse)
async def get_category_rankings(category: str, limit: int = 10):
    """Get top models ranked by category"""
    try:
        if category not in CLASSIFICATION_CATEGORIES:
            raise HTTPException(status_code=400, detail="Invalid category")
        
        ranked_models = []
        
        # Try Redis first
        if redis_client:
            try:
                model_keys = redis_client.keys("model:*")
                for key in model_keys:
                    model_data = json.loads(redis_client.get(key))
                    category_scores = model_data.get('category_scores', {})
                    
                    if category in category_scores:
                        score_data = category_scores[category]
                        ranked_models.append({
                            'model_id': model_data['model_id'],
                            'display_name': model_data.get('display_name', ''),
                            'provider': model_data.get('provider', ''),
                            'score': score_data['score'],
                            'confidence': score_data['confidence']
                        })
            except Exception as redis_error:
                logger.warning(f"Redis error, falling back to global storage: {redis_error}")
        
        # Fallback to global variable
        if not ranked_models:
            global global_consolidated_models
            for model_id, model_data in global_consolidated_models.items():
                if category in model_data.category_scores:
                    score_data = model_data.category_scores[category]
                    ranked_models.append({
                        'model_id': model_id,
                        'display_name': model_data.display_name,
                        'provider': model_data.provider,
                        'score': score_data['score'],
                        'confidence': score_data['confidence']
                    })
        
        # Sort by score (descending) and limit results
        ranked_models.sort(key=lambda x: x['score'], reverse=True)
        limited_results = ranked_models[:limit]
        
        return CategoryRankingResponse(
            category=category,
            rankings=limited_results,
            total_models=len(ranked_models)
        )
        
    except Exception as e:
        logger.error(f"Failed to get rankings: {e}")
        raise HTTPException(status_code=500, detail=str(e))

async def consolidate_data():
    """Main data consolidation process"""
    try:
        logger.info("Starting data consolidation process...")
        
        # Fetch data from all sources
        static_data = await static_manager.fetch_data()
        benchmark_data = await benchmark_manager.fetch_data()
        analytics_data = await analytics_manager.fetch_data()
        
        # Match models across sources
        matched_models = model_matcher.match_models(static_data, benchmark_data, analytics_data)
        
        # Process each model
        consolidated_models = {}
        for model_id, model_sources in matched_models.items():
            try:
                # Calculate category scores
                category_scores = category_calculator.calculate_scores(model_sources)
                
                if not category_scores:
                    continue  # Skip models with no calculable scores
                
                # Create enhanced model data
                enhanced_model = EnhancedModel(
                    model_id=model_id,
                    provider=model_sources['static'].get('provider', 'unknown'),
                    display_name=model_sources['static'].get('display_name', model_id),
                    static_data=model_sources['static'],
                    category_scores={k: asdict(v) for k, v in category_scores.items()},
                    data_provenance={
                        'static': DataProvenance(
                            source=static_manager.source_name,
                            last_updated=static_manager.last_update.isoformat() if static_manager.last_update else '',
                            data_quality=static_manager.data_quality
                        ),
                        'benchmarks': DataProvenance(
                            source=benchmark_manager.source_name,
                            last_updated=benchmark_manager.last_update.isoformat() if benchmark_manager.last_update else '',
                            data_quality=benchmark_manager.data_quality
                        ),
                        'analytics': DataProvenance(
                            source=analytics_manager.source_name,
                            last_updated=analytics_manager.last_update.isoformat() if analytics_manager.last_update else '',
                            data_quality=analytics_manager.data_quality
                        )
                    },
                    performance_metadata=calculate_performance_metadata(category_scores),
                    last_consolidated=datetime.now().isoformat(),
                    overall_quality=calculate_overall_quality(model_sources)
                )
                
                consolidated_models[model_id] = enhanced_model
                
                # Cache in Redis
                if redis_client:
                    redis_client.setex(
                        f"model:{model_id}",
                        config.get('cache.policies.models', 3600),
                        json.dumps(asdict(enhanced_model))
                    )
                
            except Exception as e:
                logger.error(f"Failed to process model {model_id}: {e}")
                continue
        
        logger.info(f"Consolidated {len(consolidated_models)} models successfully")
        
        # Store in global variable for API access
        global global_consolidated_models
        global_consolidated_models = consolidated_models
        
        # Save consolidated data to file
        output_path = config.get('output.file_path', 'enhanced_models.json')
        with open(output_path, 'w') as f:
            json.dump({k: asdict(v) for k, v in consolidated_models.items()}, f, indent=2)
        
        return consolidated_models
        
    except Exception as e:
        logger.error(f"Data consolidation failed: {e}")
        raise

def calculate_performance_metadata(category_scores: Dict[str, CategoryScore]) -> Dict[str, Any]:
    """Calculate performance metadata for a model"""
    if not category_scores:
        return {}
    
    # Find best and worst categories
    scores_by_category = {cat: score.score for cat, score in category_scores.items()}
    
    best_categories = [cat for cat, score in scores_by_category.items() 
                      if score >= max(scores_by_category.values()) - 5]
    worst_categories = [cat for cat, score in scores_by_category.items() 
                       if score <= min(scores_by_category.values()) + 10]
    
    # Calculate overall performance tier
    avg_score = sum(scores_by_category.values()) / len(scores_by_category)
    performance_tier = "high" if avg_score >= 80 else "medium" if avg_score >= 60 else "low"
    
    return {
        "best_at": best_categories,
        "worst_at": worst_categories,
        "performance_tier": performance_tier,
        "overall_score": round(avg_score, 1),
        "category_breadth": len(category_scores),
        "avg_confidence": round(sum(score.confidence for score in category_scores.values()) / len(category_scores), 2)
    }

def calculate_overall_quality(model_sources: Dict[str, Any]) -> float:
    """Calculate overall data quality for a model"""
    quality_scores = []
    
    # Static data quality (high baseline)
    if model_sources.get('static'):
        quality_scores.append(0.9)
    
    # Benchmark data quality
    if model_sources.get('benchmarks'):
        quality_scores.append(0.8)
    
    # Analytics data quality
    if model_sources.get('analytics'):
        quality_scores.append(0.85)
    
    return sum(quality_scores) / len(quality_scores) if quality_scores else 0.0

@app.on_event("startup")
async def startup_event():
    """Initialize the service"""
    global redis_client
    
    logger.info("Starting Enhanced Ingestor Service...")
    
    # Initialize Redis connection
    try:
        redis_host = os.getenv('REDIS_HOST', config.get('cache.redis.host', 'localhost'))
        redis_port = int(os.getenv('REDIS_PORT', config.get('cache.redis.port', 6379)))
        redis_client = redis.Redis(host=redis_host, port=redis_port, decode_responses=True)
        redis_client.ping()  # Test connection
        logger.info(f"Connected to Redis cache at {redis_host}:{redis_port}")
    except Exception as e:
        logger.warning(f"Failed to connect to Redis: {e}")
        redis_client = None
    
    # Start initial data consolidation
    asyncio.create_task(consolidate_data())

if __name__ == "__main__":
    import uvicorn
    port = int(os.environ.get("PORT", 8001))
    uvicorn.run(app, host="0.0.0.0", port=port)