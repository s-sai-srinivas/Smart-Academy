import { useState, useEffect, useRef, useCallback } from 'react';

export interface UseContestFullscreenOptions {
  warningDuration?: number;
  maxViolations?: number;
  promptKey?: string | null;
  enabled?: boolean;
}

export interface UseContestFullscreenReturn {
  isFullscreen: boolean;
  showInitialPrompt: boolean;
  showWarning: boolean;
  violations: number;
  warningTime: Date | null;
  maxViolations: number;
  requestFullscreen: () => Promise<void>;
}

/**
 * Custom hook for enforcing full-screen mode during contests
 */
export const useContestFullscreen = ({
  warningDuration: _warningDuration = 60000,
  maxViolations = 3,
  promptKey = null,
  enabled = true,
}: UseContestFullscreenOptions = {}): UseContestFullscreenReturn => {
  const [isFullscreen, setIsFullscreen] = useState(() => {
    if (!enabled) return true;
    try {
      return !!(
        document.fullscreenElement ||
        (document as Document & { webkitFullscreenElement?: Element }).webkitFullscreenElement ||
        (document as Document & { msFullscreenElement?: Element }).msFullscreenElement
      );
    } catch {
      return false;
    }
  });
  const [showInitialPrompt, setShowInitialPrompt] = useState(() => {
    if (!enabled) return false;
    if (!promptKey) return true;
    return localStorage.getItem(promptKey) !== '1';
  });
  const [showWarning, setShowWarning] = useState(false);
  const [violations, setViolations] = useState(0);
  const [warningTime, setWarningTime] = useState<Date | null>(null);

  const violationCountRef = useRef(0);
  const showInitialPromptRef = useRef(showInitialPrompt);
  const isFullscreenRef = useRef(false);
  const prevFullscreenRef = useRef(false);

  useEffect(() => {
    if (!enabled) {
      showInitialPromptRef.current = false;
      return;
    }
    if (!promptKey) return;
    const shouldShow = localStorage.getItem(promptKey) !== '1';
    showInitialPromptRef.current = shouldShow;
  }, [promptKey, enabled]);

  useEffect(() => {
    if (!enabled) {
      violationCountRef.current = 0;
      showInitialPromptRef.current = false;
      isFullscreenRef.current = true;
      prevFullscreenRef.current = true;
      return;
    }
  }, [enabled]);

  const requestFullscreen = useCallback(async () => {
    if (!enabled) return;
    try {
      const elem = document.documentElement;
      if (elem.requestFullscreen) {
        await elem.requestFullscreen();
      } else if (
        (elem as HTMLElement & { webkitRequestFullscreen?: () => Promise<void> })
          .webkitRequestFullscreen
      ) {
        await (
          elem as HTMLElement & { webkitRequestFullscreen: () => Promise<void> }
        ).webkitRequestFullscreen();
      } else if (
        (elem as HTMLElement & { msRequestFullscreen?: () => Promise<void> }).msRequestFullscreen
      ) {
        await (
          elem as HTMLElement & { msRequestFullscreen: () => Promise<void> }
        ).msRequestFullscreen();
      }
      showInitialPromptRef.current = false;
      setShowInitialPrompt(false);
      if (promptKey) {
        localStorage.setItem(promptKey, '1');
      }
    } catch (err) {
      console.warn('Fullscreen request failed:', err);
    }
  }, [promptKey, enabled]);

  const checkFullscreen = useCallback((): boolean => {
    const fullscreenElement =
      document.fullscreenElement ||
      (document as Document & { webkitFullscreenElement?: Element }).webkitFullscreenElement ||
      (document as Document & { msFullscreenElement?: Element }).msFullscreenElement;
    return !!fullscreenElement;
  }, []);

  const incrementViolation = useCallback(() => {
    violationCountRef.current += 1;
    setViolations(violationCountRef.current);
  }, []);

  useEffect(() => {
    if (!enabled) return;

    const initialFullscreen = checkFullscreen();
    isFullscreenRef.current = initialFullscreen;
    prevFullscreenRef.current = initialFullscreen;

    const handleFullscreenChange = () => {
      const inFullscreen = checkFullscreen();
      const wasFullscreen = prevFullscreenRef.current;

      prevFullscreenRef.current = inFullscreen;
      isFullscreenRef.current = inFullscreen;
      setIsFullscreen(inFullscreen);

      if (!inFullscreen && !showInitialPromptRef.current) {
        setShowWarning(true);
        setWarningTime(new Date());
        if (wasFullscreen) {
          incrementViolation();
        }
      } else if (inFullscreen) {
        setShowWarning(false);
        setWarningTime(null);
      }
    };

    document.addEventListener('fullscreenchange', handleFullscreenChange);
    document.addEventListener('webkitfullscreenchange', handleFullscreenChange);
    document.addEventListener('msfullscreenchange', handleFullscreenChange);

    return () => {
      document.removeEventListener('fullscreenchange', handleFullscreenChange);
      document.removeEventListener('webkitfullscreenchange', handleFullscreenChange);
      document.removeEventListener('msfullscreenchange', handleFullscreenChange);
    };
  }, [checkFullscreen, enabled, incrementViolation]);

  useEffect(() => {
    if (!enabled) return;
    if (showInitialPrompt) return;

    const handleContextMenu = (e: MouseEvent) => {
      e.preventDefault();
      return false;
    };

    document.addEventListener('contextmenu', handleContextMenu);
    return () => {
      document.removeEventListener('contextmenu', handleContextMenu);
    };
  }, [showInitialPrompt, enabled]);

  useEffect(() => {
    if (!enabled) return;
    if (showInitialPrompt) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && ['c', 'x', 'v', 'a'].includes(e.key.toLowerCase())) {
        e.preventDefault();
        return false;
      }
      if (e.key === 'F11') {
        e.preventDefault();
        return false;
      }
    };

    const handleCopyCut = (e: ClipboardEvent) => {
      e.preventDefault();
      return false;
    };

    document.addEventListener('keydown', handleKeyDown);
    document.addEventListener('copy', handleCopyCut);
    document.addEventListener('cut', handleCopyCut);

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.removeEventListener('copy', handleCopyCut);
      document.removeEventListener('cut', handleCopyCut);
    };
  }, [showInitialPrompt, enabled]);

  return {
    isFullscreen,
    showInitialPrompt,
    showWarning,
    violations,
    warningTime,
    maxViolations,
    requestFullscreen,
  };
};

export default useContestFullscreen;
