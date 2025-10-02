import { useState, useEffect } from 'react'
import Head from 'next/head'
import Link from 'next/link'
import {
  Bot,
  DollarSign,
  Target,
  Building2,
  Zap,
  BarChart3,
  CheckCircle,
  ArrowRight,
  Play,
  Code,
  Lightbulb,
  Users,
  LogIn
} from 'lucide-react'
import { useAuth } from '../src/context/AuthContext'
import StatsCounter from '../src/components/StatsCounter'
import InteractiveDemo from '../src/components/InteractiveDemo'
import FeatureCard from '../src/components/FeatureCard'
import CostCalculator from '../src/components/CostCalculator'

export default function Home() {
  const [mounted, setMounted] = useState(false)
  const { user, isLoading } = useAuth()

  useEffect(() => {
    setMounted(true)
  }, [])

  if (!mounted) return null

  return (
    <>
      <Head>
        <title>LLM Router - Save 80%+ on AI Costs | Intelligent Model Routing</title>
        <meta name="description" content="Reduce AI spending by 80%+ with intelligent LLM routing across 200+ models. Enterprise-grade optimization with sub-second response times." />
        <meta name="keywords" content="LLM routing, AI cost optimization, model selection, enterprise AI, API gateway" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <link rel="icon" href="/favicon.ico" />
        
        {/* Open Graph */}
        <meta property="og:title" content="LLM Router - Save 80%+ on AI Costs" />
        <meta property="og:description" content="Intelligent model routing across 200+ models with enterprise-grade optimization" />
        <meta property="og:type" content="website" />
        
        {/* Schema.org structured data */}
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{
            __html: JSON.stringify({
              "@context": "https://schema.org",
              "@type": "SoftwareApplication",
              "name": "LLM Router",
              "description": "Intelligent AI model routing with cost optimization",
              "applicationCategory": "BusinessApplication",
              "operatingSystem": "Web",
              "offers": {
                "@type": "Offer",
                "price": "0",
                "priceCurrency": "USD"
              }
            })
          }}
        />
      </Head>

      <main className="min-h-screen bg-gradient-to-br from-slate-50 to-blue-50">
        {/* Navigation */}
        <nav className="fixed top-0 w-full bg-white/80 backdrop-blur-md border-b border-gray-200 z-50">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex justify-between items-center h-16">
              <div className="flex items-center space-x-2">
                <Bot className="h-8 w-8 text-primary-600" />
                <span className="text-xl font-bold text-gray-900">LLM Router</span>
              </div>
              <div className="hidden md:flex items-center space-x-8">
                <a href="#features" className="text-gray-700 hover:text-primary-600 transition-colors">Features</a>
                <a href="#demo" className="text-gray-700 hover:text-primary-600 transition-colors">Demo</a>
                <a href="#pricing" className="text-gray-700 hover:text-primary-600 transition-colors">Pricing</a>
                <a href="#docs" className="text-gray-700 hover:text-primary-600 transition-colors">Docs</a>
                {!isLoading && (
                  user ? (
                    <Link href="/dashboard">
                      <button className="bg-primary-600 text-white px-4 py-2 rounded-lg hover:bg-primary-700 transition-colors">
                        Dashboard
                      </button>
                    </Link>
                  ) : (
                    <div className="flex items-center space-x-4">
                      <Link href="/login" className="text-gray-700 hover:text-primary-600 transition-colors flex items-center space-x-1">
                        <LogIn className="h-4 w-4" />
                        <span>Sign In</span>
                      </Link>
                      <Link href="/register">
                        <button className="bg-primary-600 text-white px-4 py-2 rounded-lg hover:bg-primary-700 transition-colors">
                          Get Started
                        </button>
                      </Link>
                    </div>
                  )
                )}
              </div>
            </div>
          </div>
        </nav>

        {/* Hero Section */}
        <section className="pt-24 pb-16">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="text-center">
              <h1 className="text-4xl md:text-6xl font-bold text-gray-900 mb-6 animate-slide-up">
                Save <span className="gradient-text">Up to 99%</span> on AI Costs
                <br />
                Without Sacrificing Quality
              </h1>
              <p className="text-xl text-gray-600 mb-8 max-w-3xl mx-auto animate-slide-up">
                Intelligent LLM routing across <strong>26 verified models</strong> with real-time cost optimization.
                Get the right model for every task with data-driven recommendations.
              </p>
              
              {/* CTA Buttons */}
              <div className="flex flex-col sm:flex-row gap-4 justify-center items-center mb-12 animate-slide-up">
                {!isLoading && (
                  user ? (
                    <Link href="/dashboard">
                      <button className="bg-primary-600 text-white px-8 py-4 rounded-xl text-lg font-semibold hover:bg-primary-700 transition-all transform hover:scale-105 flex items-center space-x-2">
                        <BarChart3 className="h-5 w-5" />
                        <span>Go to Dashboard</span>
                      </button>
                    </Link>
                  ) : (
                    <Link href="/register">
                      <button className="bg-primary-600 text-white px-8 py-4 rounded-xl text-lg font-semibold hover:bg-primary-700 transition-all transform hover:scale-105 flex items-center space-x-2">
                        <Play className="h-5 w-5" />
                        <span>Try Interactive Demo</span>
                      </button>
                    </Link>
                  )
                )}
                <a href="#demo">
                  <button className="border border-gray-300 text-gray-700 px-8 py-4 rounded-xl text-lg font-semibold hover:bg-gray-50 transition-all flex items-center space-x-2">
                    <Code className="h-5 w-5" />
                    <span>View API Docs</span>
                  </button>
                </a>
              </div>

              {/* Live Stats */}
              <div className="grid grid-cols-2 md:grid-cols-4 gap-8 max-w-4xl mx-auto">
                <StatsCounter
                  icon={<Bot className="h-8 w-8 text-primary-600" />}
                  value={26}
                  label="AI Models"
                  suffix=""
                />
                <StatsCounter
                  icon={<DollarSign className="h-8 w-8 text-success-600" />}
                  value={87}
                  label="Avg Cost Savings"
                  suffix="%"
                />
                <StatsCounter 
                  icon={<Zap className="h-8 w-8 text-warning-500" />}
                  value={500}
                  label="Response Time"
                  suffix="ms"
                />
                <StatsCounter 
                  icon={<Building2 className="h-8 w-8 text-blue-600" />}
                  value={99.9}
                  label="Uptime SLA"
                  suffix="%"
                />
              </div>
            </div>
          </div>
        </section>

        {/* Value Proposition Cards */}
        <section className="py-16 bg-white">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="text-center mb-16">
              <h2 className="text-3xl md:text-4xl font-bold text-gray-900 mb-4">
                Why Choose LLM Router?
              </h2>
              <p className="text-xl text-gray-600 max-w-2xl mx-auto">
                Transform your AI operations with intelligent routing that saves costs while improving performance
              </p>
            </div>

            <div className="feature-grid">
              <FeatureCard
                icon={<DollarSign className="h-12 w-12 text-success-600" />}
                title="Massive Cost Reduction"
                description="Reduce AI spending by 80%+ through intelligent model routing and cost-optimized selection without sacrificing quality."
                features={[
                  "Smart budget-conscious routing",
                  "Real-time cost optimization", 
                  "Enterprise volume discounts",
                  "Transparent pricing analytics"
                ]}
              />
              
              <FeatureCard
                icon={<Target className="h-12 w-12 text-primary-600" />}
                title="Smart Task Routing"
                description="ML-powered classification automatically selects the optimal model for each task type and complexity level."
                features={[
                  "Sub-second task classification",
                  "Context-aware model selection",
                  "Complexity-based routing",
                  "Multi-modal support"
                ]}
              />
              
              <FeatureCard
                icon={<Building2 className="h-12 w-12 text-blue-600" />}
                title="Enterprise Ready"
                description="Production-grade infrastructure with SLAs, security, and comprehensive monitoring for mission-critical applications."
                features={[
                  "99.9% uptime guarantee",
                  "Enterprise security & compliance",
                  "Real-time monitoring & alerts",
                  "24/7 dedicated support"
                ]}
              />
              
              <FeatureCard
                icon={<Zap className="h-12 w-12 text-warning-500" />}
                title="Lightning Fast"
                description="Sub-second response times with global CDN deployment and optimized routing algorithms for maximum performance."
                features={[
                  "< 500ms average latency",
                  "Global edge deployment",
                  "Intelligent caching",
                  "Auto-scaling infrastructure"
                ]}
              />
            </div>
          </div>
        </section>

        {/* Interactive Demo Section */}
        <section id="demo" className="py-20 bg-gradient-to-br from-gray-50 to-blue-50">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="text-center mb-16">
              <h2 className="text-3xl md:text-4xl font-bold text-gray-900 mb-4">
                See It In Action
              </h2>
              <p className="text-xl text-gray-600 max-w-2xl mx-auto">
                Try our live demo to see how LLM Router intelligently selects models and calculates cost savings
              </p>
            </div>
            
            <InteractiveDemo />
          </div>
        </section>

        {/* Cost Calculator */}
        <section className="py-20 bg-white">
          <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="text-center mb-16">
              <h2 className="text-3xl md:text-4xl font-bold text-gray-900 mb-4">
                Calculate Your Savings
              </h2>
              <p className="text-xl text-gray-600">
                See how much you could save by switching to intelligent model routing
              </p>
            </div>
            
            <CostCalculator />
          </div>
        </section>

        {/* Social Proof / Testimonials */}
        <section className="py-20 bg-gray-50">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="text-center mb-16">
              <h2 className="text-3xl md:text-4xl font-bold text-gray-900 mb-4">
                Trusted by Industry Leaders
              </h2>
              <p className="text-xl text-gray-600">
                Join thousands of developers and enterprises optimizing their AI costs
              </p>
            </div>

            <div className="grid md:grid-cols-3 gap-8">
              {[
                {
                  quote: "LLM Router reduced our AI costs by 87% while actually improving response quality. Real savings backed by data analytics.",
                  author: "Sarah Chen",
                  title: "CTO, TechFlow Systems",
                  savings: "$50k/month saved"
                },
                {
                  quote: "The intelligent routing is incredibly accurate. It knows exactly when to use premium models vs cost-effective alternatives.",
                  author: "Michael Rodriguez",
                  title: "AI Engineering Lead, DataCorp",
                  savings: "$120k/month saved"
                },
                {
                  quote: "Integration was seamless and the cost savings were immediate. Our development team loves the API simplicity.",
                  author: "Alex Thompson",
                  title: "Lead Developer, InnovateAI",
                  savings: "$30k/month saved"
                }
              ].map((testimonial, index) => (
                <div key={index} className="bg-white p-8 rounded-xl shadow-lg">
                  <div className="flex items-center space-x-1 mb-4">
                    {[...Array(5)].map((_, i) => (
                      <div key={i} className="w-5 h-5 bg-yellow-400 rounded-full"></div>
                    ))}
                  </div>
                  <p className="text-gray-700 mb-6 italic">"{testimonial.quote}"</p>
                  <div className="border-t pt-4">
                    <div className="font-semibold text-gray-900">{testimonial.author}</div>
                    <div className="text-gray-600 text-sm">{testimonial.title}</div>
                    <div className="text-success-600 font-semibold text-sm mt-2">{testimonial.savings}</div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* CTA Section */}
        <section className="py-20 bg-primary-600">
          <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
            <h2 className="text-3xl md:text-4xl font-bold text-white mb-6">
              Ready to Optimize Your AI Costs?
            </h2>
            <p className="text-xl text-primary-100 mb-8 max-w-2xl mx-auto">
              Join thousands of developers saving 80%+ on AI costs while improving performance. 
              Get started with our free tier in under 5 minutes.
            </p>
            
            <div className="flex flex-col sm:flex-row gap-4 justify-center">
              <button className="bg-white text-primary-600 px-8 py-4 rounded-xl text-lg font-semibold hover:bg-gray-100 transition-all transform hover:scale-105 flex items-center justify-center space-x-2">
                <span>Start Free Trial</span>
                <ArrowRight className="h-5 w-5" />
              </button>
              <button className="border-2 border-white text-white px-8 py-4 rounded-xl text-lg font-semibold hover:bg-white hover:text-primary-600 transition-all">
                Schedule Demo
              </button>
            </div>
            
            <p className="text-primary-200 text-sm mt-6">
              No credit card required • 10K free requests/month • Cancel anytime
            </p>
          </div>
        </section>

        {/* Footer */}
        <footer className="bg-gray-900 text-white py-16">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="grid md:grid-cols-4 gap-8">
              <div>
                <div className="flex items-center space-x-2 mb-6">
                  <Bot className="h-8 w-8 text-primary-400" />
                  <span className="text-xl font-bold">LLM Router</span>
                </div>
                <p className="text-gray-400 mb-4">
                  Intelligent AI model routing with enterprise-grade optimization and massive cost savings.
                </p>
                <div className="flex space-x-4">
                  <a href="#" className="text-gray-400 hover:text-white transition-colors">Twitter</a>
                  <a href="#" className="text-gray-400 hover:text-white transition-colors">GitHub</a>
                  <a href="#" className="text-gray-400 hover:text-white transition-colors">LinkedIn</a>
                </div>
              </div>
              
              <div>
                <h3 className="font-semibold mb-4">Product</h3>
                <ul className="space-y-2 text-gray-400">
                  <li><a href="#" className="hover:text-white transition-colors">Features</a></li>
                  <li><a href="#" className="hover:text-white transition-colors">Pricing</a></li>
                  <li><a href="#" className="hover:text-white transition-colors">API Reference</a></li>
                  <li><a href="#" className="hover:text-white transition-colors">Integrations</a></li>
                </ul>
              </div>
              
              <div>
                <h3 className="font-semibold mb-4">Resources</h3>
                <ul className="space-y-2 text-gray-400">
                  <li><a href="#" className="hover:text-white transition-colors">Documentation</a></li>
                  <li><a href="#" className="hover:text-white transition-colors">Guides</a></li>
                  <li><a href="#" className="hover:text-white transition-colors">Blog</a></li>
                  <li><a href="#" className="hover:text-white transition-colors">Status</a></li>
                </ul>
              </div>
              
              <div>
                <h3 className="font-semibold mb-4">Company</h3>
                <ul className="space-y-2 text-gray-400">
                  <li><a href="#" className="hover:text-white transition-colors">About</a></li>
                  <li><a href="#" className="hover:text-white transition-colors">Contact</a></li>
                  <li><a href="#" className="hover:text-white transition-colors">Privacy</a></li>
                  <li><a href="#" className="hover:text-white transition-colors">Terms</a></li>
                </ul>
              </div>
            </div>
            
            <div className="border-t border-gray-800 mt-12 pt-8 text-center text-gray-400">
              <p>&copy; 2025 LLM Router. All rights reserved.</p>
            </div>
          </div>
        </footer>
      </main>
    </>
  )
}