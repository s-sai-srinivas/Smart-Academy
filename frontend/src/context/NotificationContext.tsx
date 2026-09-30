import {
  createContext,
  useContext,
  useState,
  useCallback,
  type ReactNode,
  type ReactElement,
} from 'react';
import type { NotificationItem, NotificationContextValue } from '../types/common';

export type NotificationTypeKey = 'success' | 'error' | 'warning' | 'info';

export const NotificationType: Record<string, NotificationTypeKey> = {
  SUCCESS: 'success',
  ERROR: 'error',
  WARNING: 'warning',
  INFO: 'info',
};

const NotificationContext = createContext<NotificationContextValue | null>(null);

export function NotificationProvider({ children }: { children: ReactNode }) {
  const [notifications, setNotifications] = useState<NotificationItem[]>([]);

  const addNotification = useCallback(
    (type: NotificationTypeKey, message: string, duration = 5000) => {
      const id = Date.now();
      setNotifications((prev) => [...prev, { id, type, message }]);

      if (duration > 0) {
        setTimeout(() => {
          setNotifications((prev) => prev.filter((n) => n.id !== id));
        }, duration);
      }

      return id;
    },
    []
  );

  const removeNotification = useCallback((id: number) => {
    setNotifications((prev) => prev.filter((n) => n.id !== id));
  }, []);

  const showSuccess = useCallback(
    (message: string, duration?: number) =>
      addNotification(NotificationType.SUCCESS, message, duration),
    [addNotification]
  );

  const showError = useCallback(
    (message: string, duration?: number) =>
      addNotification(NotificationType.ERROR, message, duration),
    [addNotification]
  );

  const showWarning = useCallback(
    (message: string, duration?: number) =>
      addNotification(NotificationType.WARNING, message, duration),
    [addNotification]
  );

  const showInfo = useCallback(
    (message: string, duration?: number) =>
      addNotification(NotificationType.INFO, message, duration),
    [addNotification]
  );

  const value: NotificationContextValue = {
    notifications,
    removeNotification,
    success: showSuccess,
    error: showError,
    warning: showWarning,
    info: showInfo,
  };

  return (
    <NotificationContext.Provider value={value}>
      {children}
      <NotificationContainer />
    </NotificationContext.Provider>
  );
}

function NotificationContainer() {
  const ctx = useContext(NotificationContext);
  if (!ctx) return null;
  const { notifications, removeNotification } = ctx;

  if (notifications.length === 0) return null;

  return (
    <div className="fixed top-4 right-4 z-[1000] space-y-2" role="alert" aria-live="polite">
      {notifications.map((notification) => (
        <Notification
          key={notification.id}
          type={notification.type}
          message={notification.message}
          onClose={() => removeNotification(notification.id)}
        />
      ))}
    </div>
  );
}

interface NotificationProps {
  type: NotificationTypeKey;
  message: string;
  onClose: () => void;
}

function Notification({ type, message, onClose }: NotificationProps) {
  const styles: Record<NotificationTypeKey, string> = {
    success: 'bg-green-500/20 border-green-500/50 text-green-400',
    error: 'bg-red-500/20 border-red-500/50 text-red-400',
    warning: 'bg-yellow-500/20 border-yellow-500/50 text-yellow-400',
    info: 'bg-blue-500/20 border-blue-500/50 text-blue-400',
  };

  const icons: Record<NotificationTypeKey, ReactElement> = {
    success: (
      <svg
        className="w-5 h-5"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
      >
        <polyline points="20 6 9 17 4 12"></polyline>
        <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
      </svg>
    ),
    error: (
      <svg
        className="w-5 h-5"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
      >
        <circle cx="12" cy="12" r="10"></circle>
        <line x1="15" y1="9" x2="9" y2="15"></line>
        <line x1="9" y1="9" x2="15" y2="15"></line>
      </svg>
    ),
    warning: (
      <svg
        className="w-5 h-5"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
      >
        <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
        <line x1="12" y1="9" x2="12" y2="13"></line>
        <line x1="12" y1="17" x2="12.01" y2="17"></line>
      </svg>
    ),
    info: (
      <svg
        className="w-5 h-5"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
      >
        <circle cx="12" cy="12" r="10"></circle>
        <line x1="12" y1="16" x2="12" y2="12"></line>
        <line x1="12" y1="8" x2="12.01" y2="8"></line>
      </svg>
    ),
  };

  return (
    <div
      className={`flex items-center gap-3 px-4 py-3 rounded-lg border shadow-lg min-w-[300px] max-w-[500px] animate-slide-in ${styles[type]}`}
      role="alertdialog"
      aria-label={`${type} notification`}
    >
      {icons[type]}
      <p className="flex-1 text-sm font-medium">{message}</p>
      <button
        onClick={onClose}
        className="flex-shrink-0 opacity-70 hover:opacity-100 transition-opacity"
        aria-label="Dismiss notification"
      >
        <svg
          className="w-4 h-4"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
        >
          <line x1="18" y1="6" x2="6" y2="18"></line>
          <line x1="6" y1="6" x2="18" y2="18"></line>
        </svg>
      </button>
    </div>
  );
}

export function useNotification(): NotificationContextValue {
  const context = useContext(NotificationContext);
  if (!context) {
    throw new Error('useNotification must be used within a NotificationProvider');
  }
  return context;
}

export default NotificationContext;
