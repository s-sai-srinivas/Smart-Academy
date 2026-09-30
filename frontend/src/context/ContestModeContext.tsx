import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  useRef,
  type ReactNode,
} from 'react';
import type { ContestModeContextValue } from '../types/contest';

const STORAGE_KEY = 'active_contest_session';

interface StoredContestSession {
  contestId: number;
  startTime: string;
  endTime: string;
  title: string;
  hasFinished: boolean;
  storedAt: string;
}

const ContestModeContext = createContext<ContestModeContextValue | null>(null);

// Helper to get stored contest session from localStorage
const getStoredContestSession = (): StoredContestSession | null => {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (!stored) return null;
    const parsed = JSON.parse(stored) as StoredContestSession;
    if (parsed.endTime && new Date(parsed.endTime) > new Date()) {
      return parsed;
    }
    localStorage.removeItem(STORAGE_KEY);
    return null;
  } catch (e) {
    console.warn('Could not parse stored contest session:', e);
    localStorage.removeItem(STORAGE_KEY);
    return null;
  }
};

export function ContestModeProvider({ children }: { children: ReactNode }) {
  const storedSession = getStoredContestSession();

  const [isInContestMode, setIsInContestMode] = useState(() => storedSession?.contestId != null);
  const [activeContestId, setActiveContestId] = useState<number | null>(
    () => storedSession?.contestId || null
  );
  const [contestStartTime, setContestStartTime] = useState<string | null>(
    () => storedSession?.startTime || null
  );
  const [contestEndTime, setContestEndTime] = useState<string | null>(
    () => storedSession?.endTime || null
  );
  const [hasFinished, setHasFinished] = useState(() => storedSession?.hasFinished || false);
  const [contestTitle, setContestTitle] = useState(() => storedSession?.title || '');

  const isInContestModeRef = useRef(false);
  const activeContestIdRef = useRef<number | null>(null);
  const contestEndTimeRef = useRef<string | null>(null);
  const hasFinishedRef = useRef(false);

  useEffect(() => {
    isInContestModeRef.current = isInContestMode;
    activeContestIdRef.current = activeContestId;
    contestEndTimeRef.current = contestEndTime;
    hasFinishedRef.current = hasFinished;
  }, [isInContestMode, activeContestId, contestEndTime, hasFinished]);

  const setContestMode = useCallback(
    (contestId: number, startTime: string, endTime: string, title = '') => {
      setActiveContestId(contestId);
      setContestStartTime(startTime);
      setContestEndTime(endTime);
      setContestTitle(title);
      setHasFinished(false);
      setIsInContestMode(true);

      try {
        const session: StoredContestSession = {
          contestId,
          startTime,
          endTime,
          title,
          hasFinished: false,
          storedAt: new Date().toISOString(),
        };
        localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
      } catch (e) {
        console.warn('Could not save contest session to localStorage:', e);
      }
    },
    []
  );

  const exitContestMode = useCallback(() => {
    setIsInContestMode(false);
    setActiveContestId(null);
    setContestStartTime(null);
    setContestEndTime(null);
    setContestTitle('');
  }, []);

  const finishContest = useCallback(() => {
    setHasFinished(true);
    exitContestMode();
    try {
      localStorage.removeItem(STORAGE_KEY);
    } catch (e) {
      console.warn('Could not clear contest session from localStorage:', e);
    }
  }, [exitContestMode]);

  const isContestEnded = useCallback(() => {
    if (!contestEndTime) return false;
    return new Date() > new Date(contestEndTime);
  }, [contestEndTime]);

  const isAllowedPath = useCallback(
    (path: string) => {
      if (!activeContestId) return false;
      const contestIdStr = String(activeContestId);
      const allowedPatterns = [
        new RegExp(`^/contests/${contestIdStr}/problems$`),
        new RegExp(`^/contests/${contestIdStr}/problems/[^/]+$`),
        new RegExp(`^/contests/${contestIdStr}/leaderboard$`),
      ];
      return allowedPatterns.some((pattern) => pattern.test(path));
    },
    [activeContestId]
  );

  const getRedirectPath = useCallback(() => {
    if (!activeContestId) return '/contests';
    return `/contests/${activeContestId}/problems`;
  }, [activeContestId]);

  useEffect(() => {
    if (!isInContestMode || !contestEndTime || hasFinished) return;

    const checkTimeEnd = () => {
      if (new Date() > new Date(contestEndTime)) {
        setIsInContestMode(false);
        try {
          localStorage.removeItem(STORAGE_KEY);
        } catch (e) {
          console.warn('Could not clear contest session from localStorage:', e);
        }
      }
    };

    const interval = setInterval(checkTimeEnd, 30000);
    checkTimeEnd();
    return () => clearInterval(interval);
  }, [isInContestMode, contestEndTime, hasFinished]);

  const value: ContestModeContextValue = {
    isInContestMode,
    activeContestId,
    contestStartTime,
    contestEndTime,
    contestTitle,
    hasFinished,
    setContestMode,
    exitContestMode,
    finishContest,
    isContestEnded,
    isAllowedPath,
    getRedirectPath,
    isInContestModeRef,
    activeContestIdRef,
    contestEndTimeRef,
    hasFinishedRef,
  };

  return <ContestModeContext.Provider value={value}>{children}</ContestModeContext.Provider>;
}

export function useContestMode(): ContestModeContextValue {
  const context = useContext(ContestModeContext);
  if (!context) {
    throw new Error('useContestMode must be used within a ContestModeProvider');
  }
  return context;
}

export default ContestModeContext;
