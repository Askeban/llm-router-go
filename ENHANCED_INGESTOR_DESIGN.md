# Enhanced Layered Data Ingestor Design

## 🎯 Objective
Create an intelligent multi-source data consolidation system that combines static, scraped, and real-time data to provide quantified model capabilities for multi-category classification.

## 📊 Data Sources Analysis

### Layer 1: Static Foundation (Priority: 1, Weight: 0.3)
**Source**: `internal/models.json`
**Provides**: 
- Model metadata (id, provider, display_name)
- Technical specs (context_window, pricing, latency)
- Provider information
- Basic capabilities flags

**Data Quality**: ✅ High (manually curated)
**Update Frequency**: Manual/Infrequent
**Trust Level**: Baseline foundation

### Layer 2: Scraped Benchmarks (Priority: 2, Weight: 0.4)
**Source**: `configs/trun_data_2024.json` + future scraped data
**Provides**:
- Performance scores by category:
  - **coding**: humaneval, livecodebench
  - **reasoning**: arc_challenge, winogrande  
  - **math**: gsm8k, math benchmarks
  - **knowledge**: mmlu, truthfulqa
  - **language**: hellaswag
- Evaluation dates and methodology
- Historical performance trends

**Data Quality**: ⚠️ Medium (scraped, needs validation)
**Update Frequency**: Daily/Weekly
**Trust Level**: Historical validation

### Layer 3: Real-time Analytics (Priority: 3, Weight: 0.3)
**Source**: Analytics AI API (`artificialanalysis.ai`)
**Provides**:
- Live performance indices:
  - `artificial_analysis_intelligence_index` → reasoning
  - `artificial_analysis_coding_index` → coding  
  - `artificial_analysis_math_index` → math
- Real-time pricing and latency
- Market performance trends
- New model detection

**Data Quality**: 🔄 Variable (API dependent)
**Update Frequency**: Real-time/Hourly
**Trust Level**: Current market state

## 🧠 Category Mapping & Quantification System

### Classification Categories → Data Points Mapping

```yaml
coding:
  static_indicators: ["api_alias contains 'code'", "tags contains 'programming'"]
  benchmark_scores:
    - humaneval (primary: weight 1.0)
    - livecodebench (secondary: weight 0.8)
    - artificial_analysis_coding_index (live: weight 0.9)
  formula: "weighted_average(scores) * confidence_factor * recency_boost"
  
math:
  static_indicators: ["capabilities.math > 0.5"]
  benchmark_scores:
    - gsm8k (primary: weight 1.0)
    - artificial_analysis_math_index (live: weight 0.9)
    - math (scraped: weight 0.7)
  formula: "weighted_average(scores) * confidence_factor"

reasoning:
  static_indicators: ["context_window > 8000", "capabilities.reasoning > 0.6"]
  benchmark_scores:
    - mmlu (primary: weight 1.0)
    - mmlu_pro (enhanced: weight 1.1)
    - arc_challenge (logical: weight 0.8)
    - artificial_analysis_intelligence_index (live: weight 0.9)
    - gpqa (expert: weight 0.7)
  formula: "weighted_average(scores) * context_boost"

creative_writing:
  static_indicators: ["context_window > 4000", "cost_per_token < 0.01"]
  benchmark_scores:
    - hellaswag (language: weight 0.6)
    - truthfulqa (creativity proxy: weight 0.4)
  formula: "interpolated_score + context_bonus + cost_efficiency_bonus"

general:
  static_indicators: ["always_applicable"]
  benchmark_scores:
    - mmlu (general knowledge: weight 0.8)
    - hellaswag (language understanding: weight 0.6)
  formula: "balanced_average(all_scores) * availability_factor"

question:
  static_indicators: ["avg_latency_ms < 3000"]
  benchmark_scores:
    - mmlu (factual: weight 0.8)
    - truthfulqa (accuracy: weight 0.9)
    - artificial_analysis_intelligence_index (reasoning: weight 0.7)
  formula: "accuracy_score * (1 - latency_penalty)"

chat:
  static_indicators: ["cost_per_1k < 0.005", "avg_latency_ms < 2000"]
  benchmark_scores:
    - hellaswag (conversational: weight 0.7)
    - truthfulqa (helpfulness: weight 0.6)
  formula: "conversational_score * cost_efficiency * speed_bonus"
```

