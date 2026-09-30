/**
 * Problem, Test Case, Submission & Judge0 Types
 */

export type Difficulty = 'easy' | 'medium' | 'hard';

export interface Problem {
  id: number;
  title: string;
  description: string;
  input_format?: string;
  output_format?: string;
  constraints?: string;
  difficulty: Difficulty;
  topic_id?: number;
  topic_name?: string;
  time_limit?: number;
  memory_limit?: number;
  sample_test_cases?: TestCase[];
  hidden_test_cases?: TestCase[];
  is_practice?: boolean;
  author_id?: number;
  created_at?: string;
  updated_at?: string;
}

export interface TestCase {
  id?: number;
  input: string;
  expected_output: string;
  explanation?: string;
  is_sample?: boolean;
}

export interface Submission {
  id: number;
  problem_id: number;
  user_id?: number;
  language_id: number;
  source_code?: string;
  status_id: number;
  status?: string;
  time_taken?: number;
  memory_used?: number;
  created_at?: string;
  score?: number;
}

export interface RunResult {
  stdout?: string;
  stderr?: string;
  compile_output?: string;
  message?: string;
  status?: {
    id: number;
    description: string;
  };
  time?: string;
  memory?: number;
}

export interface Judge0Status {
  id: number;
  label: string;
  shortLabel: string;
  color: string;
  bgColor: string;
  icon: string;
}

export interface LanguageConfig {
  id: number;
  name: string;
  monacoLang: string;
  extension: string;
  defaultCode: string;
}

export interface PlagiarismResult {
  problem_id: number;
  similarity?: number;
  matches?: PlagiarismMatch[];
  report_url?: string;
  generated_at?: string;
}

export interface PlagiarismMatch {
  submission_id_1: number;
  submission_id_2: number;
  similarity: number;
  user_1?: string;
  user_2?: string;
}

export interface SolveSession {
  problem_id: number;
  user_regdno: string;
  start_time: string;
  end_time?: string;
  time_spent_seconds?: number;
}

export interface SubmissionContextValue {
  fetchSubmissions: (page?: number) => Promise<unknown>;
  fetchCode: (submissionId: number) => Promise<string>;
}
