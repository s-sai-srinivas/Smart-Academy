import { useState, useEffect, useRef, useCallback } from 'react';
import { useParams, useNavigate, useLocation } from 'react-router-dom';
import { problemsAPI, submissionsAPI } from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import { useNotification } from '../../context/NotificationContext';
import { showError } from '../../utils/showAlert';
import { SubmissionProvider, useSubmissions } from '../../context/SubmissionContext';
import Breadcrumb from '../../components/common/Breadcrumb';
import CodeEditor from '../../components/editor/CodeEditor';
import { DEFAULT_CODE } from '../../components/editor/codeEditorConstants';
import LanguageSelector from '../../components/editor/LanguageSelector';
import TestCasePanel from '../../components/editor/TestCasePanel';
import ResultsPanel, { type ExecutionResult } from '../../components/editor/ResultsPanel';
import SubmissionsPanel, {
  type PaginationInfo,
  type Submission,
} from '../../components/editor/SubmissionsPanel';
import useResizablePanels from '../../hooks/useResizablePanels';
import useJobPolling from '../../hooks/useJobPolling';

interface Problem {
  id: number;
  title: string;
  description: string;
  input_format?: string;
  output_format?: string;
  constraints?: string;
  difficulty?: string;
  tags?: string;
  time_limit?: number;
  memory_limit?: number;
  test_cases?: TestCase[];
}

interface TestCase {
  input: string;
  expected_output: string;
  is_sample?: boolean;
}

interface SavedCodeData {
  codeByLang?: Record<string, string>;
  languageId?: number;
  lastSaved?: string;
  timer?: number;
  timerFrozen?: boolean;
  lastSubmittedAt?: string;
}

interface SubmissionResponse {
  completed?: boolean;
  completed_at?: string;
  time_taken_seconds?: number;
}

interface SubmissionsPanelWithContextProps {
  refreshKey: number;
}

// Inner component that consumes SubmissionContext to avoid prop drilling
function SubmissionsPanelWithContext({ refreshKey }: SubmissionsPanelWithContextProps) {
  const { fetchSubmissions, fetchCode } = useSubmissions();
  return (
    <SubmissionsPanel
      fetchSubmissions={
        fetchSubmissions as unknown as (page: number) => Promise<{
          submissions?: Submission[];
          pagination?: PaginationInfo;
        }>
      }
      fetchCode={fetchCode}
      refreshKey={refreshKey}
    />
  );
}

