/**
 * Course, Curriculum & Academic Types
 */

export interface College {
  id: number;
  name: string;
  code?: string;
  is_active?: boolean;
  address?: string;
}

export interface Program {
  id: number;
  name: string;
  code?: string;
}

export interface Regulation {
  id: number;
  name: string;
  code?: string;
}

export interface Branch {
  id: number;
  name: string;
  code: string;
  department?: string;
}

export interface Section {
  id: number;
  name: string;
  batch_id?: number;
  branch_id?: number;
}

export interface Batch {
  id: number;
  year: number;
  name?: string;
}

export interface Curriculum {
  id: number;
  name: string;
  program_id: number;
  regulation_id: number;
  courses?: Course[];
}

export interface Course {
  id: number;
  name: string;
  code: string;
  credits?: number;
  semester?: number;
  type: 'theory' | 'lab';
  description?: string;
  curriculum_id?: number;
  // For enrolled courses
  is_enrolled?: boolean;
  progress?: number;
  completed_topics?: number;
  total_topics?: number;
}

export interface CourseOffering {
  course_offering_id: number;
  course_id: number;
  course_name?: string;
  section_id?: number;
  faculty_id?: number;
  semester?: number;
  year?: number;
}

export interface Topic {
  id: number;
  name: string;
  description?: string;
  course_id: number;
  order?: number;
  problem_count?: number;
}

export interface LabSession {
  id: number;
  name: string;
  course_id: number;
  topic_id?: number;
  date?: string;
  problems?: ProblemRef[];
}

export interface TheoryModule {
  id: number;
  name: string;
  description?: string;
  week_id?: number;
  order?: number;
  content?: string;
}

export interface TheoryWeek {
  id: number;
  week_number: number;
  title: string;
  description?: string;
  modules?: TheoryModule[];
  course_id?: number;
}

export interface ProblemRef {
  id: number;
  title: string;
  difficulty?: string;
}
