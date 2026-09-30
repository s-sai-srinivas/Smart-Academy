import axios, { type InternalAxiosRequestConfig, type AxiosResponse } from 'axios';
import { secureTokenStorage } from './secureStorage';
import type { User } from '../types/auth';

const API_BASE_URL = '/api';

// Extend axios config to support skipAuth
interface ApiConfig extends InternalAxiosRequestConfig {
  skipAuth?: boolean;
}

// Custom error shape from our response interceptor
interface ApiError extends Error {
  status?: number;
  code?: string;
  originalError?: unknown;
  originalResponse?: unknown;
  config?: unknown;
}

// Create axios instance
const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor to add auth token
// SECURITY: Token retrieved from secure storage (sessionStorage or httpOnly cookie)
api.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = secureTokenStorage.getToken();
  const apiConfig = config as ApiConfig;
  if (token && token !== 'httpOnly' && !apiConfig.skipAuth) {
    apiConfig.headers.Authorization = `Bearer ${token}`;
  }
  return apiConfig;
});

// Response interceptor for error handling
// Preserves error details and provides structured error information
api.interceptors.response.use(
  (response: AxiosResponse) => response.data,
  (error) => {
    const status = error.response?.status;
    const data = error.response?.data;
    // Preserve original error message from backend if available
    const message = data?.error || data?.message || error.message || 'Request failed';
    const errorCode = data?.code;

    // Log error details for debugging (not exposed to user)
    console.error(`API Error [${status}]:`, {
      message,
      status,
      path: error.config?.url,
      method: error.config?.method,
    });

    // Handle token expiration - automatic logout
    if (status === 401 && errorCode === 'TOKEN_EXPIRED') {
      console.warn('Token expired - logging out user');
      // Clear auth data
      secureTokenStorage.clear();
      // Redirect to login page
      window.location.href = '/login';
      return Promise.reject(new Error('Session expired. Please login again.'));
    }

    // Handle general 401 errors (invalid token, revoked token, etc.)
    if (
      status === 401 &&
      (message.includes('expired') || message.includes('revoked') || message.includes('Invalid'))
    ) {
      console.warn('Authentication failed - logging out user');
      secureTokenStorage.clear();
      window.location.href = '/login';
    }

    // Create error with all relevant details preserved
    const apiError = new Error(message) as ApiError;
    apiError.status = status;
    apiError.originalError = error;
    apiError.originalResponse = data;
    apiError.config = error.config;
    apiError.code = errorCode;

    return Promise.reject(apiError);
  }
);

// Auth API
export const authAPI = {
  getColleges: (): Promise<unknown> => api.get('/colleges', { skipAuth: true } as ApiConfig),

  login: async (
    regdno: string,
    password: string,
    college_id: number | string | null = null
  ): Promise<unknown> => {
    const payload: Record<string, unknown> = { regdno, password };
    if (college_id) payload.college_id = college_id;
    const data = (await api.post('/auth/login', payload, {
      skipAuth: true,
    } as ApiConfig)) as unknown as Record<string, unknown>;
    // Use secure storage instead of localStorage
    secureTokenStorage.setToken(data.token as string);
    secureTokenStorage.setUser(data.user as User);
    return data;
  },

  logout: (): void => {
    secureTokenStorage.clear();
  },

  getCurrentUser: (): unknown => {
    return secureTokenStorage.getUser();
  },

  isAuthenticated: (): boolean => secureTokenStorage.getToken() !== null,
};

// Problems API
export const problemsAPI = {
  getAll: (): Promise<unknown> => api.get('/problems'),

  getById: (id: number | string): Promise<unknown> => api.get(`/problems/${id}`),

  create: (data: unknown): Promise<unknown> => api.post('/admin/problems', data),

  update: (id: number | string, data: unknown): Promise<unknown> =>
    api.put(`/admin/problems/${id}`, data),

  delete: (id: number | string): Promise<unknown> => api.delete(`/admin/problems/${id}`),

  getTestCases: (problemId: number | string): Promise<unknown> =>
    api.get(`/admin/problems/${problemId}/testcases`),

  createTestCase: (problemId: number | string, data: unknown): Promise<unknown> =>
    api.post(`/admin/problems/${problemId}/testcases`, data),

  deleteTestCase: (problemId: number | string, testCaseId: number | string): Promise<unknown> =>
    api.delete(`/admin/problems/${problemId}/testcases/${testCaseId}`),
};

