/**
 * Common / Utility Types
 */

import type { ReactNode } from 'react';

export interface SelectOption {
  label: string;
  value: string | number;
}

export interface PaginationParams {
  page?: number;
  limit?: number;
}

export interface NotificationItem {
  id: number;
  type: 'success' | 'error' | 'warning' | 'info';
  message: string;
}

export interface NotificationContextValue {
  notifications: NotificationItem[];
  removeNotification: (id: number) => void;
  success: (message: string, duration?: number) => number;
  error: (message: string, duration?: number) => number;
  warning: (message: string, duration?: number) => number;
  info: (message: string, duration?: number) => number;
}

export type Theme = 'light' | 'dark';

export interface ThemeContextValue {
  theme: Theme;
  toggleTheme: () => void;
  setTheme: (theme: Theme) => void;
}

export interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  children: ReactNode;
  footer?: ReactNode;
}

export interface BreadcrumbItem {
  label: string;
  path?: string;
}

export interface RouteParams {
  [key: string]: string | undefined;
}
