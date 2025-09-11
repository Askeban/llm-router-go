import { useState } from 'react'
import { Send, Bot, DollarSign, Clock, CheckCircle, AlertCircle } from 'lucide-react'

const EXAMPLE_PROMPTS = [
  {
    category: "Agentic Pipeline",
    prompt: "Plan a complex customer agentic pipeline: fetch Polaris issues, analyze them, create tickets, understand codebase, implement solutions, test, and create PRs. Need strategic planning with cost optimization."
  },
  {
    category: "Code Review",
    prompt: "Review this production React component for security vulnerabilities, performance issues, and best practices. Provide specific recommendations for improvements."
  },
  {
    category: "Data Analysis", 
    prompt: "Analyze large dataset of customer behavior patterns and create statistical insights with confidence intervals for executive presentation."
  },
  {
    category: "Content Creation",
    prompt: "Write a comprehensive technical blog post about microservices architecture patterns with code examples and diagrams."
  }
]

const MOCK_RESPONSES = {
  "agentic": {
    model: "Claude 3 Haiku",
    cost: 0.0008,
    savings: "99.2%",
    reasoning: "Budget-optimized choice for complex multi-step workflows",
    quality: "High",
    responseTime: 420
  },
  "code": {
    model: "GPT-4o",
    cost: 0.0045,
    savings: "85.0%",
    reasoning: "Excellent code analysis capabilities with cost optimization",
    quality: "Very High", 
    responseTime: 680
  },
  "analysis": {
    model: "Claude 4",
    cost: 0.100,
    savings: "65.5%",
    reasoning: "Superior analytical reasoning for executive-level insights",
    quality: "Expert",
    responseTime: 850
  },
  "content": {
    model: "GPT-4 Turbo",
    cost: 0.045,
    savings: "70.0%",
    reasoning: "Strong technical writing with balanced cost-performance",
    quality: "High",
    responseTime: 590
  }
}

