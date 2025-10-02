# RouteLLM Ingester Services Setup Guide

## Current System Status ✅

### **What's Already Working:**
- **✅ HuggingFace Integration**: Live data from 76 models with download statistics
- **✅ Model Analytics**: Real-time quality scoring and performance metrics
- **✅ Commercial API Data**: Pricing and capability information
- **✅ Static Model Database**: 2025 latest models (GPT-5, Claude 4, etc.)

### **What's Disabled/Missing:**
- **❌ Reddit Community Data**: Model discussions, reviews, sentiment
- **❌ Twitter/X Integration**: Social media buzz and trends
- **❌ GitHub Integration**: Repository statistics and code model usage
- **❌ Model Performance Benchmarks**: Live performance data ingestion
- **❌ Community Feedback Loop**: User ratings and reviews

---

## 🔧 **Required Setup for Full Ingester Services**

### **1. Reddit API Integration**

#### **Reddit API Credentials Needed:**
```bash
# Reddit Application Registration (https://www.reddit.com/prefs/apps)
REDDIT_CLIENT_ID=your_client_id_here
REDDIT_CLIENT_SECRET=your_client_secret_here
REDDIT_USER_AGENT=RouteLLM-Bot/1.0
REDDIT_USERNAME=your_bot_username
REDDIT_PASSWORD=your_bot_password
```

#### **Setup Steps:**
1. **Create Reddit Application**:
   - Go to https://www.reddit.com/prefs/apps
   - Click "Create App" or "Create Another App"
   - Select "script" type
   - Set name: "RouteLLM Data Ingester"
   - Set description: "Ingesting community discussions about AI models"
   - Set redirect URI: `http://localhost:8080` (not used for script type)

2. **Target Subreddits for Model Data**:
   - `/r/MachineLearning` - Academic discussions
   - `/r/LocalLLaMA` - Open source model community
   - `/r/OpenAI` - GPT model discussions
   - `/r/artificial` - General AI discussions
   - `/r/ChatGPT` - User experiences
   - `/r/ArtificialIntelligence` - Industry discussions

#### **Implementation Plan:**
```go
// Add to main_deploy.go
type RedditConfig struct {
    ClientID     string
    ClientSecret string
    UserAgent    string
    Username     string
    Password     string
}

type CommunityData struct {
    ModelID     string    `json:"model_id"`
    Source      string    `json:"source"`      // "reddit", "twitter", etc.
    Sentiment   float64   `json:"sentiment"`   // -1.0 to 1.0
    Mentions    int       `json:"mentions"`
    Score       float64   `json:"score"`       // upvotes/engagement
    Timestamp   time.Time `json:"timestamp"`
    Keywords    []string  `json:"keywords"`
}
```

---

### **2. Twitter/X API Integration**

#### **Twitter API v2 Credentials Needed:**
```bash
# Twitter Developer Portal (https://developer.twitter.com)
TWITTER_BEARER_TOKEN=your_bearer_token_here
TWITTER_API_KEY=your_api_key_here
TWITTER_API_SECRET=your_api_secret_here
TWITTER_ACCESS_TOKEN=your_access_token_here
TWITTER_ACCESS_TOKEN_SECRET=your_access_token_secret_here
```

#### **Setup Steps:**
1. **Apply for Twitter Developer Account**:
   - Go to https://developer.twitter.com/en/portal/dashboard
   - Apply for Essential access (free tier)
   - Describe use case: "Analyzing AI model discussions for recommendation system"

2. **Search Queries for Model Mentions**:
   - `#GPT4` `#GPT5` `#Claude` `#Gemini`
   - `"OpenAI API"` `"Anthropic Claude"`
   - `"model comparison"` `"AI costs"`
   - `"LLM performance"`

---

### **3. GitHub API Integration**

#### **GitHub API Credentials:**
```bash
# GitHub Personal Access Token
GITHUB_TOKEN=your_personal_access_token_here
```

#### **Setup Steps:**
1. **Create Personal Access Token**:
   - Go to GitHub Settings → Developer settings → Personal access tokens
   - Generate new token with `public_repo` scope
   - Used for repository statistics and model usage data

2. **Target Data Sources**:
   - Repository stars/forks for model projects
   - README mentions of specific models
   - Issue discussions about model performance
   - Code usage patterns

---

### **4. Model Benchmark Integration**

#### **Benchmark APIs:**
```bash
# Hugging Face Leaderboards
HUGGINGFACE_API_KEY=your_huggingface_api_key_here  # Get from https://huggingface.co/settings/tokens

# Additional benchmark sources
OPENAI_API_KEY=your_openai_key_here           # For pricing updates
ANTHROPIC_API_KEY=your_anthropic_key_here     # For Claude metrics
GOOGLE_API_KEY=your_google_key_here           # For Gemini data
```

