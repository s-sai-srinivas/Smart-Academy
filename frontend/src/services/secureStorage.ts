/**
 * Secure Token Storage
 *
 * SECURITY: This module provides token storage with XSS protection.
 *
 * For production: Use httpOnly cookies set by the backend (recommended)
 * For development: Uses sessionStorage which is cleared on tab close (more secure than localStorage)
 *
 * XSS Protection:
 * - sessionStorage is more secure than localStorage (cleared on tab close)
 * - httpOnly cookies (backend-set) are the most secure (not accessible to JS)
 * - Consider implementing CSRF tokens for cookie-based auth
 */

import type { User } from '../types/auth';

const TOKEN_KEY = 'authToken';
const USER_KEY = 'currentUser';

// Check if httpOnly cookie mode is enabled (set by backend)
// Only use httpOnly mode if there's NO token in sessionStorage AND authToken cookie exists
// This prevents false httpOnly detection from stray/empty cookies
const isHttpOnlyMode = (): boolean => {
  // If we have a token in sessionStorage, always use that (not httpOnly mode)
  if (sessionStorage.getItem(TOKEN_KEY)) {
    return false;
  }
  // Only check for httpOnly cookie if no sessionStorage token
  return document.cookie.includes('authToken=') && /authToken=[^;]/.test(document.cookie);
};

/**
 * Decode JWT token to check expiry
 * Returns true if token is expired or invalid
 */
const isTokenExpiredFn = (token: string | null): boolean => {
  if (!token || token === 'httpOnly') return true;
  try {
    // JWT structure: header.payload.signature
    const parts = token.split('.');
    if (parts.length !== 3) return true;

    // Decode payload (base64url)
    const payload = parts[1];
    const decoded = JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')));

    // Check exp claim (Unix timestamp in seconds)
    if (decoded.exp) {
      const expiryTime = decoded.exp * 1000; // Convert to milliseconds
      return Date.now() >= expiryTime;
    }

    return false; // No exp claim, assume valid
  } catch (e) {
    console.error('Failed to decode token:', e);
    return true; // Invalid token structure
  }
};

// Secure token storage interface
export const secureTokenStorage = {
  /**
   * Set auth token
   * For production: Backend should set httpOnly cookie
   * For development: Uses sessionStorage
   */
  setToken: (token: string): void => {
    if (isHttpOnlyMode()) {
      // Token is set via httpOnly cookie by backend
      return;
    }
    // Use sessionStorage instead of localStorage for better security
    // sessionStorage is cleared when tab is closed, reducing XSS attack window
    sessionStorage.setItem(TOKEN_KEY, token);
  },

  /**
   * Get auth token
   */
  getToken: (): string | null => {
    if (isHttpOnlyMode()) {
      // For httpOnly cookies, we just check existence
      return document.cookie.includes('authToken=') ? 'httpOnly' : null;
    }
    return sessionStorage.getItem(TOKEN_KEY);
  },

  /**
   * Check if token is expired (client-side check)
   */
  isTokenExpired: (): boolean => {
    const token = secureTokenStorage.getToken();
    return isTokenExpiredFn(token);
  },

  /**
   * Remove auth token
   */
  removeToken: (): void => {
    if (isHttpOnlyMode()) {
      // Clear httpOnly cookie by setting expired date
      document.cookie = 'authToken=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/';
    } else {
      sessionStorage.removeItem(TOKEN_KEY);
    }
  },

  /**
   * Set current user (non-sensitive data only)
   */
  setUser: (user: User): void => {
    sessionStorage.setItem(USER_KEY, JSON.stringify(user));
  },

  /**
   * Get current user
   */
  getUser: (): User | null => {
    const user = sessionStorage.getItem(USER_KEY);
    return user ? (JSON.parse(user) as User) : null;
  },

  /**
   * Remove current user
   */
  removeUser: (): void => {
    sessionStorage.removeItem(USER_KEY);
  },

  /**
   * Clear all auth data
   */
  clear: (): void => {
    secureTokenStorage.removeToken();
    secureTokenStorage.removeUser();
  },
};

// Legacy authAPI for backward compatibility
// Using dynamic require to avoid circular dependency with api.ts
export const authAPI = {
  getColleges: (): unknown => {
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const { default: api } = require('./api');
    return api.get('/colleges', { skipAuth: true });
  },

  login: async (
    regdno: string,
    password: string,
    college_id: number | string | null = null
  ): Promise<unknown> => {
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const { default: api } = require('./api');
    const payload: Record<string, unknown> = { regdno, password };
    if (college_id) payload.college_id = college_id;
    const data = await api.post('/auth/login', payload, { skipAuth: true });

    // Use secure storage
    secureTokenStorage.setToken(data.token as string);
    secureTokenStorage.setUser(data.user as User);

    return data;
  },

  logout: (): void => {
    secureTokenStorage.clear();
  },

  getCurrentUser: (): User | null => {
    return secureTokenStorage.getUser();
  },

  isAuthenticated: (): boolean => {
    const token = secureTokenStorage.getToken();
    if (!token) return false;
    // Check if token is expired
    if (isTokenExpiredFn(token)) {
      secureTokenStorage.clear(); // Clear expired token
      return false;
    }
    return true;
  },
};