// Submissions API
export const submissionsAPI = {
  submit: (
    problemId: number | string,
    languageId: number | string,
    sourceCode: string,
    timeTaken: number
  ): Promise<unknown> =>
    api.post('/submit', {
      problem_id: problemId,
      language_id: languageId,
      source_code: sourceCode,
      time_taken: timeTaken,
    }),

  run: (
    problemId: number | string,
    languageId: number | string,
    sourceCode: string
  ): Promise<unknown> =>
    api.post('/run', { problem_id: problemId, language_id: languageId, source_code: sourceCode }),

  getForProblem: (problemId: number | string, page = 1, limit = 20): Promise<unknown> =>
    api.get(`/problems/${problemId}/submissions?page=${page}&limit=${limit}`),

  getCompleted: (): Promise<unknown> => api.get('/my/completed-problems'),

  checkCompletion: (problemId: number | string): Promise<unknown> =>
    api.get(`/my/problems/${problemId}/completed`),
};

// Plagiarism API
export const plagiarismAPI = {
  checkProblem: (
    problemId: number | string,
    languageId: number | string | null = null
  ): Promise<unknown> => {
    let url = `/plagiarism/problems/${problemId}`;
    if (languageId) url += `?language_id=${languageId}`;
    return api.get(url);
  },

  checkSubmission: (submissionId: number | string): Promise<unknown> =>
    api.get(`/plagiarism/submissions/${submissionId}`),

  getResults: (problemId: number | string): Promise<unknown> =>
    api.get(`/plagiarism/results/${problemId}`),
};

// Dashboard API
export const dashboardAPI = {
  getData: (): Promise<unknown> => api.get('/dashboard'),
};

// Profile API
export const profileAPI = {
  getProfile: (): Promise<unknown> => api.get('/profile'),
};

// Courses API
export const coursesAPI = {
  getAll: (): Promise<unknown> => api.get('/courses'), // Send auth token to get enrolled courses for students
  getById: (id: number | string): Promise<unknown> => api.get(`/courses/${id}`),
  getTopics: (courseId: number | string): Promise<unknown> =>
    api.get(`/courses/${courseId}/topics`),
  getTopicProblems: (topicId: number | string): Promise<unknown> =>
    api.get(`/topics/${topicId}/problems`),
  getTopicsOverview: (): Promise<unknown> => api.get('/topics-overview'),
  // Practice quiz for theory weeks — student-accessible route
  getPracticeQuiz: (weekId: number | string): Promise<unknown> =>
    api.get(`/theory/weeks/${weekId}/practice-quiz`),
  // Theory PDFs — student-accessible route
  getTheoryPDFs: (theoryId: number | string): Promise<unknown> =>
    api.get(`/theories/${theoryId}/pdfs`),
};

// Faculty Analytics API
export const facultyAnalyticsAPI = {
  getCourseAnalytics: (courseId: number | string): Promise<unknown> =>
    api.get(`/faculty/courses/${courseId}/analytics`),
  getSectionAnalytics: (courseId: number | string, sectionId: number | string): Promise<unknown> =>
    api.get(`/faculty/courses/${courseId}/sections/${sectionId}/analytics`),
  getStudentAnalytics: (studentId: number | string): Promise<unknown> =>
    api.get(`/faculty/students/${studentId}/analytics`),
};

// HOD Analytics API
export const hodAnalyticsAPI = {
  getCourseAnalytics: (courseId: number | string): Promise<unknown> =>
    api.get(`/hod/analytics/course/${courseId}`),
  getSectionAnalytics: (courseId: number | string, sectionId: number | string): Promise<unknown> =>
    api.get(`/hod/analytics/course/${courseId}/section/${sectionId}`),
  getStudentAnalytics: (studentId: number | string): Promise<unknown> =>
    api.get(`/hod/analytics/student/${studentId}`),
};