function ProblemSolvingPage() {
  const { topicId, problemId } = useParams<{ topicId: string; problemId: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const { user, isAdmin } = useAuth();
  const notification = useNotification();
  const courseId = (location.state as { courseId?: string } | null)?.courseId;

  const [problem, setProblem] = useState<Problem | null>(null);
  const [testCases, setTestCases] = useState<TestCase[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [isCompleted, setIsCompleted] = useState<boolean>(false);
  const [completedAt, setCompletedAt] = useState<Date | null>(null);
  const [showCompletionCelebration, setShowCompletionCelebration] = useState<boolean>(false);

  // Solve popup and timer state
  const [showSolvePopup, setShowSolvePopup] = useState<boolean>(true);
  const [isSolving, setIsSolving] = useState<boolean>(false);
  const [timer, setTimer] = useState<number>(0);
  const [isTimerFrozen, setIsTimerFrozen] = useState<boolean>(false);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const [languageId, setLanguageId] = useState<number>(71);
  const [code, setCode] = useState<string>(DEFAULT_CODE[71]);
  const [codeByLang, setCodeByLang] = useState<Record<string, string>>(() => {
    const initial: Record<string, string> = {};
    Object.keys(DEFAULT_CODE).forEach((id) => {
      initial[id] = DEFAULT_CODE[Number(id)];
    });
    return initial;
  });
  const [activeTab, setActiveTab] = useState<string>('description');
  const [consoleTab, setConsoleTab] = useState<string>('testcase');
  const [results, setResults] = useState<unknown>(null);
  const [running, setRunning] = useState<boolean>(false);
  const [submissionsRefreshKey, setSubmissionsRefreshKey] = useState<number>(0);

  // Job polling hook for async submissions
  const { pollJob, isPolling, pollProgress } = useJobPolling();

  const {
    splitRef,
    editorContainerRef,
    leftWidthPercent,
    consoleHeight,
    isHorizontalDragging,
    isVerticalDragging,
    onHorizontalPointerDown,
    onVerticalPointerDown,
  } = useResizablePanels({
    storageKey: `lab-split-${topicId}-${problemId}`,
    initialLeftPercent: 42,
    minLeftPx: 280,
    maxLeftPercent: 60,
    minRightPx: 360,
    initialConsoleHeight: 220,
    minConsoleHeight: 120,
    maxConsolePercent: 40,
    minEditorHeight: 220,
  });

  // Submissions data is now provided via SubmissionContext

  // Load saved code from localStorage and completion status
  const getStorageKey = useCallback((): string => {
    return `problem_code_${problemId ?? ''}_${user?.regdno || 'anonymous'}`;
  }, [problemId, user?.regdno]);

  const saveCodeToStorage = useCallback(
    (overrides: Partial<SavedCodeData> = {}) => {
      try {
        const key = getStorageKey();
        const data: SavedCodeData = {
          codeByLang,
          languageId,
          lastSaved: overrides.lastSaved || new Date().toISOString(),
          timer: overrides.timer ?? timer,
          timerFrozen: overrides.timerFrozen ?? isTimerFrozen,
          lastSubmittedAt: overrides.lastSubmittedAt,
        };
        localStorage.setItem(key, JSON.stringify(data));
      } catch (e) {
        console.warn('Could not save code to localStorage:', e);
      }
    },
    [getStorageKey, codeByLang, languageId, timer, isTimerFrozen]
  );

  const loadSavedCode = useCallback(() => {
    try {
      const key = getStorageKey();
      const saved = localStorage.getItem(key);
      if (saved) {
        const data = JSON.parse(saved) as SavedCodeData;

        // Merge saved code with default codes for any missing languages
        const mergedCodeByLang: Record<string, string> = { ...DEFAULT_CODE } as Record<
          string,
          string
        >;
        if (data.codeByLang) {
          Object.keys(data.codeByLang).forEach((langId) => {
            mergedCodeByLang[langId] = data.codeByLang?.[langId] ?? '';
          });
        }
        setCodeByLang(mergedCodeByLang);

        const savedLanguageId = data.languageId || languageId;
        setLanguageId(savedLanguageId);
        setCode(mergedCodeByLang[String(savedLanguageId)] || DEFAULT_CODE[savedLanguageId] || '');
        setTimer(data.timer || 0);
        setIsTimerFrozen(!!data.timerFrozen);

        // Check if was previously solving
        if (data.lastSaved) {
          const savedTime = new Date(data.lastSaved);
          const hoursSinceSave = (Date.now() - savedTime.getTime()) / (1000 * 60 * 60);
          if (hoursSinceSave < 24) {
            setShowSolvePopup(false);
            setIsSolving(true);
          }
        }
      }
    } catch (e) {
      console.warn('Could not load saved code:', e);
    }
  }, [getStorageKey, languageId]);

  const clearSavedCode = useCallback(() => {
    try {
      localStorage.removeItem(getStorageKey());
    } catch (e) {
      console.warn('Could not clear saved code:', e);
    }
  }, [getStorageKey]);

  const loadProblem = useCallback(async () => {
    if (!problemId) return;
    try {
      const data = (await problemsAPI.getById(problemId)) as { problem?: Problem } & Problem;
      const problemData = data.problem || data;
      setProblem(problemData);

      if (problemData.test_cases) {
        setTestCases(problemData.test_cases);
      } else {
        try {
          const tcData = (await problemsAPI.getTestCases(problemId)) as
            | { test_cases?: TestCase[] }
            | TestCase[];
          if (Array.isArray(tcData)) {
            setTestCases(tcData);
          } else {
            setTestCases(tcData?.test_cases || []);
          }
        } catch {
          setTestCases([]);
        }
      }
    } catch (err) {
      showError((err as Error).message || 'Failed to load problem');
    } finally {
      setLoading(false);
    }
  }, [problemId]);

  const loadCompletionStatus = useCallback(async () => {
    if (!user || !problemId) return;
    try {
      const data = (await submissionsAPI.checkCompletion(problemId)) as SubmissionResponse;
      setIsCompleted(data.completed || false);
      if (data.completed_at) {
        setCompletedAt(new Date(data.completed_at));
      }
      if (data.completed && data.time_taken_seconds !== undefined && data.time_taken_seconds >= 0) {
        setTimer(Math.floor(data.time_taken_seconds));
        setIsTimerFrozen(true);
      }
    } catch (err) {
      console.debug('Could not load completion status:', err);
    }
  }, [user, problemId]);

  useEffect(() => {
    loadProblem();
    loadCompletionStatus();
    loadSavedCode();

    return () => {
      // Cleanup timer on unmount
      if (timerRef.current) {
        clearInterval(timerRef.current);
      }
    };
  }, [loadProblem, loadCompletionStatus, loadSavedCode]);

  // Skip popup if already completed
  useEffect(() => {
    if (isCompleted) {
      setShowSolvePopup(false);
    }
  }, [isCompleted]);

  // Save code to localStorage whenever it changes
  useEffect(() => {
    if (isSolving && code) {
      saveCodeToStorage();
    }
  }, [code, languageId, isSolving, saveCodeToStorage]);

  // Timer effect
  useEffect(() => {
    if (isSolving && !isTimerFrozen) {
      timerRef.current = setInterval(() => {
        setTimer((prev) => prev + 1);
      }, 1000);
    } else if (timerRef.current) {
      clearInterval(timerRef.current);
    }

    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current);
      }
    };
  }, [isSolving, isTimerFrozen]);

  const formatTime = (seconds: number): string => {
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;
    if (hrs > 0) {
      return `${hrs}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    }
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const handleStartSolving = () => {
    setShowSolvePopup(false);
    setIsSolving(true);
    setIsTimerFrozen(false);
    setTimer(0);
  };

  const handleLanguageChange = (newLangId: number) => {
    // Get the new language's code BEFORE state updates (from current codeByLang or default)
    const newCode = codeByLang[newLangId] || DEFAULT_CODE[newLangId] || '';

    // Save current code for current language
    setCodeByLang((prev) => ({ ...prev, [languageId]: code }));
    setLanguageId(newLangId);
    setCode(newCode);
  };

  const handleRun = async () => {
    setRunning(true);
    setConsoleTab('result');
    setResults(null);
    setShowCompletionCelebration(false);
    if (!problemId) {
      setRunning(false);
      notification?.error('Problem ID is missing');
      return;
    }
    try {
      const response = (await submissionsAPI.run(parseInt(problemId, 10), languageId, code)) as {
        job_id?: string;
      } & Record<string, unknown>;

      // Check if response contains job_id (async job queue)
      if (response.job_id) {
        const jobResult = await pollJob(response.job_id, {
          onProgress: (status) =>
            setResults({
              status: (status as { status?: string }).status,
              message: (status as { message?: string }).message,
            }),
          timeout: 60000,
        });

        if (jobResult) {
          setResults({
            all_passed: (jobResult as { all_passed?: boolean }).all_passed,
            passed: (jobResult as { all_passed?: boolean }).all_passed,
            test_results: (jobResult as { test_results?: unknown }).test_results,
            ...(jobResult as Record<string, unknown>),
          });
        }
      } else {
        // Direct response (no job queue)
        setResults(response);
      }
    } catch (err) {
      setResults({ error_message: (err as Error).message, passed: false });
    } finally {
      setRunning(false);
    }
  };

  const handleSubmit = async () => {
    if (!user) {
      notification.error('Please login to submit your solution');
      return;
    }

    const frozenTime = timer;
    setIsTimerFrozen(true);
    saveCodeToStorage({
      timer: frozenTime,
      timerFrozen: true,
      lastSubmittedAt: new Date().toISOString(),
      lastSaved: new Date().toISOString(),
    });

    setRunning(true);
    setConsoleTab('result');
    setResults(null);
    setShowCompletionCelebration(false);

    try {
      if (!problemId) {
        setRunning(false);
        notification.error('Problem ID is missing');
        return;
      }

      const response = (await submissionsAPI.submit(
        parseInt(problemId, 10),
        languageId,
        code,
        frozenTime
      )) as { job_id?: string } & Record<string, unknown>;

      // Check if response contains job_id (async job queue)
      if (response.job_id) {
        const jobResult = await pollJob(response.job_id, {
          onProgress: (status) =>
            setResults({
              status: (status as { status?: string }).status,
              message: (status as { message?: string }).message,
            }),
          timeout: 90000,
        });

        if (jobResult) {
          setResults({
            all_passed: (jobResult as { all_passed?: boolean }).all_passed,
            passed: (jobResult as { all_passed?: boolean }).all_passed,
            test_results: (jobResult as { test_results?: unknown }).test_results,
            ...(jobResult as Record<string, unknown>),
          });
          setSubmissionsRefreshKey((k) => k + 1);

          // Check if all tests passed
          const allPassed = (jobResult as { all_passed?: boolean }).all_passed ?? false;
          if (allPassed && !isCompleted) {
            setShowCompletionCelebration(true);
            setIsCompleted(true);
            setCompletedAt(new Date());
            setIsSolving(false);
            if (timerRef.current) {
              clearInterval(timerRef.current);
            }
            // Clear saved code on successful completion
            clearSavedCode();
          }
        }
      } else {
        // Direct response (no job queue)
        setResults(response);
        setSubmissionsRefreshKey((k) => k + 1);

        // Check if all tests passed
        const allPassed =
          (response as { all_passed?: boolean }).all_passed ??
          (response as { passed?: boolean }).passed ??
          false;
        if (allPassed && !isCompleted) {
          setShowCompletionCelebration(true);
          setIsCompleted(true);
          setCompletedAt(new Date());
          setIsSolving(false);
          if (timerRef.current) {
            clearInterval(timerRef.current);
          }
          // Clear saved code on successful completion
          clearSavedCode();
        }
      }
    } catch (err) {
      setResults({ error_message: (err as Error).message, passed: false });
    } finally {
      setRunning(false);
    }
  };

  const handleDelete = async () => {
    if (!confirm('Are you sure you want to delete this problem?')) return;
    if (!problemId) {
      notification.error('Problem ID is missing');
      return;
    }

    try {
      await problemsAPI.delete(problemId);
      clearSavedCode();
      navigate(`/courses/${courseId}/lab/${topicId}`);
    } catch (err) {
      notification.error('Failed to delete problem: ' + (err as Error).message);
    }
  };

  const getDifficultyClass = (difficulty: string): string => {
    switch (difficulty) {
      case 'easy':
        return 'difficulty-easy';
      case 'medium':
        return 'difficulty-medium';
      case 'hard':
        return 'difficulty-hard';
      default:
        return '';
    }
  };

  if (loading) {
    return (
      <div className="text-muted" style={{ padding: '2rem' }}>
        Loading problem...
      </div>
    );
  }

  if (!problem) {
    return <div style={{ padding: '2rem', color: 'var(--error)' }}>Problem not found</div>;
  }

  return (
    <SubmissionProvider problemId={problemId ?? ''}>
      <div className="problem-view">
        {/* Solve Popup */}
        {showSolvePopup && !isCompleted && (
          <div
            style={{
              position: 'fixed',
              top: 0,
              left: 0,
              right: 0,
              bottom: 0,
              background: 'rgba(0, 0, 0, 0.85)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              zIndex: 1001,
            }}
          >
            <div
              style={{
                background: 'var(--bg-primary, #0f0f1a)',
                borderRadius: '16px',
                padding: '2.5rem',
                textAlign: 'center',
                maxWidth: '500px',
                border: '1px solid var(--border)',
                boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.5)',
              }}
            >
              <div
                style={{
                  width: '80px',
                  height: '80px',
                  margin: '0 auto 1.5rem',
                  borderRadius: '50%',
                  background: 'linear-gradient(135deg, var(--sky-500) 0%, var(--sky-600) 100%)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <svg
                  width="40"
                  height="40"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="#fff"
                  strokeWidth="2"
                >
                  <polyline points="9 11 12 14 22 4"></polyline>
                  <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"></path>
                </svg>
              </div>
              <h2 style={{ margin: '0 0 0.5rem 0', fontSize: '1.5rem', fontWeight: 600 }}>
                Ready to solve?
              </h2>
              <p style={{ margin: '0 0 2rem 0', color: 'var(--text-secondary)' }}>
                Click below to start the timer and access the code editor
              </p>
              <button
                onClick={handleStartSolving}
                style={{
                  background: 'linear-gradient(135deg, var(--sky-500) 0%, var(--sky-600) 100%)',
                  color: '#fff',
                  border: 'none',
                  padding: '1rem 3rem',
                  borderRadius: '8px',
                  fontSize: '1.1rem',
                  fontWeight: 600,
                  cursor: 'pointer',
                  transition: 'transform 0.2s',
                }}
                onMouseEnter={(e: React.MouseEvent<HTMLButtonElement>) => {
                  (e.target as HTMLButtonElement).style.transform = 'scale(1.05)';
                }}
                onMouseLeave={(e: React.MouseEvent<HTMLButtonElement>) => {
                  (e.target as HTMLButtonElement).style.transform = 'scale(1)';
                }}
              >
                Start Solving
              </button>
              <button
                onClick={() => navigate(-1)}
                style={{
                  background: 'transparent',
                  color: 'var(--text-secondary)',
                  border: 'none',
                  padding: '0.75rem',
                  cursor: 'pointer',
                  fontSize: '0.9rem',
                  marginTop: '1rem',
                }}
              >
                Go Back
              </button>
            </div>
          </div>
        )}

        {/* Completion Celebration Banner */}
        {showCompletionCelebration && (
          <div
            style={{
              position: 'fixed',
              top: 0,
              left: 0,
              right: 0,
              background: 'linear-gradient(135deg, var(--emerald-500) 0%, #059669 100%)',
              color: '#fff',
              padding: '1rem',
              zIndex: 1000,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '1rem',
              animation: 'slideDown 0.5s ease-out',
              boxShadow: '0 4px 20px rgba(16, 185, 129, 0.4)',
            }}
          >
            <svg
              width="32"
              height="32"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="3"
            >
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
              <polyline points="22 4 12 14.01 9 11.01"></polyline>
            </svg>
            <div>
              <div style={{ fontSize: '1.1rem', fontWeight: 700 }}>🎉 Congratulations!</div>
              <div style={{ fontSize: '0.9rem', opacity: 0.9 }}>
                Solved in {formatTime(timer)} • You've mastered this problem!
              </div>
            </div>
            <button
              onClick={() => setShowCompletionCelebration(false)}
              style={{
                background: 'rgba(255,255,255,0.2)',
                border: 'none',
                color: '#fff',
                padding: '0.5rem 1rem',
                borderRadius: '4px',
                cursor: 'pointer',
                marginLeft: '1rem',
              }}
            >
              Dismiss
            </button>
          </div>
        )}

        {/* Breadcrumb Navigation */}
        <Breadcrumb
          items={[
            { label: 'Dashboard', to: '/dashboard' },
            { label: 'Courses', to: '/courses' },
            ...(courseId ? [{ label: 'Course', to: `/courses/${courseId}` }] : []),
            {
              label: 'Lab Session',
              to: courseId ? `/courses/${courseId}/lab/${topicId}` : undefined,
            },
            { label: problem.title },
          ]}
        />

        <div className="split-container" ref={splitRef}>
          {/* Left Panel: Problem Description */}
          <div className="left-panel" style={{ width: `${leftWidthPercent}%` }}>
            <div className="panel-tabs">
              <button
                className={`tab-btn ${activeTab === 'description' ? 'active' : ''}`}
                onClick={() => setActiveTab('description')}
              >
                Description
              </button>
              <button
                className={`tab-btn ${activeTab === 'editorial' ? 'active' : ''}`}
                onClick={() => setActiveTab('editorial')}
              >
                Editorial
              </button>
              <button
                className={`tab-btn ${activeTab === 'submissions' ? 'active' : ''}`}
                onClick={() => setActiveTab('submissions')}
              >
                Submissions
              </button>
            </div>

            <div className="tab-content">
              {activeTab === 'description' && (
                <div className="tab-pane active">
                  {isAdmin() && (
                    <div
                      className="admin-actions"
                      style={{ display: 'flex', gap: '0.5rem', marginBottom: '1rem' }}
                    >
                      <button className="btn btn-danger" onClick={handleDelete}>
                        Delete Problem
                      </button>
                    </div>
                  )}
                  <div className="problem-detail">
                    {/* Title with Solved Badge */}
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: '0.75rem',
                        marginBottom: '0.5rem',
                        flexWrap: 'wrap',
                      }}
                    >
                      <h2 style={{ margin: 0 }}>{problem.title}</h2>
                      {isCompleted && (
                        <span
                          style={{
                            display: 'inline-flex',
                            alignItems: 'center',
                            gap: '0.35rem',
                            padding: '0.35rem 0.75rem',
                            background: 'var(--success, #10b981)',
                            color: '#fff',
                            borderRadius: '20px',
                            fontSize: '0.8rem',
                            fontWeight: 600,
                          }}
                        >
                          <svg
                            width="16"
                            height="16"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="3"
                          >
                            <polyline points="20 6 9 17 4 12"></polyline>
                          </svg>
                          Solved
                          {completedAt && (
                            <span style={{ opacity: 0.8, fontWeight: 400, marginLeft: '0.25rem' }}>
                              {completedAt.toLocaleDateString()}
                            </span>
                          )}
                        </span>
                      )}
                    </div>
                    <div style={{ marginBottom: '1rem' }}>
                      <span
                        className={`difficulty ${getDifficultyClass(problem.difficulty || '')}`}
                      >
                        {problem.difficulty}
                      </span>
                      {problem.tags &&
                        problem.tags.split(',').map((tag, i) => (
                          <span key={i} className="tag" style={{ marginLeft: '0.5rem' }}>
                            {tag.trim()}
                          </span>
                        ))}
                    </div>
                    <div style={{ whiteSpace: 'pre-wrap' }}>{problem.description}</div>

                    {/* Sample Test Cases on left panel */}
                    {testCases.filter((tc) => tc.is_sample).length > 0 && (
                      <div style={{ marginTop: '1.5rem' }}>
                        <h3
                          style={{
                            fontSize: '1rem',
                            fontWeight: 600,
                            marginBottom: '0.75rem',
                            color: 'var(--text-primary)',
                          }}
                        >
                          Examples
                        </h3>
                        {testCases
                          .filter((tc) => tc.is_sample)
                          .map((tc, i) => (
                            <div
                              key={i}
                              style={{
                                marginBottom: '1rem',
                                padding: '0.75rem',
                                background: 'var(--surface)',
                                borderRadius: '6px',
                                border: '1px solid var(--border)',
                              }}
                            >
                              <div
                                style={{
                                  fontSize: '0.85rem',
                                  fontWeight: 600,
                                  marginBottom: '0.5rem',
                                  color: 'var(--text-secondary)',
                                }}
                              >
                                Example {i + 1}
                              </div>
                              <div style={{ marginBottom: '0.5rem' }}>
                                <div
                                  style={{
                                    fontSize: '0.8rem',
                                    color: 'var(--text-muted)',
                                    marginBottom: '0.25rem',
                                  }}
                                >
                                  Input:
                                </div>
                                <pre
                                  style={{
                                    margin: 0,
                                    padding: '0.5rem 0.75rem',
                                    background: 'var(--bg-secondary, #1a1a2e)',
                                    borderRadius: '4px',
                                    fontSize: '0.85rem',
                                    whiteSpace: 'pre-wrap',
                                  }}
                                >
                                  {tc.input}
                                </pre>
                              </div>
                              <div>
                                <div
                                  style={{
                                    fontSize: '0.8rem',
                                    color: 'var(--text-muted)',
                                    marginBottom: '0.25rem',
                                  }}
                                >
                                  Output:
                                </div>
                                <pre
                                  style={{
                                    margin: 0,
                                    padding: '0.5rem 0.75rem',
                                    background: 'var(--bg-secondary, #1a1a2e)',
                                    borderRadius: '4px',
                                    fontSize: '0.85rem',
                                    whiteSpace: 'pre-wrap',
                                  }}
                                >
                                  {tc.expected_output}
                                </pre>
                              </div>
                            </div>
                          ))}
                      </div>
                    )}

                    <div style={{ marginTop: '1.5rem', color: 'var(--text-muted)' }}>
                      <div>Time Limit: {problem.time_limit || 2000}ms</div>
                      <div>Memory Limit: {(problem.memory_limit || 256000) / 1024}MB</div>
                    </div>
                  </div>
                </div>
              )}

              {activeTab === 'editorial' && (
                <div className="tab-pane active">
                  <p className="text-muted">Editorial coming soon...</p>
                </div>
              )}

              {activeTab === 'submissions' && (
                <div className="tab-pane active" style={{ height: '100%' }}>
                  {!user ? (
                    <div
                      style={{ padding: '3rem', textAlign: 'center', color: 'var(--text-muted)' }}
                    >
                      Please login to view your submissions
                    </div>
                  ) : (
                    <SubmissionsPanelWithContext refreshKey={submissionsRefreshKey} />
                  )}
                </div>
              )}
            </div>
          </div>

          {/* Resize Handle */}
          <div
            className={`resize-handle ${isHorizontalDragging ? 'active' : ''}`}
            role="separator"
            aria-orientation="vertical"
            aria-valuenow={Math.round(leftWidthPercent)}
            aria-valuemin={20}
            aria-valuemax={80}
            onPointerDown={onHorizontalPointerDown}
          ></div>

          {/* Right Panel: Code Editor */}
          <div className="right-panel">
            <div className="editor-container" ref={editorContainerRef}>
              <div className="editor-header">
                <LanguageSelector value={languageId} onChange={handleLanguageChange} />
                <div className="editor-actions">
                  <button
                    className="btn btn-secondary"
                    onClick={handleRun}
                    disabled={running || (!isSolving && !isCompleted)}
                  >
                    <svg
                      width="16"
                      height="16"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                    >
                      <polygon points="5 3 19 12 5 21 5 3"></polygon>
                    </svg>
                    Run
                  </button>
                  <button
                    className="btn btn-success"
                    onClick={handleSubmit}
                    disabled={running || (!isSolving && !isCompleted)}
                  >
                    <svg
                      width="16"
                      height="16"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                    >
                      <polyline points="20 6 9 17 4 12"></polyline>
                    </svg>
                    Submit
                  </button>
                  {(isSolving || isCompleted) && (
                    <div
                      className="solve-timer-pill"
                      title={isTimerFrozen ? 'Timer frozen at submission' : 'Time spent solving'}
                    >
                      <svg
                        width="14"
                        height="14"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                      >
                        <circle cx="12" cy="12" r="10"></circle>
                        <polyline points="12 6 12 12 16 14"></polyline>
                      </svg>
                      {formatTime(timer)}
                    </div>
                  )}
                </div>
              </div>

              <CodeEditor languageId={languageId} code={code} onChange={setCode} />

              <div
                className={`console-resize-handle ${isVerticalDragging ? 'active' : ''}`}
                role="separator"
                aria-orientation="horizontal"
                onPointerDown={onVerticalPointerDown}
              ></div>

              <div className="console-panel" style={{ height: consoleHeight }}>
                <div className="console-tabs">
                  <button
                    className={`console-tab-btn ${consoleTab === 'testcase' ? 'active' : ''}`}
                    onClick={() => setConsoleTab('testcase')}
                  >
                    Testcase
                  </button>
                  <button
                    className={`console-tab-btn ${consoleTab === 'result' ? 'active' : ''}`}
                    onClick={() => setConsoleTab('result')}
                  >
                    Test Result
                  </button>
                </div>

                <div className="console-content">
                  {consoleTab === 'testcase' ? (
                    <TestCasePanel testCases={testCases} />
                  ) : (
                    <ResultsPanel
                      results={results as ExecutionResult}
                      loading={running || isPolling}
                      pollProgress={
                        pollProgress as {
                          message?: string;
                          status?: string;
                          retry_count?: number;
                        } | null
                      }
                    />
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>

        <style>{`
                @keyframes slideDown {
                    from {
                        transform: translateY(-100%);
                        opacity: 0;
                    }
                    to {
                        transform: translateY(0);
                        opacity: 1;
                    }
                }
            `}</style>
      </div>
    </SubmissionProvider>
  );
}

export default ProblemSolvingPage;
