import { useState, useEffect } from 'react'
import { Calculator, TrendingDown, DollarSign, Zap } from 'lucide-react'

interface CalculatorResult {
  currentCost: number
  optimizedCost: number
  savings: number
  savingsPercentage: number
  paybackPeriod: number
}

const INDUSTRY_PRESETS = [
  { name: "Startup", requests: 100000, description: "Early-stage company" },
  { name: "SMB", requests: 1000000, description: "Small-medium business" },
  { name: "Enterprise", requests: 10000000, description: "Large organization" },
  { name: "Scale-up", requests: 50000000, description: "High-growth company" }
]

const USE_CASE_MULTIPLIERS = {
  "content": { cost: 0.008, description: "Content generation, writing" },
  "coding": { cost: 0.015, description: "Code generation, review" },
  "analysis": { cost: 0.025, description: "Data analysis, research" },
  "multimodal": { cost: 0.040, description: "Image, video, audio processing" },
  "agentic": { cost: 0.020, description: "Complex workflows, automation" }
}

export default function CostCalculator() {
  const [monthlyRequests, setMonthlyRequests] = useState(1000000)
  const [useCase, setUseCase] = useState("coding")
  const [currentProvider, setCurrentProvider] = useState("openai")
  const [result, setResult] = useState<CalculatorResult | null>(null)

  const calculateSavings = () => {
    const useCaseData = USE_CASE_MULTIPLIERS[useCase as keyof typeof USE_CASE_MULTIPLIERS]
    const baseTokens = 1000 // Average tokens per request
    const totalTokens = monthlyRequests * baseTokens
    
    // Current provider costs (simplified)
    const providerRates = {
      "openai": 0.030,
      "anthropic": 0.080, 
      "google": 0.025,
      "other": 0.035
    }
    
    const currentRate = providerRates[currentProvider as keyof typeof providerRates]
    const currentCost = (totalTokens / 1000) * currentRate * useCaseData.cost
    
    // LLM Router optimized cost (80% savings on average)
    const optimizationFactor = 0.20 // 80% savings
    const optimizedCost = currentCost * optimizationFactor
    
    const savings = currentCost - optimizedCost
    const savingsPercentage = ((savings / currentCost) * 100)
    
    // Payback period (assuming $99/month for startup plan)
    const monthlyPlanCost = monthlyRequests <= 1000000 ? 99 : 499
    const paybackPeriod = monthlyPlanCost / savings

    setResult({
      currentCost,
      optimizedCost,
      savings,
      savingsPercentage,
      paybackPeriod
    })
  }

  useEffect(() => {
    calculateSavings()
  }, [monthlyRequests, useCase, currentProvider])

  return (
    <div className="bg-white rounded-2xl shadow-xl overflow-hidden">
      {/* Header */}
      <div className="bg-gradient-to-r from-primary-600 to-blue-700 p-8 text-white">
        <div className="flex items-center space-x-3 mb-4">
          <Calculator className="h-8 w-8" />
          <h3 className="text-2xl font-bold">ROI Calculator</h3>
        </div>
        <p className="text-primary-100">
          Calculate your potential savings with intelligent LLM routing
        </p>
      </div>

      <div className="p-8">
        <div className="grid md:grid-cols-2 gap-8">
          {/* Input Controls */}
          <div className="space-y-6">
            {/* Industry Presets */}
            <div>
              <label className="block text-sm font-semibold text-gray-900 mb-3">
                Company Size
              </label>
              <div className="grid grid-cols-2 gap-2">
                {INDUSTRY_PRESETS.map((preset) => (
                  <button
                    key={preset.name}
                    onClick={() => setMonthlyRequests(preset.requests)}
                    className={`p-3 text-left rounded-lg border transition-all ${
                      monthlyRequests === preset.requests
                        ? 'border-primary-500 bg-primary-50'
                        : 'border-gray-200 hover:border-gray-300'
                    }`}
                  >
                    <div className="font-medium text-gray-900">{preset.name}</div>
                    <div className="text-xs text-gray-600">{preset.description}</div>
                  </button>
                ))}
              </div>
            </div>

            {/* Monthly Requests */}
            <div>
              <label className="block text-sm font-semibold text-gray-900 mb-3">
                Monthly AI Requests
              </label>
              <input
                type="number"
                value={monthlyRequests}
                onChange={(e) => setMonthlyRequests(parseInt(e.target.value) || 0)}
                className="w-full p-4 border border-gray-200 rounded-lg text-lg font-mono focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                placeholder="1,000,000"
              />
              <div className="text-sm text-gray-600 mt-2">
                {monthlyRequests.toLocaleString()} requests per month
              </div>
            </div>

            {/* Use Case */}
            <div>
              <label className="block text-sm font-semibold text-gray-900 mb-3">
                Primary Use Case
              </label>
              <select
                value={useCase}
                onChange={(e) => setUseCase(e.target.value)}
                className="w-full p-4 border border-gray-200 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              >
                {Object.entries(USE_CASE_MULTIPLIERS).map(([key, data]) => (
                  <option key={key} value={key}>
                    {data.description}
                  </option>
                ))}
              </select>
            </div>

            {/* Current Provider */}
            <div>
              <label className="block text-sm font-semibold text-gray-900 mb-3">
                Current Provider
              </label>
              <select
                value={currentProvider}
                onChange={(e) => setCurrentProvider(e.target.value)}
                className="w-full p-4 border border-gray-200 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              >
                <option value="openai">OpenAI (GPT-4, GPT-3.5)</option>
                <option value="anthropic">Anthropic (Claude)</option>
                <option value="google">Google (Gemini)</option>
                <option value="other">Other Provider</option>
              </select>
            </div>
          </div>

          {/* Results */}
          <div className="space-y-6">
            {result && (
              <>
                {/* Current vs Optimized */}
                <div className="bg-gradient-to-r from-gray-50 to-blue-50 p-6 rounded-xl">
                  <h4 className="font-bold text-gray-900 mb-4 flex items-center space-x-2">
                    <DollarSign className="h-5 w-5" />
                    <span>Monthly Cost Comparison</span>
                  </h4>
                  
                  <div className="space-y-4">
                    <div className="flex justify-between items-center">
                      <span className="text-gray-600">Current Spend:</span>
                      <span className="text-2xl font-bold text-red-600">
                        ${result.currentCost.toLocaleString('en-US', { maximumFractionDigits: 0 })}
                      </span>
                    </div>
                    
                    <div className="flex justify-between items-center">
                      <span className="text-gray-600">With LLM Router:</span>
                      <span className="text-2xl font-bold text-success-600">
                        ${result.optimizedCost.toLocaleString('en-US', { maximumFractionDigits: 0 })}
                      </span>
                    </div>
                    
                    <div className="border-t pt-4">
                      <div className="flex justify-between items-center">
                        <span className="font-semibold text-gray-900">Monthly Savings:</span>
                        <span className="text-3xl font-bold text-success-600">
                          ${result.savings.toLocaleString('en-US', { maximumFractionDigits: 0 })}
                        </span>
                      </div>
                      <div className="text-right text-success-600 font-semibold">
                        {result.savingsPercentage.toFixed(1)}% reduction
                      </div>
                    </div>
                  </div>
                </div>

                {/* Annual Projections */}
                <div className="bg-gradient-to-r from-success-50 to-green-50 p-6 rounded-xl">
                  <h4 className="font-bold text-gray-900 mb-4 flex items-center space-x-2">
                    <TrendingDown className="h-5 w-5" />
                    <span>Annual Impact</span>
                  </h4>
                  
                  <div className="grid grid-cols-2 gap-4 text-center">
                    <div>
                      <div className="text-2xl font-bold text-success-600">
                        ${(result.savings * 12).toLocaleString('en-US', { maximumFractionDigits: 0 })}
                      </div>
                      <div className="text-sm text-gray-600">Annual Savings</div>
                    </div>
                    
                    <div>
                      <div className="text-2xl font-bold text-primary-600">
                        {result.paybackPeriod < 1 ? '<1' : Math.ceil(result.paybackPeriod)}
                      </div>
                      <div className="text-sm text-gray-600">
                        Month{result.paybackPeriod >= 2 ? 's' : ''} to ROI
                      </div>
                    </div>
                  </div>
                </div>

                {/* Key Benefits */}
                <div className="bg-gradient-to-r from-blue-50 to-primary-50 p-6 rounded-xl">
                  <h4 className="font-bold text-gray-900 mb-4 flex items-center space-x-2">
                    <Zap className="h-5 w-5" />
                    <span>Additional Benefits</span>
                  </h4>
                  
                  <ul className="space-y-2 text-sm">
                    <li className="flex items-center space-x-2">
                      <div className="w-2 h-2 bg-success-500 rounded-full"></div>
                      <span>Automatic quality optimization</span>
                    </li>
                    <li className="flex items-center space-x-2">
                      <div className="w-2 h-2 bg-success-500 rounded-full"></div>
                      <span>Sub-second model selection</span>
                    </li>
                    <li className="flex items-center space-x-2">
                      <div className="w-2 h-2 bg-success-500 rounded-full"></div>
                      <span>Multi-provider redundancy</span>
                    </li>
                    <li className="flex items-center space-x-2">
                      <div className="w-2 h-2 bg-success-500 rounded-full"></div>
                      <span>Real-time cost monitoring</span>
                    </li>
                  </ul>
                </div>

                {/* CTA */}
                <div className="text-center pt-6 border-t">
                  <button className="w-full bg-primary-600 text-white py-4 px-6 rounded-xl text-lg font-semibold hover:bg-primary-700 transition-all transform hover:scale-105">
                    Start Saving Today - Free Trial
                  </button>
                  <p className="text-sm text-gray-600 mt-2">
                    No credit card required • 10K free requests/month
                  </p>
                </div>
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}