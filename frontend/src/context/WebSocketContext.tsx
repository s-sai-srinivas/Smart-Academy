import { createContext, useContext, type ReactNode } from 'react';
import { useWebSocket } from '../hooks/useWebSocket';
import type { UseWebSocketReturn } from '../hooks/useWebSocket';

/**
 * WebSocketContext
 *
 * Provides WebSocket connection state and methods to the app.
 */

const WebSocketContext = createContext<UseWebSocketReturn | null>(null);

export function WebSocketProvider({ children }: { children: ReactNode }) {
  const websocket = useWebSocket();

  return <WebSocketContext.Provider value={websocket}>{children}</WebSocketContext.Provider>;
}

export function useWebSocketContext(): UseWebSocketReturn {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error('useWebSocketContext must be used within a WebSocketProvider');
  }
  return context;
}

export default WebSocketContext;