// Principal API
export const principalAPI = {
  getStats: (): Promise<unknown> => api.get('/principal/stats'),
  getDepartments: (): Promise<unknown> => api.get('/principal/departments'),
  assignHOD: (branchId: number | string, facultyRegdNo: string): Promise<unknown> =>
    api.post('/principal/assign-hod', { branch_id: branchId, faculty_regdno: facultyRegdNo }),
  removeHOD: (branchId: number | string): Promise<unknown> =>
    api.post('/principal/remove-hod', { branch_id: branchId }),
};

// Principal Analytics API
export const principalAnalyticsAPI = {
  getOverview: (): Promise<unknown> => api.get('/principal/analytics/overview'),
  getCourseAnalytics: (): Promise<unknown> => api.get('/principal/analytics/courses'),
  getLabAnalytics: (): Promise<unknown> => api.get('/principal/analytics/labs'),
  getContestAnalytics: (): Promise<unknown> => api.get('/principal/analytics/contests'),
  getBranchAnalytics: (): Promise<unknown> => api.get('/principal/analytics/branches'),
  getStudentAnalytics: (): Promise<unknown> => api.get('/principal/analytics/students'),
  getYearAnalytics: (): Promise<unknown> => api.get('/principal/analytics/years'),
  getInsights: (): Promise<unknown> => api.get('/principal/analytics/insights'),
  getAlerts: (): Promise<unknown> => api.get('/principal/analytics/alerts'),
  getAccreditation: (): Promise<unknown> => api.get('/principal/analytics/accreditation'),
};

// Admin Principal API
export const adminPrincipalAPI = {
  create: (data: unknown): Promise<unknown> => api.post('/admin/principal/create', data),
};

// Curriculum API
export const curriculumAPI = {
  getAll: (): Promise<unknown> => api.get('/admin/curriculums'),
  create: (data: unknown): Promise<unknown> => api.post('/admin/curriculums', data),
  getCourses: (curriculumId: number | string): Promise<unknown> =>
    api.get(`/admin/curriculums/${curriculumId}/courses`),
  addCourse: (curriculumId: number | string, data: unknown): Promise<unknown> =>
    api.post(`/admin/curriculums/${curriculumId}/courses`, data),
  removeCourse: (curriculumId: number | string, courseId: number | string): Promise<unknown> =>
    api.delete(`/admin/curriculums/${curriculumId}/courses/${courseId}`),
};

// Super Admin API
export const superAdminAPI = {
  listColleges: (active?: boolean): Promise<unknown> => {
    const params = active !== undefined ? `?active=${active}` : '';
    return api.get(`/super-admin/colleges${params}`);
  },
  getCollege: (id: number | string): Promise<unknown> => api.get(`/super-admin/colleges/${id}`),
  createCollege: (data: unknown): Promise<unknown> => api.post('/super-admin/colleges', data),
  updateCollegeStatus: (id: number | string, isActive: boolean): Promise<unknown> =>
    api.put(`/super-admin/colleges/${id}/status`, { is_active: isActive }),
};

// Reference Data API
export const referenceAPI = {
  getPrograms: (): Promise<unknown> => api.get('/programs'),
  getRegulations: (): Promise<unknown> => api.get('/regulations'),
  getBranches: (): Promise<unknown> => api.get('/branches'),
  getSections: (): Promise<unknown> => api.get('/sections'),
  getBatches: (): Promise<unknown> => api.get('/batches'),
  getTopics: (subjectId?: number | string): Promise<unknown> => {
    const params = subjectId ? `?subject_id=${subjectId}` : '';
    return api.get(`/topics${params}`);
  },
  getSubjects: (includeTopics = false): Promise<unknown> => {
    const params = includeTopics ? '?include_topics=true' : '';
    return api.get(`/subjects${params}`);
  },
};

