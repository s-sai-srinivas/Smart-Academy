import { useState, useCallback, useRef } from 'react';
import { jobAPI } from '../services/api';

export interface JobPollingOptions {
  onProgress?: (status: unknown) => void;
  onComplete?: (result: unknown) => void;
  onError?: (error: string) => void;
  timeout?: number;
}

export interface UseJobPollingReturn {
  pollJob: (jobId: string, options?: JobPollingOptions) => Promise<unknown>;
  isPolling: boolean;
  pollProgress: unknown;
  pollError: string | null;
  resetPoll: () => void;
  abortPoll: () => void;
}

/**
 * Custom hook for polling job queue execution status
 */
export function useJobPolling(): UseJobPollingReturn {
  const [isPolling, setIsPolling] = useState(false);
  const [pollProgress, setPollProgress] = useState<unknown>(null);
  const [pollError, setPollError] = useState<string | null>(null);
  const abortRef = useRef(false);

  const pollJob = useCallback(async (jobId: string, options: JobPollingOptions = {}) => {
    const { onProgress, onComplete, onError, timeout = 60000 } = options;

    setIsPolling(true);
    setPollProgress({ status: 'pending', message: 'Starting execution...' });
    setPollError(null);
    abortRef.current = false;

    try {
      const result = await jobAPI.pollUntilComplete(
        jobId,
        (status) => {
          if (abortRef.current) return;
          setPollProgress(status);
          if (onProgress) onProgress(status);
        },
        timeout
      );

      if (abortRef.current) {
        return null;
      }

      setIsPolling(false);
      setPollProgress(null);

      const resultObj = result as { status?: string; error_message?: string };
      if (resultObj.status === 'completed') {
        if (onComplete) onComplete(result);
        return result;
      } else if (resultObj.status === 'failed') {
        const error = resultObj.error_message || 'Execution failed';
        setPollError(error);
        if (onError) onError(error);
        throw new Error(error);
      }

      return result;
    } catch (error) {
      if (abortRef.current) {
        return null;
      }

      setIsPolling(false);
      setPollProgress(null);
      const errorMessage = (error as Error).message || 'Polling failed';
      setPollError(errorMessage);
      if (onError) onError(errorMessage);
      throw error;
    }
  }, []);

  const resetPoll = useCallback(() => {
    abortRef.current = true;
    setIsPolling(false);
    setPollProgress(null);
    setPollError(null);
  }, []);

  const abortPoll = useCallback(() => {
    abortRef.current = true;
    setIsPolling(false);
    setPollProgress(null);
  }, []);

  return {
    pollJob,
    isPolling,
    pollProgress,
    pollError,
    resetPoll,
    abortPoll,
  };
}

export default useJobPolling;
