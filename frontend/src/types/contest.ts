/**
 * Contest & Leaderboard Types
 */

import type { TestCase } from './problem';

export type ContestStatus = 'upcoming' | 'active' | 'ended';

export interface Contest {
  id: number;
  title: string;
  description?: string;
  start_time: string;
  end_time: string;
  status?: ContestStatus;
  duration_minutes?: number;
  total_problems?: number;
  has_joined?: boolean;
  is_practice_enabled?: boolean;
  created_by?: number;
  created_at?: string;
  rules?: string;
}

export interface ContestProblem {
  id: number;
  contest_id?: number;
  problem_id: number;
  title?: string;
  difficulty?: string;
  order?: number;
  points?: number;
  solved?: boolean;
  attempts?: number;
}

export interface LeaderboardEntry {
  rank: number;
  user_id?: number;
  regdno?: string;
  name?: string;
  score: number;
  problems_solved: number;
  total_time?: number;
  submissions?: number;
}

export interface ContestSubmission {
  id: number;
  contest_id: number;
  problem_id: number;
  user_id?: number;
  language_id: number;
  status_id: number;
  status?: string;
  created_at?: string;
  score?: number;
}

export interface ContestModeContextValue {
  isInContestMode: boolean;
  activeContestId: number | null;
  contestStartTime: string | null;
  contestEndTime: string | null;
  contestTitle: string;
  hasFinished: boolean;
  setContestMode: (contestId: number, startTime: string, endTime: string, title?: string) => void;
  exitContestMode: () => void;
  finishContest: () => void;
  isContestEnded: () => boolean;
  isAllowedPath: (path: string) => boolean;
  getRedirectPath: () => string;
  // Refs for event handlers
  isInContestModeRef: React.MutableRefObject<boolean>;
  activeContestIdRef: React.MutableRefObject<number | null>;
  contestEndTimeRef: React.MutableRefObject<string | null>;
  hasFinishedRef: React.MutableRefObject<boolean>;
}

export interface GeneratedProblem {
  index?: number;
  title?: string;
  description?: string;
  input_format?: string;
  output_format?: string;
  constraints?: string;
  difficulty?: string;
  test_cases?: TestCase[];
  reference_solution?: string;
  verification_status?: 'pending' | 'passed' | 'failed';
  verification_message?: string;
}

export interface ContestGenerationJob {
  job_id: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  progress?: number;
  message?: string;
  problems?: GeneratedProblem[];
  created_at?: string;
}
