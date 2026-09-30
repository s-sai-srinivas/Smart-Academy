/**
 * Quiz & Question Types
 */

export type QuizStatus = 'draft' | 'published' | 'closed' | 'archived';
export type QuestionType = 'mcq' | 'true_false' | 'multi_select' | 'short_answer' | 'fill_blank';
export type BloomLevel = 'remember' | 'understand' | 'apply' | 'analyze' | 'evaluate' | 'create';

export interface Quiz {
  id: number;
  title: string;
  description?: string;
  course_offering_id?: number;
  course_name?: string;
  status: QuizStatus;
  duration_minutes?: number;
  total_marks?: number;
  passing_marks?: number;
  start_time?: string;
  end_time?: string;
  created_by?: number;
  created_at?: string;
  updated_at?: string;
  question_count?: number;
  attempt_count?: number;
}

export interface Question {
  id: number;
  quiz_id?: number;
  type: QuestionType;
  text: string;
  marks: number;
  options?: QuestionOption[];
  correct_answer?: string | string[] | number | number[];
  explanation?: string;
  bloom_level?: BloomLevel;
  difficulty?: 'easy' | 'medium' | 'hard';
  order?: number;
}

export interface QuestionOption {
  id?: string | number;
  label: string;
  text: string;
}

export interface QuizAttempt {
  id: number;
  quiz_id: number;
  user_id?: number;
  started_at: string;
  submitted_at?: string;
  score?: number;
  total_marks?: number;
  status: 'in_progress' | 'submitted' | 'graded';
  answers?: QuizAnswer[];
}

export interface QuizAnswer {
  question_id: number;
  answer: string | string[] | number | number[];
  is_correct?: boolean;
  marks_obtained?: number;
}

export interface QuizAnalytics {
  quiz_id: number;
  total_attempts: number;
  average_score: number;
  highest_score: number;
  lowest_score: number;
  pass_rate: number;
  question_stats?: QuestionStat[];
}

export interface QuestionStat {
  question_id: number;
  correct_count: number;
  wrong_count: number;
  correct_percentage: number;
}

export interface StudentQuizResult {
  quiz_id: number;
  score: number;
  total_marks: number;
  percentage: number;
  status: string;
  answers?: QuizAnswer[];
  time_taken_seconds?: number;
}
