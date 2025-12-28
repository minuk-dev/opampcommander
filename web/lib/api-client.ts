const API_BASE_URL = process.env.OPAMP_API_URL || 'http://localhost:8080';

export interface ApiResponse<T> {
  data?: T;
  error?: string;
}

export async function apiClient<T>(
  endpoint: string,
  options?: RequestInit
): Promise<ApiResponse<T>> {
  try {
    const response = await fetch(`${API_BASE_URL}${endpoint}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
    });

    if (!response.ok) {
      return {
        error: `API Error: ${response.status} ${response.statusText}`,
      };
    }

    const data = await response.json();
    return { data };
  } catch (error) {
    return {
      error: error instanceof Error ? error.message : 'Unknown error',
    };
  }
}
