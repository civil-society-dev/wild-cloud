/**
 * Centralized API configuration
 * All API clients should use this to get the base URL
 */

/**
 * Get the API base URL from environment or default to empty string (for proxy)
 * Empty string means use relative URLs that go through Vite's proxy in development
 */
export function getApiBaseUrl(): string {
  return import.meta.env.VITE_API_BASE_URL || '';
}

/**
 * Check if SSE is enabled
 */
export function isSSEEnabled(): boolean {
  return import.meta.env.VITE_ENABLE_SSE === 'true';
}