## 🔄 Data Processing Pipeline

### Stage 1: Data Ingestion & Normalization
```python
def normalize_data_sources():
    # Layer 1: Load static foundation
    static_data = load_static_models()
    
    # Layer 2: Load and parse benchmark data
    benchmark_data = {}
    for file in glob("configs/trun*.json"):
        benchmark_data.update(load_benchmark_file(file))
    
    # Layer 3: Fetch real-time analytics
    analytics_data = fetch_analytics_ai(api_key)
    
    return static_data, benchmark_data, analytics_data
```

### Stage 2: Model Identification & Matching
```python
def match_models_across_sources(static, benchmarks, analytics):
    """
    Intelligent model matching using:
    1. Exact ID match
    2. Canonical name matching (removing provider prefixes)
    3. Fuzzy string similarity
    4. Semantic embedding similarity (using sentence transformers)
    """
    matched_models = {}
    
    for model_id in static.keys():
        matches = {
            'static': static[model_id],
            'benchmarks': find_benchmark_match(model_id, benchmarks),
            'analytics': find_analytics_match(model_id, analytics)
        }
        matched_models[model_id] = matches
    
    return matched_models
```

### Stage 3: Category Score Calculation
```python
def calculate_category_scores(model_data):
    """
    Calculate scores for each classification category using:
    1. Weighted benchmark scores
    2. Recency decay factors
    3. Data quality confidence
    4. Source reliability weighting
    """
    category_scores = {}
    
    for category in CLASSIFICATION_CATEGORIES:
        score = 0.0
        total_weight = 0.0
        
        # Process each data source with appropriate weights
        for source, weight in [('static', 0.3), ('benchmarks', 0.4), ('analytics', 0.3)]:
            source_score = extract_category_score(model_data[source], category)
            if source_score is not None:
                # Apply recency decay
                age_factor = calculate_recency_factor(model_data[source].get('date'))
                # Apply data quality factor
                quality_factor = get_quality_factor(source, model_data[source])
                
                adjusted_score = source_score * age_factor * quality_factor
                score += adjusted_score * weight
                total_weight += weight
        
        # Normalize and apply confidence
        if total_weight > 0:
            normalized_score = score / total_weight
            confidence = min(total_weight / 1.0, 1.0)  # Full confidence needs all sources
            category_scores[category] = {
                'score': normalized_score,
                'confidence': confidence,
                'data_quality': calculate_data_quality(model_data)
            }
    
    return category_scores
```

### Stage 4: Quality Assurance & Validation
```python
def validate_consolidated_data(models):
    """
    Multi-layer validation:
    1. Required field presence
    2. Score range validation (0-100)
    3. Cross-source consistency checks
    4. Anomaly detection using statistical methods
    5. Semantic consistency validation
    """
    validated_models = {}
    quality_reports = {}
    
    for model_id, model_data in models.items():
        # Basic validation
        validation_result = validate_model_data(model_data)
        
        # Anomaly detection
        anomaly_score = detect_anomalies(model_data, global_distribution)
        
        # Cross-reference validation
        consistency_score = check_cross_source_consistency(model_data)
        
        if validation_result.is_valid and anomaly_score < 0.3:
            validated_models[model_id] = model_data
            quality_reports[model_id] = {
                'validation_score': validation_result.score,
                'anomaly_score': anomaly_score,
                'consistency_score': consistency_score,
                'data_completeness': calculate_completeness(model_data)
            }
    
    return validated_models, quality_reports
```

## 📊 Enhanced Model Output Format

