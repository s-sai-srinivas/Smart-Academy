import { useEffect, useRef, useCallback, useState } from 'react';
import { useAuth } from '../context/AuthContext';
import { secureTokenStorage } from '../services/secureStorage';

export type WebSocketEventType = 'FORCE_LOGOUT' | 'PING' | 'PONG';

export const WebSocketEvents: Record<string, WebSocketEventType> = {
  FORCE_LOGOUT: 'FORCE_LOGOUT',
  PING: 'PING',
  PONG: 'PONG',
};

export interface WebSocketMessage {
  type: WebSocketEventType;
  data?: { message?: string };
}

export interface UseWebSocketReturn {
  isConnected: boolean;
  connectionError: string | null;
  on: (eventType: string, callback: (data: unknown) => void) => () => void;
  off: (eventType: string, callback?: ((data: unknown) => void) | null) => void;
  send: (type: string, data: unknown) => boolean;
  disconnect: () => void;
  reconnect: () => void;
}

// WebSocket endpoint is opt-in: the REST backend does not expose /api/ws and
// Vercel cannot proxy WS upgrades, so connecting to same-origin always fails.
// Set VITE_WS_URL (e.g. wss://api.example.com/api/ws) when a WS backend exists.
const WS_BASE_URL = import.meta.env.VITE_WS_URL as string | undefined;

/**
 * Custom hook for WebSocket connection with auto-reconnect
 */
export function useWebSocket(): UseWebSocketReturn {
  const { user, isAuthenticated, logout } = useAuth();
  const [isConnected, setIsConnected] = useState(false);
  const [connectionError, setConnectionError] = useState<string | null>(null);

  const socketRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const eventListenersRef = useRef(new Map<string, Set<(data: unknown) => void>>());
  const isUnmountedRef = useRef(false);
  const connectRef = useRef<() => void>(() => {});
  const disconnectRef = useRef<() => void>(() => {});

  const MAX_RECONNECT_ATTEMPTS = 10;
  const BASE_RECONNECT_DELAY = 1000;
  const MAX_RECONNECT_DELAY = 30000;

  const getReconnectDelay = useCallback(() => {
    const delay = Math.min(
      BASE_RECONNECT_DELAY * Math.pow(2, reconnectAttemptsRef.current),
      MAX_RECONNECT_DELAY
    );
    return delay + Math.random() * 1000;
  }, []);

  const handleMessage = useCallback(
    (event: MessageEvent) => {
      try {
        const message = JSON.parse(event.data) as WebSocketMessage;

        if (message.type === WebSocketEvents.PING) {
          if (socketRef.current?.readyState === WebSocket.OPEN) {
            socketRef.current.send(JSON.stringify({ type: WebSocketEvents.PONG }));
          }
          return;
        }

        if (message.type === WebSocketEvents.FORCE_LOGOUT) {
          console.log('WebSocket: Received FORCE_LOGOUT event');
          logout();
          alert(
            message.data?.message ||
              'Your session has been terminated. Please contact your administrator.'
          );
          return;
        }

        const listeners = eventListenersRef.current.get(message.type);
        if (listeners) {
          listeners.forEach((callback) => callback(message.data));
        }
      } catch (err) {
        console.error('WebSocket: Failed to parse message:', err);
      }
    },
    [logout]
  );

  const connect = useCallback(() => {
    if (!isAuthenticated || !user || isUnmountedRef.current || !WS_BASE_URL) {
      return;
    }

    if (
      socketRef.current?.readyState === WebSocket.OPEN ||
      socketRef.current?.readyState === WebSocket.CONNECTING
    ) {
      return;
    }

    const token = secureTokenStorage.getToken();
    const wsUrl =
      token && token !== 'httpOnly'
        ? `${WS_BASE_URL}?token=${encodeURIComponent(token)}`
        : WS_BASE_URL;

    try {
      setConnectionError(null);
      const socket = new WebSocket(wsUrl);

      socket.onopen = () => {
        console.log('WebSocket: Connected');
        setIsConnected(true);
        setConnectionError(null);
        reconnectAttemptsRef.current = 0;
      };

      socket.onmessage = handleMessage;

      socket.onclose = (event: CloseEvent) => {
        console.log('WebSocket: Disconnected', event.code, event.reason);
        setIsConnected(false);

        if (!isUnmountedRef.current && event.code !== 1000) {
          const delay = getReconnectDelay();
          console.log(
            `WebSocket: Reconnecting in ${delay}ms (attempt ${reconnectAttemptsRef.current + 1})`
          );

          reconnectTimeoutRef.current = setTimeout(() => {
            if (!isUnmountedRef.current && reconnectAttemptsRef.current < MAX_RECONNECT_ATTEMPTS) {
              reconnectAttemptsRef.current++;
              connectRef.current();
            } else if (reconnectAttemptsRef.current >= MAX_RECONNECT_ATTEMPTS) {
              console.error('WebSocket: Max reconnect attempts reached');
              setConnectionError('Unable to maintain connection. Please refresh the page.');
            }
          }, delay);
        }
      };

      socket.onerror = (error: Event) => {
        console.error('WebSocket: Error', error);
        setConnectionError('Connection error');
      };

      socketRef.current = socket;
    } catch (err) {
      console.error('WebSocket: Failed to create connection:', err);
      setConnectionError('Failed to connect');
    }
  }, [isAuthenticated, user, handleMessage, getReconnectDelay]);

  useEffect(() => {
    connectRef.current = connect;
  });

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (socketRef.current) {
      socketRef.current.close(1000, 'Intentional disconnect');
      socketRef.current = null;
    }

    setIsConnected(false);
  }, []);

  const on = useCallback((eventType: string, callback: (data: unknown) => void) => {
    const existing = eventListenersRef.current.get(eventType);
    if (existing) {
      existing.add(callback);
    } else {
      const s = new Set<(data: unknown) => void>();
      s.add(callback);
      eventListenersRef.current.set(eventType, s);
    }

    return () => {
      eventListenersRef.current.get(eventType)?.delete(callback);
    };
  }, []);

  const off = useCallback((eventType: string, callback?: ((data: unknown) => void) | null) => {
    if (callback) {
      eventListenersRef.current.get(eventType)?.delete(callback);
    } else {
      eventListenersRef.current.delete(eventType);
    }
  }, []);

  const send = useCallback((type: string, data: unknown): boolean => {
    if (socketRef.current?.readyState === WebSocket.OPEN) {
      socketRef.current.send(JSON.stringify({ type, data }));
      return true;
    }
    return false;
  }, []);

  useEffect(() => {
    isUnmountedRef.current = false;

    if (isAuthenticated) {
      connectRef.current();
    } else {
      disconnectRef.current();
    }

    return () => {
      isUnmountedRef.current = true;
      disconnectRef.current();
    };
  }, [isAuthenticated]);

  useEffect(() => {
    connectRef.current = connect;
    disconnectRef.current = disconnect;
  }, [connect, disconnect]);

  return {
    isConnected,
    connectionError,
    on,
    off,
    send,
    disconnect,
    reconnect: connect,
  };
}

export default useWebSocket;
