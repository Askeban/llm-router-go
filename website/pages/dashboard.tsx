import React, { useEffect, useState } from 'react';
import { useRouter } from 'next/router';
import { useAuth } from '../src/context/AuthContext';
import {
  Loader,
  User,
  Key,
  BarChart3,
  Settings,
  LogOut,
  Copy,
  Check,
  AlertCircle,
  TrendingUp,
  Activity,
  Zap
} from 'lucide-react';

interface UserStats {
  total_requests: number;
  total_tokens: number;
  monthly_limit: number;
  usage_percent: number;
  requests_remaining: number;
  plan_type: string;
  period: string;
}

const DashboardPage: React.FC = () => {
  const { user, isAuthenticated, isLoading, logout } = useAuth();
  const router = useRouter();
  const [stats, setStats] = useState<UserStats | null>(null);
  const [loadingStats, setLoadingStats] = useState(true);
  const [copiedKey, setCopiedKey] = useState(false);

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.push('/login');
    }
  }, [isAuthenticated, isLoading, router]);

  useEffect(() => {
    if (isAuthenticated) {
      fetchUsageStats();
    }
  }, [isAuthenticated]);

  const fetchUsageStats = async () => {
    try {
      const token = localStorage.getItem('token');
      const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/v1/auth/usage`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        const data = await response.json();
        setStats(data);
      }
    } catch (error) {
      console.error('Failed to fetch usage stats:', error);
    } finally {
      setLoadingStats(false);
    }
  };

  const handleLogout = () => {
    logout();
    router.push('/');
  };

  const copyApiKey = () => {
    // TODO: Implement API key generation
    navigator.clipboard.writeText('api_key_placeholder');
    setCopiedKey(true);
    setTimeout(() => setCopiedKey(false), 2000);
  };

  if (isLoading || !isAuthenticated) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <Loader className="w-8 h-8 animate-spin text-blue-600" />
      </div>
    );
  }

  const getPlanColor = (plan: string) => {
    switch (plan) {
      case 'beta': return 'bg-purple-100 text-purple-800';
      case 'pro': return 'bg-blue-100 text-blue-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const getUsageColor = (percent: number) => {
    if (percent >= 90) return 'bg-red-500';
    if (percent >= 70) return 'bg-yellow-500';
    return 'bg-green-500';
  };

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex justify-between items-center">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
              <p className="text-sm text-gray-600 mt-1">Welcome back, {user?.full_name}</p>
            </div>
            <div className="flex items-center space-x-4">
              <span className={`px-3 py-1 rounded-full text-sm font-medium ${getPlanColor(user?.plan_type || 'free')}`}>
                {user?.plan_type?.toUpperCase()} Plan
              </span>
              <button
                onClick={handleLogout}
                className="flex items-center px-4 py-2 text-gray-700 hover:text-gray-900 transition-colors"
              >
                <LogOut className="w-4 h-4 mr-2" />
                Logout
              </button>
            </div>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Beta Access Notice */}
        {user?.beta_access && (
          <div className="mb-6 bg-gradient-to-r from-purple-50 to-indigo-50 border border-purple-200 rounded-lg p-4">
            <div className="flex items-center">
              <Zap className="w-5 h-5 text-purple-600 mr-3" />
              <div>
                <h3 className="text-sm font-semibold text-purple-900">Beta Access Active</h3>
                <p className="text-sm text-purple-700">You have exclusive access to beta features and higher rate limits</p>
              </div>
            </div>
          </div>
        )}

        {/* Stats Grid */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          {/* Total Requests */}
          <div className="bg-white rounded-lg shadow-sm p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600 mb-1">Total Requests</p>
                <p className="text-3xl font-bold text-gray-900">
                  {loadingStats ? '-' : stats?.total_requests.toLocaleString()}
                </p>
              </div>
              <div className="w-12 h-12 bg-blue-100 rounded-lg flex items-center justify-center">
                <Activity className="w-6 h-6 text-blue-600" />
              </div>
            </div>
            <p className="text-xs text-gray-500 mt-2">This month</p>
          </div>

          {/* Requests Remaining */}
          <div className="bg-white rounded-lg shadow-sm p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600 mb-1">Remaining</p>
                <p className="text-3xl font-bold text-gray-900">
                  {loadingStats ? '-' : stats?.requests_remaining.toLocaleString()}
                </p>
              </div>
              <div className="w-12 h-12 bg-green-100 rounded-lg flex items-center justify-center">
                <TrendingUp className="w-6 h-6 text-green-600" />
              </div>
            </div>
            <p className="text-xs text-gray-500 mt-2">Of {stats?.monthly_limit.toLocaleString()} monthly limit</p>
          </div>

          {/* Total Tokens */}
          <div className="bg-white rounded-lg shadow-sm p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600 mb-1">Total Tokens</p>
                <p className="text-3xl font-bold text-gray-900">
                  {loadingStats ? '-' : stats?.total_tokens.toLocaleString()}
                </p>
              </div>
              <div className="w-12 h-12 bg-purple-100 rounded-lg flex items-center justify-center">
                <BarChart3 className="w-6 h-6 text-purple-600" />
              </div>
            </div>
            <p className="text-xs text-gray-500 mt-2">Processed this month</p>
          </div>
        </div>

        {/* Usage Progress */}
        {stats && (
          <div className="bg-white rounded-lg shadow-sm p-6 mb-8">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900">Usage Overview</h3>
              <span className="text-sm text-gray-600">{stats.usage_percent}% used</span>
            </div>
            <div className="w-full bg-gray-200 rounded-full h-4">
              <div
                className={`h-4 rounded-full transition-all ${getUsageColor(stats.usage_percent)}`}
                style={{ width: `${Math.min(stats.usage_percent, 100)}%` }}
              />
            </div>
            <div className="flex justify-between mt-2 text-xs text-gray-600">
              <span>{stats.total_requests.toLocaleString()} requests</span>
              <span>{stats.monthly_limit.toLocaleString()} limit</span>
            </div>
          </div>
        )}

        {/* API Keys Section */}
        <div className="bg-white rounded-lg shadow-sm p-6 mb-8">
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center">
              <Key className="w-5 h-5 text-gray-600 mr-3" />
              <h3 className="text-lg font-semibold text-gray-900">API Keys</h3>
            </div>
          </div>

          <div className="bg-gray-50 rounded-lg p-4 border border-gray-200">
            <div className="flex items-center justify-between">
              <div className="flex-1">
                <p className="text-sm text-gray-600 mb-2">Your API Key</p>
                <code className="text-sm text-gray-900 bg-white px-3 py-2 rounded border border-gray-200 inline-block">
                  Coming Soon - API Key Management
                </code>
              </div>
              <button
                onClick={copyApiKey}
                className="ml-4 flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                disabled
              >
                {copiedKey ? (
                  <>
                    <Check className="w-4 h-4 mr-2" />
                    Copied
                  </>
                ) : (
                  <>
                    <Copy className="w-4 h-4 mr-2" />
                    Copy
                  </>
                )}
              </button>
            </div>
          </div>

          <div className="mt-4 flex items-start">
            <AlertCircle className="w-4 h-4 text-yellow-600 mr-2 mt-0.5" />
            <p className="text-sm text-gray-600">
              API key management is coming soon. You'll be able to generate, rotate, and manage multiple API keys.
            </p>
          </div>
        </div>

        {/* Account Information */}
        <div className="bg-white rounded-lg shadow-sm p-6">
          <div className="flex items-center mb-6">
            <User className="w-5 h-5 text-gray-600 mr-3" />
            <h3 className="text-lg font-semibold text-gray-900">Account Information</h3>
          </div>

          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-sm text-gray-600">Full Name</p>
                <p className="text-sm font-medium text-gray-900 mt-1">{user?.full_name}</p>
              </div>
              <div>
                <p className="text-sm text-gray-600">Email</p>
                <p className="text-sm font-medium text-gray-900 mt-1">{user?.email}</p>
              </div>
              <div>
                <p className="text-sm text-gray-600">Plan Type</p>
                <p className="text-sm font-medium text-gray-900 mt-1">{user?.plan_type}</p>
              </div>
              <div>
                <p className="text-sm text-gray-600">Status</p>
                <p className="text-sm font-medium text-gray-900 mt-1">{user?.status}</p>
              </div>
            </div>
          </div>
        </div>

        {/* Quick Links */}
        <div className="mt-8 grid grid-cols-1 md:grid-cols-2 gap-6">
          <a
            href="/docs"
            className="bg-white rounded-lg shadow-sm p-6 hover:shadow-md transition-shadow"
          >
            <h4 className="font-semibold text-gray-900 mb-2">API Documentation</h4>
            <p className="text-sm text-gray-600">Learn how to integrate RouteLLM into your applications</p>
          </a>
          <a
            href="/"
            className="bg-white rounded-lg shadow-sm p-6 hover:shadow-md transition-shadow"
          >
            <h4 className="font-semibold text-gray-900 mb-2">Try Interactive Demo</h4>
            <p className="text-sm text-gray-600">Test model recommendations with the interactive demo</p>
          </a>
        </div>
      </main>
    </div>
  );
};

export default DashboardPage;