// Contests API
export const contestsAPI = {
  getAll: (): Promise<unknown> => api.get('/contests'),
  getById: (id: number | string): Promise<unknown> => api.get(`/contests/${id}`),
  join: (id: number | string): Promise<unknown> => api.post(`/contests/${id}/join`),
  finish: (id: number | string): Promise<unknown> => api.post(`/contests/${id}/finish`),
  getLeaderboard: (id: number | string): Promise<unknown> => api.get(`/contests/${id}/leaderboard`),
  submitSolution: (
    contestId: number | string,
    problemId: number | string,
    languageId: number | string,
    sourceCode: string
  ): Promise<unknown> =>
    api.post(`/contests/${contestId}/problems/${problemId}/submit`, {
      language_id: languageId,
      source_code: sourceCode,
    }),
  reportEscViolation: (contestId: number | string): Promise<unknown> =>
    api.post(`/contests/${contestId}/violations/esc`),
  // Student – get a single problem within a contest (validates participation)
  getProblem: (contestId: number | string, problemId: number | string): Promise<unknown> =>
    api.get(`/contests/${contestId}/problems/${problemId}`),
  // Student – get own submissions for a contest problem
  getSubmissions: (
    contestId: number | string,
    problemId: number | string,
    page = 1,
    limit = 20
  ): Promise<unknown> =>
    api.get(`/contests/${contestId}/problems/${problemId}/submissions?page=${page}&limit=${limit}`),
  // Lazy load source code for a contest submission
  getSubmissionCode: (submissionId: number | string): Promise<unknown> =>
    api.get(`/contest-submissions/${submissionId}/code`),

  // Faculty/HOD - Download contest results CSV
  downloadResultsCSV: async (contestId: number | string): Promise<void> => {
    const response = await api.get(`/faculty/contests/${contestId}/results/csv`, {
      responseType: 'blob',
    } as ApiConfig);
    // Create download link
    const url = window.URL.createObjectURL(new Blob([response as unknown as BlobPart]));
    const link = document.createElement('a');
    link.href = url;
    link.setAttribute('download', `contest_${contestId}_results_overview.csv`);
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.URL.revokeObjectURL(url);
  },

  // Faculty/HOD - Download detailed contest results CSV
  downloadDetailedCSV: async (contestId: number | string): Promise<void> => {
    const response = await api.get(`/faculty/contests/${contestId}/results/detailed-csv`, {
      responseType: 'blob',
    } as ApiConfig);
    // Create download link
    const url = window.URL.createObjectURL(new Blob([response as unknown as BlobPart]));
    const link = document.createElement('a');
    link.href = url;
    link.setAttribute('download', `contest_${contestId}_results_detailed.csv`);
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.URL.revokeObjectURL(url);
  },

  // Faculty/HOD - Get plagiarism analysis (cached results)
  getPlagiarism: (contestId: number | string): Promise<unknown> =>
    api.get(`/faculty/contests/${contestId}/plagiarism`),

  // Faculty/HOD - Run plagiarism check (POST to force fresh analysis)
  checkPlagiarism: (contestId: number | string): Promise<unknown> =>
    api.post(`/faculty/contests/${contestId}/plagiarism/check`),

  // Faculty/HOD - Get eligible students who haven't joined the contest
  getNonParticipants: (contestId: number | string): Promise<unknown> =>
    api.get(`/faculty/contests/${contestId}/non-participants`),

  // HOD - Get all contests for HOD's college (college-level, no branch filtering)
  getHODContests: (): Promise<unknown> => api.get('/hod/contests'),

  // Admin endpoints
  create: (data: unknown): Promise<unknown> => api.post('/admin/contests', data),
  update: (id: number | string, data: unknown): Promise<unknown> =>
    api.put(`/admin/contests/${id}`, data),
  delete: (id: number | string): Promise<unknown> => api.delete(`/admin/contests/${id}`),
  addProblem: (contestId: number | string, data: unknown): Promise<unknown> =>
    api.post(`/admin/contests/${contestId}/problems`, data),
  removeProblem: (contestId: number | string, problemId: number | string): Promise<unknown> =>
    api.delete(`/admin/contests/${contestId}/problems/${problemId}`),
  // Contest sections
  getContestSections: (contestId: number | string): Promise<unknown> =>
    api.get(`/admin/contests/${contestId}/sections`),
  // Contest Quiz management (section-aware)
  getContestQuiz: (contestId: number | string, sectionId?: number | string): Promise<unknown> => {
    const params = sectionId ? `?section_id=${sectionId}` : '';
    return api.get(`/admin/contests/${contestId}/quiz${params}`);
  },
  upsertContestQuiz: (contestId: number | string, data: unknown): Promise<unknown> =>
    api.post(`/admin/contests/${contestId}/quiz`, data),
  deleteContestQuiz: (contestId: number | string): Promise<unknown> =>
    api.delete(`/admin/contests/${contestId}/quiz`),
  // Practice mode - enable/disable practice for ended contests
  enablePracticeMode: (contestId: number | string, data: unknown): Promise<unknown> =>
    api.post(`/admin/contests/${contestId}/practice`, data),
};

