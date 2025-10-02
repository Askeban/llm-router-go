const API_BASE_URL = process.env.API_BASE_URL || 'https://api.routellm.dev';

interface Model {
  id: string;
  name: string;
  provider: string;
  description: string;
  task_type: string;
  modalities: string[];
  quality_score: number;
  cost_per_1k: number;
  avg_latency_ms: number;
  context_window: number;
  safety_score: number;
  reasoning_score: number;
  is_open_source: boolean;
  release_date: string;
}

interface RecommendationRequest {
  requirements: {
    task_type: string;
    budget_per_1k_tokens: number;
    quality_threshold: number;
    max_latency_ms: number;
    required_capabilities: string[];
    modality?: string;
    context_window_min?: number;
  };
  preferences: {
    provider_preference: string[];
    open_source_only: boolean;
    include_reasoning: boolean;
  };
  use_case: string;
}

interface RecommendationResponse {
  data: {
    recommendations: Array<{
      model: Model;
      overall_score: number;
      reasoning: string;
    }>;
    total_models: number;
    processing_time: string;
  };
  timestamp: string;
}

class ApiService {
  private getAuthHeaders() {
    const token = typeof window !== 'undefined' ? localStorage.getItem('auth_token') : null;
    return {
      'Content-Type': 'application/json',
      ...(token && { 'Authorization': `Bearer ${token}` })
    };
  }

  async getModels(): Promise<Model[]> {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/public/models`, {
        method: 'GET',
        headers: this.getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      return data.data || [];
    } catch (error) {
      console.error('Failed to fetch models:', error);
      throw error;
    }
  }

  async getModelsByModality(modality: string): Promise<Model[]> {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/models/multimodal?modality=${encodeURIComponent(modality)}`, {
        method: 'GET',
        headers: this.getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      return data.models || [];
    } catch (error) {
      console.error('Failed to fetch models by modality:', error);
      throw error;
    }
  }

  async getRecommendations(request: RecommendationRequest): Promise<RecommendationResponse> {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/recommendations`, {
        method: 'POST',
        headers: this.getAuthHeaders(),
        body: JSON.stringify(request),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      return data;
    } catch (error) {
      console.error('Failed to get recommendations:', error);
      throw error;
    }
  }

  async getAnalytics() {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/analytics/usage`, {
        method: 'GET',
        headers: this.getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      return data;
    } catch (error) {
      console.error('Failed to fetch analytics:', error);
      throw error;
    }
  }

  async getSystemStats() {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/analytics/system-stats`, {
        method: 'GET',
        headers: this.getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      return data;
    } catch (error) {
      console.error('Failed to fetch system stats:', error);
      throw error;
    }
  }

  async submitFeedback(recommendationId: string, feedback: {
    rating: number;
    selected_model: string;
    comments?: string;
  }) {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/recommendations/feedback`, {
        method: 'POST',
        headers: this.getAuthHeaders(),
        body: JSON.stringify({
          recommendation_id: recommendationId,
          ...feedback,
        }),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to submit feedback:', error);
      throw error;
    }
  }

  async getProviderComparison() {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/analytics/provider-comparison`, {
        method: 'GET',
        headers: this.getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to fetch provider comparison:', error);
      throw error;
    }
  }

  async getCapabilityAnalytics() {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/analytics/capabilities`, {
        method: 'GET',
        headers: this.getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to fetch capability analytics:', error);
      throw error;
    }
  }
}

export const apiService = new ApiService();
export type { Model, RecommendationRequest, RecommendationResponse };