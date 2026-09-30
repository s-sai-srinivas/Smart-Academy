import { createContext, useContext, useCallback, useMemo, type ReactNode } from 'react';
import { submissionsAPI } from '../services/api';
import { secureTokenStorage } from '../services/secureStorage';
import type { SubmissionContextValue } from '../types/problem';

const SubmissionContext = createContext<SubmissionContextValue | null>(null);

export function useSubmissionContext(
  problemId?: string | number | null
): SubmissionContextValue | null {
  const context = useContext(SubmissionContext);
  if (!context && problemId) {
    throw new Error('useSubmissionContext must be used within a SubmissionProvider');
  }
  return context;
}

export function SubmissionProvider({
  problemId,
  children,
}: {
  problemId: string | number;
  children: ReactNode;
}) {
  const fetchSubmissions = useCallback(
    (page?: number) => submissionsAPI.getForProblem(problemId, page),
    [problemId]
  );

  const fetchCode = useCallback(async (submissionId: number): Promise<string> => {
    const token = secureTokenStorage.getToken();
    const authValue = token === 'httpOnly' ? 'httpOnly' : `Bearer ${token}`;
    const response = await fetch(`/api/submissions/${submissionId}/code`, {
      headers: { Authorization: authValue },
    });
    if (!response.ok) throw new Error('Failed to load code');
    const data = (await response.json()) as { source_code: string };
    return data.source_code;
  }, []);

  const value = useMemo<SubmissionContextValue>(
    () => ({
      fetchSubmissions,
      fetchCode,
    }),
    [fetchSubmissions, fetchCode]
  );

  return <SubmissionContext.Provider value={value}>{children}</SubmissionContext.Provider>;
}

export function useSubmissions(): SubmissionContextValue {
  const context = useContext(SubmissionContext);
  if (!context) {
    throw new Error('useSubmissions must be used within a SubmissionProvider');
  }
  return context;
}

export default SubmissionContext;