// Quiz API - Faculty
export const quizAPI = {
  // Faculty offerings
  getMyOfferings: (): Promise<unknown> => api.get('/faculty/my-offerings'),
  getTheoryModules: (offeringId: number | string): Promise<unknown> =>
    api.get(`/faculty/offerings/${offeringId}/theory-modules`),

  // Quiz CRUD
  create: (data: unknown): Promise<unknown> => api.post('/faculty/quizzes', data),
  update: (quizId: number | string, data: unknown): Promise<unknown> =>
    api.put(`/faculty/quizzes/${quizId}`, data),
  delete: (quizId: number | string): Promise<unknown> => api.delete(`/faculty/quizzes/${quizId}`),
  get: (quizId: number | string): Promise<unknown> => api.get(`/faculty/quizzes/${quizId}`),
  list: (): Promise<unknown> => api.get('/faculty/quizzes'),
  listByOffering: (offeringId: number | string): Promise<unknown> =>
    api.get(`/faculty/quizzes?course_offering_id=${offeringId}`),

  // Quiz lifecycle
  publish: (quizId: number | string): Promise<unknown> =>
    api.post(`/faculty/quizzes/${quizId}/publish`),
  close: (quizId: number | string): Promise<unknown> =>
    api.post(`/faculty/quizzes/${quizId}/close`),

  // Questions management
  saveQuestions: (quizId: number | string, questions: unknown[]): Promise<unknown> =>
    api.post(`/faculty/quizzes/${quizId}/questions`, { questions }),
  editQuestion: (
    quizId: number | string,
    questionId: number | string,
    updates: unknown
  ): Promise<unknown> => api.put(`/faculty/quizzes/${quizId}/questions/${questionId}`, updates),
  deleteQuestion: (quizId: number | string, questionId: number | string): Promise<unknown> =>
    api.delete(`/faculty/quizzes/${quizId}/questions/${questionId}`),
  regenerateQuestions: (quizId: number | string, data: unknown): Promise<unknown> =>
    api.post(`/faculty/quizzes/${quizId}/regenerate`, data),

  // AI generation
  generateAI: (data: unknown): Promise<unknown> => api.post('/faculty/quizzes/generate', data),

  // Analytics
  getAnalytics: (quizId: number | string): Promise<unknown> =>
    api.get(`/faculty/quizzes/${quizId}/analytics`),
  getAttempts: (quizId: number | string): Promise<unknown> =>
    api.get(`/faculty/quizzes/${quizId}/attempts`),
};

// Quiz API - Student
export const studentQuizAPI = {
  listAvailable: (courseOfferingId?: number | string): Promise<unknown> =>
    api.get(`/quizzes${courseOfferingId ? `?course_offering_id=${courseOfferingId}` : ''}`),
  get: (quizId: number | string): Promise<unknown> => api.get(`/quizzes/${quizId}`),
  startAttempt: (quizId: number | string): Promise<unknown> => api.post(`/quizzes/${quizId}/start`),
  submitAttempt: (
    quizId: number | string,
    attemptId: number | string,
    answers: unknown
  ): Promise<unknown> => api.post(`/quizzes/${quizId}/submit`, { attempt_id: attemptId, answers }),
  getResult: (quizId: number | string): Promise<unknown> => api.get(`/quizzes/${quizId}/result`),
};

// HOD Quiz API
export const hodQuizAPI = {
  list: (): Promise<unknown> => api.get('/hod/quizzes'),
  getAnalytics: (quizId: number | string): Promise<unknown> =>
    api.get(`/hod/quizzes/${quizId}/analytics`),
};

