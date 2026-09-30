/**
 * Authentication & Authorization Types
 */

export type UserRole =
  | 'student'
  | 'faculty'
  | 'admin'
  | 'college_admin'
  | 'hod'
  | 'principal'
  | 'super_admin';

export interface User {
  id: number;
  regdno: string;
  name: string;
  email: string;
  role: UserRole;
  college_id?: number;
  branch_id?: number;
  section_id?: number;
  avatar_url?: string;
  // Additional fields that may come from backend
  batch_id?: number;
  program_id?: number;
  regulation_id?: number;
}

export interface AuthContextValue {
  user: User | null;
  loading: boolean;
  login: (regdno: string, password: string, collegeId: string | number) => Promise<LoginResponse>;
  logout: () => void;
  isAuthenticated: boolean;
  isAdmin: () => boolean;
  isSuperAdmin: () => boolean;
  isFaculty: () => boolean;
  isHOD: () => boolean;
  isPrincipal: () => boolean;
  isStudent: () => boolean;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface SecureTokenStorage {
  setToken: (token: string) => void;
  getToken: () => string | null;
  isTokenExpired: () => boolean;
  removeToken: () => void;
  setUser: (user: User) => void;
  getUser: () => User | null;
  removeUser: () => void;
  clear: () => void;
}

export interface RoleChecks {
  isAdmin: () => boolean;
  isSuperAdmin: () => boolean;
  isFaculty: () => boolean;
  isHOD: () => boolean;
  isPrincipal: () => boolean;
  isStudent: () => boolean;
}
