import { createContext, useContext, useState, useCallback, useMemo, type ReactNode } from 'react';
import { authAPI } from '../services/api';
import { secureTokenStorage } from '../services/secureStorage';
import type { User, AuthContextValue, RoleChecks, LoginResponse } from '../types/auth';

const AuthContext = createContext<AuthContextValue | null>(null);

// Define role check functions outside component to avoid re-creation
const createRoleChecks = (user: User | null): RoleChecks => ({
  isAdmin: () => user?.role === 'admin' || user?.role === 'college_admin',
  isSuperAdmin: () => user?.role === 'super_admin',
  isFaculty: () => user?.role === 'faculty',
  isHOD: () => user?.role === 'hod',
  isPrincipal: () => user?.role === 'principal',
  isStudent: () => user?.role === 'student',
});

export function AuthProvider({ children }: { children: ReactNode }) {
  // Validate token on initialization - check for expired tokens
  const validateStoredToken = (): User | null => {
    if (secureTokenStorage.isTokenExpired()) {
      console.warn('Token expired on app load - clearing auth');
      secureTokenStorage.clear();
      return null;
    }
    return secureTokenStorage.getUser();
  };

  const [user, setUser] = useState<User | null>(validateStoredToken);
  const [loading, setLoading] = useState(false);

  const login = useCallback(
    async (
      regdno: string,
      password: string,
      collegeId: string | number
    ): Promise<LoginResponse> => {
      setLoading(true);
      try {
        const data = (await authAPI.login(regdno, password, collegeId)) as unknown as LoginResponse;
        setUser(data.user || null);
        return data;
      } finally {
        setLoading(false);
      }
    },
    []
  );

  const logout = useCallback(() => {
    authAPI.logout();
    setUser(null);
  }, []);

  // Memoize role check functions to prevent re-creation on every render
  const roleChecks = useMemo(() => createRoleChecks(user), [user]);

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      loading,
      login,
      logout,
      ...roleChecks,
      isAuthenticated: !!user,
    }),
    [user, loading, login, logout, roleChecks]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}

export default AuthContext;