export default function InteractiveDemo() {
  const [selectedPrompt, setSelectedPrompt] = useState(EXAMPLE_PROMPTS[0])
  const [customPrompt, setCustomPrompt] = useState('')
  const [budgetPriority, setBudgetPriority] = useState(true)
  const [isLoading, setIsLoading] = useState(false)
  const [response, setResponse] = useState<any>(null)

  const handleDemo = async () => {
    setIsLoading(true)
    setResponse(null)
    
    // Simulate API call
    await new Promise(resolve => setTimeout(resolve, 1500))
    
    // Mock response based on category
    let mockKey = "agentic"
    const prompt = customPrompt || selectedPrompt.prompt
    if (prompt.toLowerCase().includes("code") || prompt.toLowerCase().includes("review")) {
      mockKey = "code"
    } else if (prompt.toLowerCase().includes("analysis") || prompt.toLowerCase().includes("data")) {
      mockKey = "analysis"  
    } else if (prompt.toLowerCase().includes("write") || prompt.toLowerCase().includes("content")) {
      mockKey = "content"
    }
    
    setResponse(MOCK_RESPONSES[mockKey as keyof typeof MOCK_RESPONSES])
    setIsLoading(false)
  }

  return (
    <div className="max-w-5xl mx-auto">
      <div className="bg-white rounded-2xl shadow-xl overflow-hidden">
        {/* Demo Controls */}
        <div className="p-8 border-b border-gray-200">
          <div className="grid md:grid-cols-2 gap-8">
            {/* Prompt Selection */}
            <div>
              <label className="block text-sm font-semibold text-gray-900 mb-3">
                Choose Example or Write Custom Prompt
              </label>
              <div className="grid grid-cols-2 gap-2 mb-4">
                {EXAMPLE_PROMPTS.map((example, index) => (
                  <button
                    key={index}
                    onClick={() => {
                      setSelectedPrompt(example)
                      setCustomPrompt('')
                    }}
                    className={`p-3 text-sm rounded-lg border transition-all ${
                      selectedPrompt === example && !customPrompt
                        ? 'border-primary-500 bg-primary-50 text-primary-700'
                        : 'border-gray-200 hover:border-gray-300'
                    }`}
                  >
                    {example.category}
                  </button>
                ))}
              </div>
              
              <textarea
                value={customPrompt}
                onChange={(e) => setCustomPrompt(e.target.value)}
                placeholder="Or write your custom prompt here..."
                className="w-full h-32 p-4 border border-gray-200 rounded-lg resize-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              />
            </div>

            {/* Settings */}
            <div>
              <label className="block text-sm font-semibold text-gray-900 mb-3">
                Optimization Priority
              </label>
              
              <div className="space-y-3 mb-6">
                <label className="flex items-center space-x-3 cursor-pointer">
                  <input
                    type="radio"
                    name="priority"
                    checked={budgetPriority}
                    onChange={() => setBudgetPriority(true)}
                    className="w-4 h-4 text-primary-600"
                  />
                  <div>
                    <div className="font-medium text-gray-900">💰 Cost-Optimized</div>
                    <div className="text-sm text-gray-600">Maximize savings while maintaining quality</div>
                  </div>
                </label>
                
                <label className="flex items-center space-x-3 cursor-pointer">
                  <input
                    type="radio"
                    name="priority"
                    checked={!budgetPriority}
                    onChange={() => setBudgetPriority(false)}
                    className="w-4 h-4 text-primary-600"
                  />
                  <div>
                    <div className="font-medium text-gray-900">🚀 Performance-First</div>
                    <div className="text-sm text-gray-600">Best quality regardless of cost</div>
                  </div>
                </label>
              </div>

              <button
                onClick={handleDemo}
                disabled={isLoading || (!customPrompt && !selectedPrompt)}
                className="w-full bg-primary-600 text-white py-3 px-6 rounded-lg font-semibold hover:bg-primary-700 disabled:bg-gray-300 disabled:cursor-not-allowed transition-all flex items-center justify-center space-x-2"
              >
                {isLoading ? (
                  <>
                    <div className="animate-spin rounded-full h-5 w-5 border-2 border-white border-t-transparent"></div>
                    <span>Analyzing...</span>
                  </>
                ) : (
                  <>
                    <Send className="h-5 w-5" />
                    <span>Get Recommendation</span>
                  </>
                )}
              </button>
            </div>
          </div>

          {/* Current Prompt Display */}
          {(customPrompt || selectedPrompt) && (
            <div className="mt-6 p-4 bg-gray-50 rounded-lg">
              <div className="text-sm font-medium text-gray-700 mb-2">Current Prompt:</div>
              <div className="text-sm text-gray-600 italic">
                "{(customPrompt || selectedPrompt.prompt).substring(0, 200)}..."
              </div>
            </div>
          )}
        </div>

        {/* Results */}
        {response && (
          <div className="p-8 bg-gradient-to-r from-green-50 to-blue-50">
            <div className="flex items-center space-x-2 mb-6">
              <CheckCircle className="h-6 w-6 text-success-500" />
              <h3 className="text-lg font-bold text-gray-900">Recommendation Generated</h3>
            </div>

            <div className="grid md:grid-cols-3 gap-6">
              {/* Selected Model */}
              <div className="bg-white p-6 rounded-xl shadow-sm">
                <div className="flex items-center space-x-3 mb-3">
                  <Bot className="h-8 w-8 text-primary-600" />
                  <div>
                    <div className="font-bold text-gray-900">{response.model}</div>
                    <div className="text-sm text-gray-600">Recommended Model</div>
                  </div>
                </div>
                <div className="text-sm text-gray-700 mb-2">
                  <strong>Quality:</strong> {response.quality}
                </div>
                <div className="text-xs text-gray-600">
                  {response.reasoning}
                </div>
              </div>

              {/* Cost Analysis */}
              <div className="bg-white p-6 rounded-xl shadow-sm">
                <div className="flex items-center space-x-3 mb-3">
                  <DollarSign className="h-8 w-8 text-success-600" />
                  <div>
                    <div className="font-bold text-gray-900">${response.cost.toFixed(4)}/1K</div>
                    <div className="text-sm text-gray-600">Estimated Cost</div>
                  </div>
                </div>
                <div className="text-lg font-bold text-success-600 mb-1">
                  {response.savings} saved
                </div>
                <div className="text-xs text-gray-600">
                  vs. premium alternatives
                </div>
              </div>

              {/* Performance */}
              <div className="bg-white p-6 rounded-xl shadow-sm">
                <div className="flex items-center space-x-3 mb-3">
                  <Clock className="h-8 w-8 text-warning-500" />
                  <div>
                    <div className="font-bold text-gray-900">{response.responseTime}ms</div>
                    <div className="text-sm text-gray-600">Response Time</div>
                  </div>
                </div>
                <div className="text-sm text-gray-700 mb-2">
                  <strong>Status:</strong> Excellent
                </div>
                <div className="text-xs text-gray-600">
                  Sub-second classification
                </div>
              </div>
            </div>

            {/* API Example */}
            <div className="mt-8 bg-gray-900 rounded-lg p-6">
              <div className="flex items-center justify-between mb-4">
                <h4 className="text-white font-semibold">API Integration Example</h4>
                <button className="text-gray-400 hover:text-white text-sm">Copy</button>
              </div>
              <pre className="text-green-400 text-sm font-mono overflow-x-auto">
{`curl -X POST https://api.llmrouter.ai/v2/recommend/smart \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "prompt": "${(customPrompt || selectedPrompt.prompt).substring(0, 80)}...",
    "budget_priority": ${budgetPriority}
  }'`}
              </pre>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}