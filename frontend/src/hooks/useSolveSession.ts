import { useState, useEffect, useRef, useCallback } from 'react';
import { secureTokenStorage } from '../services/secureStorage';
import { STORAGE_KEYS, formatTime } from '../constants/judge0';

export interface UseSolveSessionOptions {
  problemId: string | number;
  userRegdNo?: string;
  isCompleted?: boolean;
  onComplete?: () => void;
}

export interface UseSolveSessionReturn {
  showSolvePopup: boolean;
  isSolving: boolean;
  timer: number;
  sessionId: string | null;
  isLoading: boolean;
  error: string | null;
  formattedTime: string;
  startSolving: () => Promise<void>;
  endSolving: () => Promise<void>;
  resetSolving: () => void;
  getTimeTaken: () => number;
  canSolve: boolean;
  hasActiveSession: boolean;
}

interface StoredSession {
  sessionId: string;
  startedAt: string;
  problemId: string | number;
}

/**
 * Custom hook for managing solve sessions with timer and server sync
 */
export const useSolveSession = ({
  problemId,
  userRegdNo,
  isCompleted = false,
  onComplete: _onComplete,
}: UseSolveSessionOptions): UseSolveSessionReturn => {
  const [showSolvePopup, setShowSolvePopup] = useState(true);
  const [isSolving, setIsSolving] = useState(false);
  const [timer, setTimer] = useState(0);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const sessionStartRef = useRef<Date | null>(null);
  const syncIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const saveSessionToStorage = useCallback(
    (sessId: string, startedAt: string) => {
      try {
        const key = STORAGE_KEYS.SOLVE_SESSION(problemId, userRegdNo || 'anonymous');
        const stored: StoredSession = { sessionId: sessId, startedAt, problemId };
        localStorage.setItem(key, JSON.stringify(stored));
      } catch (e) {
        console.warn('Could not save session to storage:', e);
      }
    },
    [problemId, userRegdNo]
  );

  const clearSessionStorage = useCallback(() => {
    try {
      const key = STORAGE_KEYS.SOLVE_SESSION(problemId, userRegdNo || 'anonymous');
      localStorage.removeItem(key);
    } catch (e) {
      console.warn('Could not clear session storage:', e);
    }
  }, [problemId, userRegdNo]);

  const clearCodeStorage = useCallback(() => {
    try {
      const key = STORAGE_KEYS.PROBLEM_CODE(problemId, userRegdNo || 'anonymous');
      localStorage.removeItem(key);
    } catch (e) {
      console.warn('Could not clear code storage:', e);
    }
  }, [problemId, userRegdNo]);

  const startSolving = useCallback(async () => {
    if (!userRegdNo) {
      setShowSolvePopup(false);
      setIsSolving(true);
      setTimer(0);
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const token = secureTokenStorage.getToken();
      const response = await fetch('/api/problems/' + problemId + '/start', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        throw new Error('Failed to start session');
      }

      const data = (await response.json()) as { session_id: string; started_at: string };
      setSessionId(data.session_id);
      sessionStartRef.current = new Date(data.started_at);

      setShowSolvePopup(false);
      setIsSolving(true);
      setTimer(0);
      saveSessionToStorage(data.session_id, data.started_at);
    } catch (err) {
      console.error('Failed to start solve session:', err);
      setError((err as Error).message);
      setShowSolvePopup(false);
      setIsSolving(true);
      setTimer(0);
    } finally {
      setIsLoading(false);
    }
  }, [problemId, userRegdNo, saveSessionToStorage]);

  const endSolving = useCallback(async () => {
    setIsSolving(false);
    if (timerRef.current) clearInterval(timerRef.current);
    if (syncIntervalRef.current) clearInterval(syncIntervalRef.current);
    clearSessionStorage();
    setSessionId(null);
    sessionStartRef.current = null;
  }, [clearSessionStorage]);

  const resetSolving = useCallback(() => {
    endSolving();
    setTimer(0);
    setShowSolvePopup(true);
    clearCodeStorage();
  }, [endSolving, clearCodeStorage]);

  const getTimeTaken = useCallback((): number => {
    if (sessionStartRef.current) {
      return Math.floor((Date.now() - sessionStartRef.current.getTime()) / 1000);
    }
    return timer;
  }, [timer]);

  const loadExistingSession = useCallback(() => {
    if (!userRegdNo || isCompleted) return;

    try {
      const key = STORAGE_KEYS.SOLVE_SESSION(problemId, userRegdNo);
      const saved = localStorage.getItem(key);

      if (saved) {
        const session = JSON.parse(saved) as StoredSession;
        const sessionAge = Date.now() - new Date(session.startedAt).getTime();
        const maxAge = 24 * 60 * 60 * 1000; // 24 hours

        if (sessionAge < maxAge) {
          setSessionId(session.sessionId);
          sessionStartRef.current = new Date(session.startedAt);
          setTimer(Math.floor(sessionAge / 1000));
          setShowSolvePopup(false);
          setIsSolving(true);
        } else {
          clearSessionStorage();
        }
      }
    } catch (e) {
      console.warn('Could not load existing session:', e);
    }
  }, [problemId, userRegdNo, isCompleted, clearSessionStorage]);

  useEffect(() => {
    if (isSolving && sessionStartRef.current) {
      timerRef.current = setInterval(() => {
        const startTime = sessionStartRef.current;
        if (!startTime) return;
        const elapsed = Math.floor((Date.now() - startTime.getTime()) / 1000);
        setTimer(elapsed);
      }, 1000);

      syncIntervalRef.current = setInterval(() => {
        // Optional: Send heartbeat to server to keep session alive
      }, 30000);
    }

    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
      if (syncIntervalRef.current) clearInterval(syncIntervalRef.current);
    };
  }, [isSolving]);

  useEffect(() => {
    loadExistingSession();
  }, [loadExistingSession]);

  useEffect(() => {
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === STORAGE_KEYS.SOLVE_SESSION(problemId, userRegdNo || 'anonymous')) {
        if (!e.newValue) {
          endSolving();
        }
      }
    };

    window.addEventListener('storage', handleStorageChange);
    return () => window.removeEventListener('storage', handleStorageChange);
  }, [problemId, userRegdNo, endSolving]);

  return {
    showSolvePopup,
    isSolving,
    timer,
    sessionId,
    isLoading,
    error,
    formattedTime: formatTime(timer),
    startSolving,
    endSolving,
    resetSolving,
    getTimeTaken,
    canSolve: !isCompleted,
    hasActiveSession: !!sessionId,
  };
};

export default useSolveSession;
