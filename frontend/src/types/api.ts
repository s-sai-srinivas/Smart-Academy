/**
 * API & Error Types
 */

export interface ApiError extends Error {
  status?: number;
  code?: string;
  originalError?: unknown;
  originalResponse?: unknown;
  config?: unknown;
}

export interface ApiResponse<T = unknown> {
  data: T;
  message?: string;
}

export interface ApiRequestConfig {
  skipAuth?: boolean;
  // Allow other axios config properties
  [key: string]: unknown;
}

export interface JobStatus {
  id: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  message?: string;
  error_message?: string;
  result?: unknown;
  progress?: number;
}

export interface PaginatedResponse<T = unknown> {
  items: T[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}
