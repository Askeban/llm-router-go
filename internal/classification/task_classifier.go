package classification

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/Askeban/llm-router-go/internal/recommendation"
)

// TaskClassifier analyzes user prompts to determine task type, category, and complexity
type TaskClassifier struct {
	// Pattern matchers for different task types and categories
	patterns map[string]map[string][]*regexp.Regexp
	
	// Complexity indicators
	complexityIndicators map[string][]string
}

// ClassificationResult represents the analysis of a user prompt
type ClassificationResult struct {
	TaskType           string                 `json:"task_type"`
	Category           string                 `json:"category"`
	Complexity         string                 `json:"complexity"`
	Priority           string                 `json:"priority"`
	Requirements       map[string]interface{} `json:"requirements"`
	Confidence         float64                `json:"confidence"`
	DetectedKeywords   []string               `json:"detected_keywords"`
	ReasoningSteps     []string               `json:"reasoning_steps"`
}

func NewTaskClassifier() *TaskClassifier {
	tc := &TaskClassifier{
		patterns:             make(map[string]map[string][]*regexp.Regexp),
		complexityIndicators: make(map[string][]string),
	}
	
	tc.initializePatterns()
	tc.initializeComplexityIndicators()
	
	return tc
}

func (tc *TaskClassifier) initializePatterns() {
	// Initialize task type patterns
	tc.patterns["task_type"] = make(map[string][]*regexp.Regexp)
	
	// Image generation patterns (put first to prioritize over text)
	tc.patterns["task_type"]["image"] = []*regexp.Regexp{
		// High priority - explicit image generation phrases
		regexp.MustCompile(`(?i)\b(generate|create|make|draw|paint|sketch|render|design|produce)\s+.*\b(image|picture|photo|visual|graphic|artwork|illustration)\b`),
		regexp.MustCompile(`(?i)\b(create|generate|make)\s+.*\b(marketing|promotional|advertising)\s+.*\b(image|visual|graphic)\b`),
		// Medium priority - image-related nouns
		regexp.MustCompile(`(?i)\b(image|picture|photo|drawing|illustration|artwork|visual|graphic|design)\b`),
		regexp.MustCompile(`(?i)\b(logo|poster|banner|icon|avatar|thumbnail|wallpaper|background)\b`),
		regexp.MustCompile(`(?i)\b(photorealistic|artistic|cartoon|3d|digital\s+art|concept\s+art)\b`),
		// Specific image tasks
		regexp.MustCompile(`(?i)\b(professional|marketing|social\s+media|instagram|facebook)\s+.*\b(image|visual|photo|graphic)\b`),
	}
	
	// Text task patterns (refined to avoid conflict with image generation)
	tc.patterns["task_type"]["text"] = []*regexp.Regexp{
		// Pure text tasks without visual components
		regexp.MustCompile(`(?i)\b(write|compose|draft|explain|analyze|summarize|translate|answer|help|assist)\b`),
		regexp.MustCompile(`(?i)\b(code|program|script|function|algorithm|debug|fix|review)\b`),
		regexp.MustCompile(`(?i)\b(math|calculate|solve|equation|formula|compute|statistics)\b`),
		regexp.MustCompile(`(?i)\b(essay|article|blog|story|report|email|letter|documentation|content)\b`),
		// Text-specific generation
		regexp.MustCompile(`(?i)\b(generate|create)\s+.*\b(text|content|copy|description|caption)\b`),
	}
	
	// Video generation patterns
	tc.patterns["task_type"]["video"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(video|movie|clip|animation|film|commercial|trailer)\b`),
		regexp.MustCompile(`(?i)\b(generate|create|make|produce)\s+.*\b(video|animation)\b`),
		regexp.MustCompile(`(?i)\b(cinematic|motion|sequence|scene|footage)\b`),
	}
	
	// Audio generation patterns
	tc.patterns["task_type"]["audio"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(audio|sound|voice|speech|music|song|narration)\b`),
		regexp.MustCompile(`(?i)\b(generate|create|make|synthesize)\s+.*\b(audio|voice|music)\b`),
		regexp.MustCompile(`(?i)\b(voiceover|podcast|jingle|soundbite|tts|text.to.speech)\b`),
	}
	
	// Initialize category patterns
	tc.patterns["category"] = make(map[string][]*regexp.Regexp)
	
	// Coding category
	tc.patterns["category"]["coding"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(code|program|script|function|class|method|algorithm|api|framework|library)\b`),
		regexp.MustCompile(`(?i)\b(python|javascript|java|go|rust|c\+\+|typescript|react|node|django)\b`),
		regexp.MustCompile(`(?i)\b(debug|fix|optimize|refactor|implement|develop|build|deploy)\b`),
		regexp.MustCompile(`(?i)\b(database|sql|rest|graphql|docker|kubernetes|git)\b`),
	}
	
	// Math category  
	tc.patterns["category"]["math"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(math|mathematics|calculate|solve|equation|formula|algebra|calculus|statistics)\b`),
		regexp.MustCompile(`(?i)\b(integral|derivative|matrix|vector|probability|geometry|trigonometry)\b`),
		regexp.MustCompile(`(?i)\b(compute|numerical|analytical|mathematical|quantitative)\b`),
	}
	
	// Reasoning category
	tc.patterns["category"]["reasoning"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(analyze|reason|logic|evaluate|assess|compare|contrast|examine)\b`),
		regexp.MustCompile(`(?i)\b(argument|evidence|conclusion|inference|deduction|critical thinking)\b`),
		regexp.MustCompile(`(?i)\b(problem solving|decision|strategy|planning|optimization)\b`),
	}
	
	// Writing category
	tc.patterns["category"]["writing"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(write|compose|draft|create|author|edit|proofread|rewrite)\b`),
		regexp.MustCompile(`(?i)\b(essay|article|blog|story|report|email|letter|content|copy)\b`),
		regexp.MustCompile(`(?i)\b(creative writing|technical writing|copywriting|journalism)\b`),
	}
	
	// Analysis category
	tc.patterns["category"]["analysis"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(analyze|analysis|examine|evaluate|assess|review|investigate)\b`),
		regexp.MustCompile(`(?i)\b(data|trend|pattern|insight|interpretation|conclusion)\b`),
		regexp.MustCompile(`(?i)\b(research|study|survey|findings|results|metrics)\b`),
	}
	
	// Creative category (for generative tasks)
	tc.patterns["category"]["creative"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(creative|artistic|imaginative|original|unique|innovative)\b`),
		regexp.MustCompile(`(?i)\b(art|design|style|aesthetic|beautiful|colorful|abstract)\b`),
	}
	
	// Photorealistic category (for images)
	tc.patterns["category"]["photorealistic"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(photorealistic|realistic|photo.realistic|lifelike|natural)\b`),
		regexp.MustCompile(`(?i)\b(professional|high.quality|detailed|sharp|clear)\b`),
	}
}

func (tc *TaskClassifier) initializeComplexityIndicators() {
	// Simple complexity indicators
	tc.complexityIndicators["simple"] = []string{
		"simple", "basic", "easy", "quick", "short", "brief", "straightforward",
		"beginner", "intro", "getting started", "hello world", "tutorial",
	}
	
	// Medium complexity indicators
	tc.complexityIndicators["medium"] = []string{
		"medium", "intermediate", "moderate", "standard", "typical", "regular",
		"multi-step", "detailed", "comprehensive", "thorough",
	}
	
	// Hard complexity indicators
	tc.complexityIndicators["hard"] = []string{
		"hard", "difficult", "complex", "advanced", "challenging", "sophisticated",
		"enterprise", "production", "scalable", "optimized", "performance",
		"multi-threaded", "distributed", "microservices", "machine learning",
	}
	
	// Expert complexity indicators
	tc.complexityIndicators["expert"] = []string{
		"expert", "professional", "enterprise-grade", "research-level", "cutting-edge",
		"state-of-the-art", "highly optimized", "custom", "specialized",
		"architectural", "system design", "distributed systems", "high-performance",
	}
}

// ClassifyPrompt analyzes a user prompt and returns classification results
func (tc *TaskClassifier) ClassifyPrompt(prompt string) ClassificationResult {
	result := ClassificationResult{
		Requirements:     make(map[string]interface{}),
		DetectedKeywords: []string{},
		ReasoningSteps:   []string{},
	}
	
	promptLower := strings.ToLower(prompt)
	
	// Step 1: Determine task type
	taskType, taskTypeConfidence := tc.classifyTaskType(prompt, promptLower)
	result.TaskType = taskType
	result.ReasoningSteps = append(result.ReasoningSteps, 
		fmt.Sprintf("Identified task type '%s' with %.2f confidence", taskType, taskTypeConfidence))
	
	// Step 2: Determine category
	category, categoryConfidence := tc.classifyCategory(prompt, promptLower, taskType)
	result.Category = category
	result.ReasoningSteps = append(result.ReasoningSteps, 
		fmt.Sprintf("Identified category '%s' with %.2f confidence", category, categoryConfidence))
	
	// Step 3: Determine complexity
	complexity, complexityConfidence := tc.classifyComplexity(prompt, promptLower)
	result.Complexity = complexity
	result.ReasoningSteps = append(result.ReasoningSteps, 
		fmt.Sprintf("Identified complexity '%s' with %.2f confidence", complexity, complexityConfidence))
	
	// Step 4: Determine priority from context
	priority := tc.inferPriority(prompt, promptLower)
	result.Priority = priority
	if priority != "balanced" {
		result.ReasoningSteps = append(result.ReasoningSteps, 
			fmt.Sprintf("Inferred priority '%s' from context", priority))
	}
	
	// Step 5: Extract special requirements
	requirements := tc.extractRequirements(prompt, promptLower)
	result.Requirements = requirements
	if len(requirements) > 0 {
		result.ReasoningSteps = append(result.ReasoningSteps, 
			fmt.Sprintf("Extracted %d special requirements", len(requirements)))
	}
	
	// Step 6: Calculate overall confidence
	result.Confidence = (taskTypeConfidence + categoryConfidence + complexityConfidence) / 3.0
	
	// Step 7: Extract detected keywords
	result.DetectedKeywords = tc.extractKeywords(prompt, promptLower)
	
	return result
}

func (tc *TaskClassifier) classifyTaskType(prompt, promptLower string) (string, float64) {
	scores := make(map[string]float64)
	
	// Check patterns for each task type
	for taskType, patterns := range tc.patterns["task_type"] {
		score := 0.0
		for _, pattern := range patterns {
			matches := pattern.FindAllString(prompt, -1)
			score += float64(len(matches)) * 0.2
		}
		scores[taskType] = score
	}
	
	// Handle visual-conflicting text patterns: if image patterns found, reduce text scores for conflicting patterns
	if scores["image"] > 0.2 {
		// Check if the prompt contains visual terms that might trigger false text matches
		promptLower := strings.ToLower(prompt)
		visualTerms := []string{"image", "picture", "photo", "visual", "graphic", "marketing image", "generate image"}
		hasVisualTerms := false
		for _, term := range visualTerms {
			if strings.Contains(promptLower, term) {
				hasVisualTerms = true
				break
			}
		}
		
		// If we have visual terms AND image patterns, reduce conflicting text score
		if hasVisualTerms {
			scores["text"] = scores["text"] * 0.3 // Significantly reduce text score
		}
	}
	
	// Special logic for multimodal detection - only if BOTH image and text have significant scores
	if scores["image"] > 0.4 && scores["text"] > 0.4 {
		scores["multimodal"] = scores["image"] + scores["text"]*0.5
	}
	
	// Prioritize pure image generation over multimodal for clear image prompts
	if scores["image"] > 0.4 && scores["text"] < 0.3 {
		scores["image"] = scores["image"] * 1.5 // Boost pure image tasks
	}
	
	// Find the highest scoring task type
	maxScore := 0.0
	selectedType := "text" // default
	
	for taskType, score := range scores {
		if score > maxScore {
			maxScore = score
			selectedType = taskType
		}
	}
	
	// If no clear match, default to text with lower confidence
	if maxScore == 0 {
		return "text", 0.3
	}
	
	// Normalize confidence (cap at 1.0)
	confidence := math.Min(maxScore, 1.0)
	return selectedType, confidence
}

func (tc *TaskClassifier) classifyCategory(prompt, promptLower, taskType string) (string, float64) {
	scores := make(map[string]float64)
	
	// Check patterns for each category
	for category, patterns := range tc.patterns["category"] {
		score := 0.0
		for _, pattern := range patterns {
			matches := pattern.FindAllString(prompt, -1)
			score += float64(len(matches)) * 0.3
		}
		scores[category] = score
	}
	
	// Apply task type specific logic
	if taskType == "image" || taskType == "video" || taskType == "audio" || taskType == "multimodal" {
		// For generative tasks, boost creative and photorealistic categories
		if scores["creative"] == 0 && scores["photorealistic"] == 0 {
			// Default to creative for generative tasks
			scores["creative"] = 0.6
		}
		// Boost existing creative/photorealistic scores
		if scores["creative"] > 0 {
			scores["creative"] = scores["creative"] * 1.5
		}
		if scores["photorealistic"] > 0 {
			scores["photorealistic"] = scores["photorealistic"] * 1.5
		}
	}
	
	// Find the highest scoring category
	maxScore := 0.0
	selectedCategory := tc.getDefaultCategory(taskType)
	
	for category, score := range scores {
		if score > maxScore {
			maxScore = score
			selectedCategory = category
		}
	}
	
	confidence := math.Min(maxScore, 1.0)
	if confidence == 0 {
		confidence = 0.4 // Default confidence for category
	}
	
	return selectedCategory, confidence
}

func (tc *TaskClassifier) getDefaultCategory(taskType string) string {
	switch taskType {
	case "text":
		return "writing"
	case "image":
		return "creative"
	case "video":
		return "creative"
	case "audio":
		return "creative"
	default:
		return "writing"
	}
}

func (tc *TaskClassifier) classifyComplexity(prompt, promptLower string) (string, float64) {
	scores := make(map[string]int)
	
	// Count indicators for each complexity level
	for complexity, indicators := range tc.complexityIndicators {
		count := 0
		for _, indicator := range indicators {
			if strings.Contains(promptLower, indicator) {
				count++
			}
		}
		scores[complexity] = count
	}
	
	// Additional heuristics for complexity
	
	// Length-based heuristic
	wordCount := len(strings.Fields(prompt))
	if wordCount > 100 {
		scores["hard"] += 1
	} else if wordCount > 50 {
		scores["medium"] += 1
	} else {
		scores["simple"] += 1
	}
	
	// Technical depth heuristic
	technicalTerms := []string{
		"architecture", "framework", "optimization", "scalability", "performance",
		"distributed", "microservices", "kubernetes", "machine learning", "api",
		"algorithm", "data structure", "design pattern", "best practices",
	}
	technicalCount := 0
	for _, term := range technicalTerms {
		if strings.Contains(promptLower, term) {
			technicalCount++
		}
	}
	
	if technicalCount >= 3 {
		scores["expert"] += 2
	} else if technicalCount >= 2 {
		scores["hard"] += 1
	}
	
	// Find the highest scoring complexity
	maxScore := 0
	selectedComplexity := "medium" // default
	
	for complexity, score := range scores {
		if score > maxScore {
			maxScore = score
			selectedComplexity = complexity
		}
	}
	
	// Calculate confidence based on score and indicators found
	confidence := 0.5 // base confidence
	if maxScore > 0 {
		confidence = math.Min(0.3 + float64(maxScore)*0.2, 1.0)
	}
	
	return selectedComplexity, confidence
}

func (tc *TaskClassifier) inferPriority(prompt, promptLower string) string {
	// Check for explicit priority indicators
	if strings.Contains(promptLower, "fast") || strings.Contains(promptLower, "quick") || 
	   strings.Contains(promptLower, "speed") || strings.Contains(promptLower, "urgent") {
		return "speed"
	}
	
	if strings.Contains(promptLower, "cheap") || strings.Contains(promptLower, "cost") || 
	   strings.Contains(promptLower, "budget") || strings.Contains(promptLower, "free") {
		return "cost"
	}
	
	if strings.Contains(promptLower, "best") || strings.Contains(promptLower, "high quality") || 
	   strings.Contains(promptLower, "professional") || strings.Contains(promptLower, "excellent") {
		return "quality"
	}
	
	return "balanced" // default
}

func (tc *TaskClassifier) extractRequirements(prompt, promptLower string) map[string]interface{} {
	requirements := make(map[string]interface{})
	
	// Open source requirement
	if strings.Contains(promptLower, "open source") || strings.Contains(promptLower, "opensource") {
		requirements["open_source"] = true
	}
	
	// Free tier requirement
	if strings.Contains(promptLower, "free") && !strings.Contains(promptLower, "free form") {
		requirements["free_tier"] = true
	}
	
	// Speed requirements
	if strings.Contains(promptLower, "fast") || strings.Contains(promptLower, "speed") {
		requirements["min_speed"] = 50.0 // tokens per second
	}
	
	// Quality requirements
	if strings.Contains(promptLower, "high quality") || strings.Contains(promptLower, "professional") {
		requirements["min_quality"] = 0.85
	}
	
	// Image-specific requirements
	if strings.Contains(promptLower, "high resolution") || strings.Contains(promptLower, "4k") {
		requirements["high_resolution"] = true
	}
	
	if strings.Contains(promptLower, "style control") || strings.Contains(promptLower, "specific style") {
		requirements["style_control"] = true
	}
	
	// Video-specific requirements
	durationMatch := regexp.MustCompile(`(\d+)\s*(second|minute|min)s?`).FindStringSubmatch(promptLower)
	if len(durationMatch) > 0 {
		requirements["duration"] = durationMatch[1] + " " + durationMatch[2]
	}
	
	return requirements
}

func (tc *TaskClassifier) extractKeywords(prompt, promptLower string) []string {
	// Extract meaningful keywords from the prompt
	keywords := []string{}
	
	// Technical keywords
	techKeywords := []string{
		"python", "javascript", "go", "rust", "react", "node", "api", "database",
		"machine learning", "ai", "neural network", "deep learning",
		"web development", "mobile app", "frontend", "backend",
		"docker", "kubernetes", "cloud", "aws", "azure",
	}
	
	for _, keyword := range techKeywords {
		if strings.Contains(promptLower, keyword) {
			keywords = append(keywords, keyword)
		}
	}
	
	// Extract quoted phrases (likely important)
	quotedPattern := regexp.MustCompile(`"([^"]*)"`)
	matches := quotedPattern.FindAllStringSubmatch(prompt, -1)
	for _, match := range matches {
		if len(match) > 1 && len(match[1]) > 2 {
			keywords = append(keywords, match[1])
		}
	}
	
	return keywords
}

// ConvertToRecommendationRequest converts classification result to recommendation request
func (tc *TaskClassifier) ConvertToRecommendationRequest(classification ClassificationResult, context string) recommendation.RecommendationRequest {
	return recommendation.RecommendationRequest{
		TaskType:     classification.TaskType,
		Category:     classification.Category,
		Complexity:   classification.Complexity,
		Priority:     classification.Priority,
		Requirements: classification.Requirements,
		Context:      context,
	}
}