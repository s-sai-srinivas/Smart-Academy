import { useCallback, useEffect, useRef, useState, type PointerEvent } from 'react';

const clamp = (value: number, min: number, max: number): number =>
  Math.min(max, Math.max(min, value));

export interface UseResizablePanelsOptions {
  storageKey?: string;
  initialLeftPercent?: number;
  minLeftPx?: number;
  maxLeftPercent?: number;
  minRightPx?: number;
  initialConsoleHeight?: number;
  minConsoleHeight?: number;
  maxConsolePercent?: number;
  minEditorHeight?: number;
}

export interface UseResizablePanelsReturn {
  splitRef: React.RefObject<HTMLDivElement | null>;
  editorContainerRef: React.RefObject<HTMLDivElement | null>;
  leftWidthPercent: number;
  consoleHeight: number;
  isHorizontalDragging: boolean;
  isVerticalDragging: boolean;
  onHorizontalPointerDown: (event: PointerEvent<HTMLDivElement>) => void;
  onVerticalPointerDown: (event: PointerEvent<HTMLDivElement>) => void;
}

interface DragState {
  type: 'horizontal' | 'vertical' | null;
  latestClientX: number;
  latestClientY: number;
  rafId: number;
}

function useResizablePanels({
  storageKey,
  initialLeftPercent = 42,
  minLeftPx = 280,
  maxLeftPercent = 60,
  minRightPx = 360,
  initialConsoleHeight = 220,
  minConsoleHeight = 120,
  maxConsolePercent = 40,
  minEditorHeight = 180,
}: UseResizablePanelsOptions = {}): UseResizablePanelsReturn {
  const splitRef = useRef<HTMLDivElement | null>(null);
  const editorContainerRef = useRef<HTMLDivElement | null>(null);
  const dragStateRef = useRef<DragState>({
    type: null,
    latestClientX: 0,
    latestClientY: 0,
    rafId: 0,
  });

  const [leftWidthPercent, setLeftWidthPercent] = useState(() => {
    if (!storageKey) return initialLeftPercent;
    try {
      const raw = window.localStorage.getItem(storageKey);
      if (!raw) return initialLeftPercent;
      const parsed = JSON.parse(raw) as { leftWidthPercent?: number };
      return typeof parsed.leftWidthPercent === 'number'
        ? parsed.leftWidthPercent
        : initialLeftPercent;
    } catch {
      return initialLeftPercent;
    }
  });
  const [consoleHeight, setConsoleHeight] = useState(() => {
    if (!storageKey) return initialConsoleHeight;
    try {
      const raw = window.localStorage.getItem(storageKey);
      if (!raw) return initialConsoleHeight;
      const parsed = JSON.parse(raw) as { consoleHeight?: number };
      return typeof parsed.consoleHeight === 'number' ? parsed.consoleHeight : initialConsoleHeight;
    } catch {
      return initialConsoleHeight;
    }
  });
  const [isHorizontalDragging, setIsHorizontalDragging] = useState(false);
  const [isVerticalDragging, setIsVerticalDragging] = useState(false);

  const getLeftLimits = useCallback(() => {
    if (!splitRef.current) {
      return { minLeft: minLeftPx, maxLeft: minLeftPx };
    }
    const rect = splitRef.current.getBoundingClientRect();
    const maxByPercent = rect.width * (maxLeftPercent / 100);
    let maxLeft = Math.min(maxByPercent, rect.width - minRightPx);
    if (maxLeft < minLeftPx) {
      maxLeft = minLeftPx;
    }
    return { minLeft: minLeftPx, maxLeft };
  }, [maxLeftPercent, minLeftPx, minRightPx]);

  const getConsoleLimits = useCallback(() => {
    if (!editorContainerRef.current) {
      return { minConsole: minConsoleHeight, maxConsole: minConsoleHeight };
    }
    const rect = editorContainerRef.current.getBoundingClientRect();
    const maxByPercent = rect.height * (maxConsolePercent / 100);
    let maxConsole = Math.min(maxByPercent, rect.height - minEditorHeight);
    if (maxConsole < minConsoleHeight) {
      maxConsole = minConsoleHeight;
    }
    return { minConsole: minConsoleHeight, maxConsole };
  }, [maxConsolePercent, minConsoleHeight, minEditorHeight]);

  const applyLeftFromClientX = useCallback(
    (clientX: number) => {
      if (!splitRef.current) return;
      const rect = splitRef.current.getBoundingClientRect();
      const { minLeft, maxLeft } = getLeftLimits();
      const rawLeft = clientX - rect.left;
      const clamped = clamp(rawLeft, minLeft, maxLeft);
      const nextPercent = (clamped / rect.width) * 100;
      setLeftWidthPercent(nextPercent);
    },
    [getLeftLimits]
  );

  const applyConsoleFromClientY = useCallback(
    (clientY: number) => {
      if (!editorContainerRef.current) return;
      const rect = editorContainerRef.current.getBoundingClientRect();
      const { minConsole, maxConsole } = getConsoleLimits();
      const rawConsoleHeight = rect.bottom - clientY;
      const clamped = clamp(rawConsoleHeight, minConsole, maxConsole);
      setConsoleHeight(clamped);
    },
    [getConsoleLimits]
  );

  const runDragUpdate = useCallback(() => {
    const state = dragStateRef.current;
    if (state.type === 'horizontal') {
      applyLeftFromClientX(state.latestClientX);
    } else if (state.type === 'vertical') {
      applyConsoleFromClientY(state.latestClientY);
    }
    state.rafId = 0;
  }, [applyLeftFromClientX, applyConsoleFromClientY]);

  const handlePointerMove = useCallback(
    (event: globalThis.PointerEvent) => {
      const state = dragStateRef.current;
      if (!state.type) return;
      state.latestClientX = event.clientX;
      state.latestClientY = event.clientY;
      if (!state.rafId) {
        state.rafId = window.requestAnimationFrame(runDragUpdate);
      }
    },
    [runDragUpdate]
  );

  const endDrag = useCallback(() => {
    const state = dragStateRef.current;
    if (!state.type) return;
    state.type = null;
    if (state.rafId) {
      window.cancelAnimationFrame(state.rafId);
      state.rafId = 0;
    }
    document.body.classList.remove('no-select', 'resizing-col', 'resizing-row');
    setIsHorizontalDragging(false);
    setIsVerticalDragging(false);

    if (storageKey) {
      try {
        const payload = { leftWidthPercent, consoleHeight };
        window.localStorage.setItem(storageKey, JSON.stringify(payload));
      } catch {
        // Ignore storage errors
      }
    }
  }, [consoleHeight, leftWidthPercent, storageKey]);

  const startHorizontalDrag = useCallback(
    (event: PointerEvent<HTMLDivElement>) => {
      event.preventDefault();
      dragStateRef.current.type = 'horizontal';
      dragStateRef.current.latestClientX = event.clientX;
      document.body.classList.add('no-select', 'resizing-col');
      setIsHorizontalDragging(true);
      applyLeftFromClientX(event.clientX);
    },
    [applyLeftFromClientX]
  );

  const startVerticalDrag = useCallback(
    (event: PointerEvent<HTMLDivElement>) => {
      event.preventDefault();
      dragStateRef.current.type = 'vertical';
      dragStateRef.current.latestClientY = event.clientY;
      document.body.classList.add('no-select', 'resizing-row');
      setIsVerticalDragging(true);
      applyConsoleFromClientY(event.clientY);
    },
    [applyConsoleFromClientY]
  );

  useEffect(() => {
    window.addEventListener('pointermove', handlePointerMove);
    window.addEventListener('pointerup', endDrag);
    return () => {
      window.removeEventListener('pointermove', handlePointerMove);
      window.removeEventListener('pointerup', endDrag);
    };
  }, [endDrag, handlePointerMove]);

  useEffect(() => {
    const handleResize = () => {
      if (splitRef.current) {
        const rect = splitRef.current.getBoundingClientRect();
        const { minLeft, maxLeft } = getLeftLimits();
        setLeftWidthPercent((current) => {
          const rawLeft = (current / 100) * rect.width;
          const clampedLeft = clamp(rawLeft, minLeft, maxLeft);
          return (clampedLeft / rect.width) * 100;
        });
      }
      if (editorContainerRef.current) {
        const { minConsole, maxConsole } = getConsoleLimits();
        setConsoleHeight((current) => clamp(current, minConsole, maxConsole));
      }
    };
    handleResize();
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, [getLeftLimits, getConsoleLimits]);

  return {
    splitRef,
    editorContainerRef,
    leftWidthPercent,
    consoleHeight,
    isHorizontalDragging,
    isVerticalDragging,
    onHorizontalPointerDown: startHorizontalDrag,
    onVerticalPointerDown: startVerticalDrag,
  };
}

export default useResizablePanels;