```json
{
  "model_id": "gpt-4",
  "provider": "openai",
  "display_name": "GPT-4",
  "static_data": {
    "context_window": 8192,
    "cost_in_per_1k": 0.03,
    "cost_out_per_1k": 0.06,
    "avg_latency_ms": 2000,
    "open_source": false
  },
  "category_scores": {
    "coding": {
      "score": 78.5,
      "confidence": 0.95,
      "contributing_scores": {
        "humaneval": {"score": 67.0, "weight": 1.0, "source": "benchmarks", "date": "2024-01-15"},
        "livecodebench": {"score": 67.0, "weight": 0.8, "source": "benchmarks", "date": "2024-01-15"},
        "artificial_analysis_coding_index": {"score": 78.5, "weight": 0.9, "source": "analytics", "date": "2024-09-07"}
      }
    },
    "math": {
      "score": 89.2,
      "confidence": 0.90,
      "contributing_scores": {
        "gsm8k": {"score": 92.0, "weight": 1.0, "source": "benchmarks", "date": "2024-01-15"},
        "artificial_analysis_math_index": {"score": 82.1, "weight": 0.9, "source": "analytics", "date": "2024-09-07"}
      }
    },
    "reasoning": {
      "score": 86.8,
      "confidence": 0.98,
      "contributing_scores": {
        "mmlu": {"score": 86.4, "weight": 1.0, "source": "benchmarks", "date": "2024-01-15"},
        "arc_challenge": {"score": 96.3, "weight": 0.8, "source": "benchmarks", "date": "2024-01-15"},
        "artificial_analysis_intelligence_index": {"score": 85.2, "weight": 0.9, "source": "analytics", "date": "2024-09-07"}
      }
    }
  },
  "data_provenance": {
    "static_data": {
      "source": "internal/models.json",
      "last_updated": "2024-08-15T10:00:00Z",
      "data_quality": 1.0
    },
    "scraped_data": {
      "source": "configs/trun_data_2024.json",
      "last_updated": "2024-01-15T10:00:00Z",
      "data_quality": 0.85
    },
    "api_data": {
      "source": "artificialanalysis.ai",
      "last_updated": "2024-09-07T18:30:00Z",
      "data_quality": 0.92
    },
    "last_consolidated": "2024-09-07T18:45:00Z",
    "data_quality": 0.92
  },
  "performance_metadata": {
    "best_at": ["reasoning", "math"],
    "worst_at": ["creative_writing"],
    "cost_efficiency": "low",
    "speed_tier": "medium",
    "overall_rank": 2
  }
}
```

## 🔧 Implementation Architecture

### Service Components

1. **Data Source Managers**
   - `StaticDataManager`: Handles models.json loading
   - `BenchmarkManager`: Processes trun*.json files
   - `AnalyticsAPIManager`: Real-time API integration

2. **Processing Engine**
   - `ModelMatcher`: Intelligent cross-source matching
   - `CategoryCalculator`: Multi-source score calculation
   - `QualityValidator`: Data validation and anomaly detection

3. **Cache & Storage**
   - `RedisCache`: Real-time data caching
   - `ConsolidationStore`: Final processed data storage
   - `MetricsStore`: Performance tracking

4. **API Endpoints**
   - `/ingest/sync` - Trigger full synchronization
   - `/ingest/status` - System health and data quality
   - `/models/{id}/scores` - Category scores for specific model
   - `/models/rankings/{category}` - Ranked models by category

### Data Flow
```
Static Data (models.json) ─┐
                           ├─▶ Model Matcher ─▶ Category Calculator ─▶ Quality Validator ─▶ Final Output
Benchmark Data (trun*.json) ─┤                                                                      ▲
                           │                                                                        │
Analytics AI API ──────────┘                                                                      │
                                                                                                   │
                           Redis Cache ◄──────────────────────────────────────────────────────────┘
```

## 🎯 Key Benefits

1. **Multi-Source Intelligence**: Combines static reliability with real-time accuracy
2. **Category-Specific Scoring**: Tailored metrics for each classification category
3. **Quality Assurance**: Multi-layer validation and anomaly detection
4. **Recency Weighting**: Recent data gets higher influence
5. **Confidence Tracking**: Transparency in data quality and completeness
6. **Semantic Matching**: Intelligent model identification across sources
7. **Scalable Architecture**: Easy to add new data sources and categories

## 📈 Performance Optimizations

1. **Incremental Updates**: Only process changed data
2. **Intelligent Caching**: Redis-based caching with TTL
3. **Parallel Processing**: Concurrent data source fetching
4. **Batch Operations**: Efficient database operations
5. **Background Sync**: Non-blocking data updates
6. **Quality Thresholds**: Skip low-quality data sources