---

## 🏗️ **Implementation Architecture**

### **Ingester Service Structure:**
```go
// Ingester service (new microservice)
type IngesterService struct {
    RedditClient   *reddit.Client
    TwitterClient  *twitter.Client
    GitHubClient   *github.Client
    Database       *sql.DB
    UpdateInterval time.Duration
}

func (s *IngesterService) StartIngestion() {
    go s.ingestRedditData()
    go s.ingestTwitterData()
    go s.ingestGitHubData()
    go s.updateBenchmarks()
}
```

### **Database Schema Extensions:**
```sql
-- Community sentiment data
CREATE TABLE community_sentiment (
    id SERIAL PRIMARY KEY,
    model_id VARCHAR(255),
    source VARCHAR(50),      -- 'reddit', 'twitter', 'github'
    sentiment DECIMAL(3,2),  -- -1.00 to 1.00
    mentions INTEGER,
    score DECIMAL(10,2),
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    keywords TEXT[]
);

-- Model performance benchmarks
CREATE TABLE model_benchmarks (
    id SERIAL PRIMARY KEY,
    model_id VARCHAR(255),
    benchmark_name VARCHAR(100),
    score DECIMAL(10,4),
    rank INTEGER,
    date_recorded TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

---

## 🚀 **Quick Setup Commands**

### **1. Environment Variables Setup:**
```bash
# Add to your GCP Cloud Run environment variables
gcloud run services update llm-router-api \
  --region=us-central1 \
  --set-env-vars="REDDIT_CLIENT_ID=your_client_id,REDDIT_CLIENT_SECRET=your_secret,TWITTER_BEARER_TOKEN=your_token"
```

### **2. Database Setup (if using PostgreSQL):**
```bash
# Connect to your database and run:
psql -h your-db-host -U your-user -d routellm -f ingester_schema.sql
```

### **3. Service Deployment:**
```bash
# Deploy ingester as separate service
gcloud run deploy llm-ingester \
  --image=gcr.io/routellm-prod/llm-ingester:latest \
  --region=us-central1 \
  --memory=1Gi \
  --cpu=1 \
  --set-env-vars="REDDIT_CLIENT_ID=...,TWITTER_BEARER_TOKEN=..."
```

---

## 📊 **Expected Data Enhancement**

### **With Full Ingester Services:**

#### **Before (Current):**
- 76 models with static quality scores
- HuggingFace download numbers
- Manual model metadata

#### **After (With Ingesters):**
- **Community Sentiment**: Real user opinions (-1.0 to +1.0)
- **Social Buzz**: Twitter mentions and engagement
- **Developer Adoption**: GitHub usage patterns
- **Live Benchmarks**: Updated performance scores
- **Trend Analysis**: Rising/falling model popularity

### **Enhanced Recommendation Logic:**
```go
// Updated scoring with community data
func calculateEnhancedScore(model Model, communityData CommunityData) float64 {
    baseScore := model.Quality * 0.4
    communityScore := (communityData.Sentiment + 1.0) / 2.0 * 0.2  // Normalize to 0-1
    popularityScore := math.Log(float64(communityData.Mentions)) * 0.2
    trendScore := calculateTrendScore(model.ID) * 0.2

    return baseScore + communityScore + popularityScore + trendScore
}
```

---

## 🎯 **What You Need to Provide**

### **Immediate Requirements:**
1. **Reddit API Credentials** (15 min setup)
2. **Twitter Developer Account** (1-2 days approval)
3. **GitHub Personal Access Token** (5 min setup)
4. **Additional API Keys** (OpenAI, Anthropic, Google for live pricing)

### **Optional Enhancements:**
5. **Database Upgrade** (PostgreSQL for better analytics)
6. **Monitoring Setup** (DataDog, New Relic for ingester health)
7. **Cache Layer** (Redis for community data)

### **Development Time Estimate:**
- **Basic Reddit Integration**: 1-2 days
- **Twitter Integration**: 1 day
- **GitHub Integration**: 1 day
- **Database Schema**: 0.5 days
- **Enhanced Scoring**: 1 day
- **Testing & Deployment**: 1 day

**Total: ~1 week for full ingester system**

---

## ✅ **Priority Order**

1. **High Priority**: Reddit API (most valuable community data)
2. **Medium Priority**: GitHub API (developer sentiment)
3. **Low Priority**: Twitter API (noisy but trending data)
4. **Optional**: Live benchmark APIs

Once you provide the API credentials, I can implement the ingester services and integrate them with your existing RouteLLM system to provide much richer, community-driven model recommendations!