// Lesson Plan API - Admin (AI-powered course content generation)
export const lessonPlanAPI = {
  // Upload lesson plan for processing
  upload: (file: File, courseId: number | string): Promise<unknown> => {
    const formData = new FormData();
    formData.append('lesson_plan', file);
    formData.append('course_id', String(courseId));
    return api.post('/admin/lesson-plans/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    } as ApiConfig);
  },

  // Get job status
  getJobStatus: (jobId: string): Promise<unknown> => api.get(`/admin/lesson-plans/jobs/${jobId}`),

  // Get preview of generated content
  getPreview: (jobId: string): Promise<unknown> =>
    api.get(`/admin/lesson-plans/jobs/${jobId}/preview`),

  // Approve and save generated content
  approveGeneration: (jobId: string, modifications: unknown): Promise<unknown> =>
    api.post(`/admin/lesson-plans/jobs/${jobId}/approve`, { modifications }),

  // Practice quiz management
  getPracticeQuiz: (weekId: number | string): Promise<unknown> =>
    api.get(`/admin/theory/weeks/${weekId}/practice-quiz`),
  updatePracticeQuiz: (weekId: number | string, data: unknown): Promise<unknown> =>
    api.put(`/admin/theory/weeks/${weekId}/practice-quiz`, data),
  regeneratePracticeQuiz: (weekId: number | string): Promise<unknown> =>
    api.post(`/admin/theory/weeks/${weekId}/practice-quiz/regenerate`),
};

// Theory PDF API - Admin
export const theoryAPI = {
  uploadPDF: (
    theoryId: number | string,
    file: File,
    moduleId?: number | string
  ): Promise<unknown> => {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('theory_id', String(theoryId));
    if (moduleId) {
      formData.append('theory_module_id', String(moduleId));
    }
    return api.post('/admin/theories/pdf/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    } as ApiConfig);
  },
  getPDFs: (theoryId: number | string): Promise<unknown> =>
    api.get(`/admin/theories/${theoryId}/pdfs`),
  deletePDF: (pdfId: number | string): Promise<unknown> =>
    api.delete(`/admin/theories/pdfs/${pdfId}`),
};

// Contest Generation API - Admin (AI-powered contest problem generation)
export const contestGenerationAPI = {
  // Start generating contest problems
  generate: (data: unknown): Promise<unknown> => api.post('/admin/contests/generate', data),

  // Get job status
  getJobStatus: (jobId: string): Promise<unknown> => api.get(`/admin/contests/generation/${jobId}`),

  // Get generated problems for review
  getGeneratedProblems: (jobId: string): Promise<unknown> =>
    api.get(`/admin/contests/generation/${jobId}/problems`),

  // Get full details of a specific generated problem
  getProblemDetail: (jobId: string, index: number): Promise<unknown> =>
    api.get(`/admin/contests/generation/${jobId}/problems/${index}`),

  // Approve and save selected problems
  // contestData can be: { approved_problem_indices, contest_id } OR { approved_problem_indices, create_contest, contest_title, ... }
  approveProblems: (jobId: string, contestData: unknown): Promise<unknown> =>
    api.post(`/admin/contests/generation/${jobId}/approve`, contestData),

  // Request regeneration of a specific problem
  regenerateProblem: (jobId: string, problemIndex: number, reason: string): Promise<unknown> =>
    api.post(`/admin/contests/generation/${jobId}/regenerate`, {
      problem_index: problemIndex,
      reason,
    }),

  // Generate editorial hints for a problem (post-contest)
  generateEditorial: (data: unknown): Promise<unknown> =>
    api.post('/admin/contests/editorial/generate', data),

  // Get editorial hints for a problem
  getEditorial: (contestId: number | string, problemId: number | string): Promise<unknown> =>
    api.get(`/admin/contests/${contestId}/problems/${problemId}/editorial`),
};

// Practice API - For students to practice global problems
export const practiceAPI = {
  // Get all practice problems (global problems)
  getProblems: (): Promise<unknown> => api.get('/practice/problems'),

  // Get a single practice problem
  getProblem: (id: number | string): Promise<unknown> => api.get(`/practice/problems/${id}`),

  // Run code against sample test cases
  run: (
    problemId: number | string,
    languageId: number | string,
    sourceCode: string
  ): Promise<unknown> =>
    api.post('/practice/run', {
      problem_id: problemId,
      language_id: languageId,
      source_code: sourceCode,
    }),

  // Submit solution for practice problem
  submit: (
    problemId: number | string,
    languageId: number | string,
    sourceCode: string,
    timeTaken: number
  ): Promise<unknown> =>
    api.post('/practice/submit', {
      problem_id: problemId,
      language_id: languageId,
      source_code: sourceCode,
      time_taken: timeTaken,
    }),

  // Get submissions for a practice problem
  getSubmissions: (problemId: number | string, page = 1, limit = 20): Promise<unknown> =>
    api.get(`/practice/problems/${problemId}/submissions?page=${page}&limit=${limit}`),
};

// Super Admin Practice Problems API
export const superAdminPracticeAPI = {
  // Get all global problems
  getProblems: (): Promise<unknown> => api.get('/super-admin/problems'),

  // Create a global problem
  create: (data: unknown): Promise<unknown> => api.post('/super-admin/problems', data),

  // Update a global problem
  update: (id: number | string, data: unknown): Promise<unknown> =>
    api.put(`/super-admin/problems/${id}`, data),

  // Delete a global problem
  delete: (id: number | string): Promise<unknown> => api.delete(`/super-admin/problems/${id}`),

  // Add test case to a global problem
  createTestCase: (problemId: number | string, data: unknown): Promise<unknown> =>
    api.post(`/super-admin/problems/${problemId}/testcases`, data),
};

// Super Admin Subject API
export const superAdminSubjectAPI = {
  getAll: (includeTopics = false): Promise<unknown> => {
    const params = includeTopics ? '?include_topics=true' : '';
    return api.get(`/subjects${params}`);
  },
  create: (data: unknown): Promise<unknown> => api.post('/super-admin/subjects', data),
  update: (id: number | string, data: unknown): Promise<unknown> =>
    api.put(`/super-admin/subjects/${id}`, data),
  delete: (id: number | string): Promise<unknown> => api.delete(`/super-admin/subjects/${id}`),
};

// Super Admin Topic API
export const superAdminTopicAPI = {
  getAll: (): Promise<unknown> => api.get('/topics'),
  create: (data: unknown): Promise<unknown> => api.post('/super-admin/topics', data),
  update: (id: number | string, data: unknown): Promise<unknown> =>
    api.put(`/super-admin/topics/${id}`, data),
  delete: (id: number | string): Promise<unknown> => api.delete(`/super-admin/topics/${id}`),
};

// Job Queue API - For polling execution status
export const jobAPI = {
  // Get job status
  getStatus: (jobId: string): Promise<unknown> => api.get(`/jobs/${jobId}`),

  // Poll until completion (max 60 seconds)
  // onProgress callback receives status updates during polling
  pollUntilComplete: async (
    jobId: string,
    onProgress?: (status: unknown) => void,
    timeout = 60000
  ): Promise<unknown> => {
    const startTime = Date.now();
    const pollInterval = 2000; // Poll every 2 seconds

    while (Date.now() - startTime < timeout) {
      try {
        const status = await api.get(`/jobs/${jobId}`);

        // Call progress callback if provided
        if (onProgress && status) {
          onProgress(status);
        }

        // Check if job is complete
        const statusObj = status as unknown as { status?: string };
        if (statusObj.status === 'completed' || statusObj.status === 'failed') {
          return status;
        }

        // Wait before next poll
        await new Promise((resolve) => setTimeout(resolve, pollInterval));
      } catch (error) {
        // If error is 404, job might not exist yet - continue polling
        if ((error as ApiError).status === 404) {
          await new Promise((resolve) => setTimeout(resolve, pollInterval));
          continue;
        }
        // Other errors - throw
        throw error;
      }
    }

    // Timeout exceeded
    throw new Error('Job polling timeout - execution took too long');
  },
};

export default api;
