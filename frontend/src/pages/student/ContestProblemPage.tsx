import { useState, useEffect, useCallback } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { contestsAPI, submissionsAPI } from '../../services/api';
import { showAlert, showError } from '../../utils/showAlert';
import CodeEditor from '../../components/editor/CodeEditor';
import { DEFAULT_CODE } from '../../components/editor/codeEditorConstants';
import useResizablePanels from '../../hooks/useResizablePanels';
import LanguageSelector from '../../components/editor/LanguageSelector';
import TestCasePanel from '../../components/editor/TestCasePanel';
import ResultsPanel, { type ExecutionResult } from '../../components/editor/ResultsPanel';
import SolveTimer from '../../components/editor/SolveTimer';
import SubmissionsPanel from '../../components/editor/SubmissionsPanel';
import Breadcrumb from '../../components/common/Breadcrumb';
import useContestFullscreen from '../../hooks/useContestFullscreen';
import { useContestMode } from '../../context/ContestModeContext';
import useJobPolling from '../../hooks/useJobPolling';
import type { PaginationInfo } from '../../components/editor/SubmissionsPanel';

interface Contest {
  id: number;
  title: string;
  start_time: string;
  end_time: string;
  is_practice_active?: boolean;
  has_joined?: boolean;
  has_finished?: boolean;
  problems?: ContestProblem[];
}

interface ContestProblem {
  problem_id: number;
  title: string;
  is_solved?: boolean;
  is_attempted?: boolean;
  points?: number;
  difficulty?: string;
}

interface Problem {
  problem_id: number;
  title: string;
  description?: string;
  input_format?: string;
  output_format?: string;
  constraints?: string;
  sample_test_cases?: TestCase[];
  difficulty?: string;
  points_in_contest?: number;
  time_limit?: number;
  memory_limit?: number;
  tags?: string;
  is_solved?: boolean;
}

interface TestCase {
  input: string;
  expected_output: string;
  is_sample?: boolean;
}

interface Submission {
  id: number;
  language_id: number;
  passed: boolean;
  submitted_at: string;
  source_code?: string | null;
  execution_time?: number;
  memory_used?: number;
  total_tests?: number;
  passed_tests?: number;
}

interface EscViolationResponse {
  esc_violations?: number;
}

function ContestProblemPage() {
  const { id, problemId } = useParams<{ id: string; problemId: string }>();
  const navigate = useNavigate();
  const contestId = id ?? '';
  const contestProblemId = problemId ?? '';

  // Contest mode context
  const {
    isInContestMode: _isInContestMode,
    activeContestId: _activeContestId,
    contestTitle: _contestTitle,
    setContestMode,
    finishContest: _finishContestFromContext,
    isContestEnded: _isContestEnded,
  } = useContestMode();

  // Contest and problem data
  const [contest, setContest] = useState<Contest | null>(null);
  const [problem, setProblem] = useState<Problem | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [isSolved, setIsSolved] = useState<boolean>(false);
  const [isEscBlocked, setIsEscBlocked] = useState<boolean>(false);
  const isPracticeMode = Boolean(
    contest?.is_practice_active && contest?.end_time && new Date() > new Date(contest.end_time)
  );

  // Code editor state
  // Helper: build a stable localStorage key per contest-problem-language
  const storageKey = (langId: number): string => `code_${contestId}_${contestProblemId}_${langId}`;

  const [languageId, setLanguageId] = useState<number>(71);
  const [code, setCode] = useState<string>(
    () => localStorage.getItem(storageKey(71)) ?? DEFAULT_CODE[71]
  );
  // codeByLang is scoped to the current problemId — built fresh on every problem mount
  const [codeByLang, setCodeByLang] = useState<Record<string, string>>(() => {
    const initial: Record<string, string> = {};
    Object.keys(DEFAULT_CODE).forEach((langId) => {
      initial[langId] =
        localStorage.getItem(storageKey(Number(langId))) ?? DEFAULT_CODE[Number(langId)];
    });
    return initial;
  });

  // Submission state
  const [submitting, setSubmitting] = useState<boolean>(false);
  const [running, setRunning] = useState<boolean>(false);
  const [results, setResults] = useState<unknown>(null);

  // Tab state
  const [activeTab, setActiveTab] = useState<string>('description');
  const [consoleTab, setConsoleTab] = useState<string>('testcase');
  const [submissionsRefreshKey, setSubmissionsRefreshKey] = useState<number>(0);

  // Problem navigation sidebar state
  const [showProblemNav, setShowProblemNav] = useState<boolean>(false);

  // Job polling hook for queue-based submissions
  const { pollJob, isPolling, pollProgress } = useJobPolling();

  // Full-screen enforcement for contest integrity - tracks violations for plagiarism records, does NOT disqualify
  const {
    isFullscreen,
    showInitialPrompt,
    showWarning,
    violations,
    maxViolations: _maxViolations,
    requestFullscreen,
  } = useContestFullscreen({
    warningDuration: 60000, // 1 minute
    maxViolations: 3,
    promptKey: `contest_fullscreen_prompt_${id}`,
    enabled: !isPracticeMode,
  });

  // Submissions data callbacks for SubmissionsPanel
  // NOTE: These hooks MUST be declared before any conditional returns
  // to comply with React's Rules of Hooks (hooks must always run in the same order)
  const fetchSubmissions = useCallback(
    (page: number) =>
      contestId && contestProblemId
        ? (contestsAPI.getSubmissions(contestId, contestProblemId, page) as Promise<{
            submissions?: Submission[];
            pagination?: PaginationInfo;
          }>)
        : Promise.resolve({
            submissions: [],
            pagination: {
              page,
              limit: 20,
              total: 0,
              total_pages: 1,
              has_next: false,
              has_prev: false,
            },
          }),
    [contestId, contestProblemId]
  );
  const fetchCode = useCallback(async (submissionId: number): Promise<string> => {
    const data = (await contestsAPI.getSubmissionCode(submissionId)) as { source_code: string };
    return data.source_code;
  }, []);

  // Data loading function - MUST be defined before useEffect that calls it
  // and before any early returns, to avoid TDZ (Temporal Dead Zone) errors
  const loadData = useCallback(async () => {
    try {
      if (!contestId || !contestProblemId) return;
      setLoading(true);
      const [contestData, problemData] = await Promise.all([
        contestsAPI.getById(contestId) as Promise<Contest>,
        contestsAPI.getProblem(contestId, contestProblemId) as Promise<Problem>,
      ]);

      setContest(contestData);
      setProblem(problemData);
      setIsSolved(!!problemData?.is_solved);
      // Contest mode is set below - disqualification is no longer enforced
      // but we still show warning if user has many violations

      // Set contest mode if this is an active contest the user has joined
      const now = new Date();
      const start = new Date(contestData.start_time);
      const end = new Date(contestData.end_time);
      if (contestData.has_joined && now >= start && now <= end && !contestData.has_finished) {
        setContestMode(
          contestData.id,
          contestData.start_time,
          contestData.end_time,
          contestData.title
        );
      }
    } catch (err) {
      console.error('Failed to load contest problem:', err);
      const redirectTo = (err as { originalResponse?: { redirect_to?: string } })?.originalResponse
        ?.redirect_to;
      if (redirectTo === 'contest_problems') {
        navigate(`/contests/${contestId}/problems`, { replace: true });
        return;
      }
      showError(
        typeof err === 'string' ? err : (err as Error)?.message || 'Failed to load problem'
      );
    } finally {
      setLoading(false);
    }
  }, [contestId, contestProblemId, navigate, setContestMode]);

  useEffect(() => {
    if (!contestId || !contestProblemId || contestProblemId === 'undefined') {
      showError('Invalid problem URL. Please go back and select a problem.');
      setLoading(false);
      return;
    }
    loadData();
  }, [loadData, contestId, contestProblemId]);

  // Reset editor state whenever the problem changes
  // This ensures each problem gets its own saved code (or the default)
  useEffect(() => {
    if (!contestProblemId || contestProblemId === 'undefined') return;

    const defaultLangId = 71;
    const rebuild: Record<string, string> = {};
    Object.keys(DEFAULT_CODE).forEach((langId) => {
      rebuild[langId] =
        localStorage.getItem(`code_${contestId}_${contestProblemId}_${langId}`) ??
        DEFAULT_CODE[Number(langId)];
    });
    setCodeByLang(rebuild);
    setLanguageId(defaultLangId);
    setCode(rebuild[String(defaultLangId)]);
    // Also reset results/tab so old run results don't persist
    setResults(null);
    setConsoleTab('testcase');
  }, [contestId, contestProblemId]);

  // ── Resizable Panels Hook - MUST be declared BEFORE conditional returns ──
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
    storageKey: `contest-split-${contestId}`,
    initialLeftPercent: 42,
    minLeftPx: 280,
    maxLeftPercent: 60,
    minRightPx: 360,
    initialConsoleHeight: 220,
    minConsoleHeight: 120,
    maxConsolePercent: 40,
    minEditorHeight: 220,
  });

  const handleEscAttempt = useCallback(async () => {
    if (isPracticeMode) return;
    if (isEscBlocked) return;
    setIsEscBlocked(true);
    try {
      // Report violation to backend for tracking
      const violation = contestId
        ? ((await contestsAPI.reportEscViolation(contestId)) as EscViolationResponse)
        : null;
      // Violation is recorded but user is NOT disqualified
      await showAlert({
        icon: 'warning',
        title: 'Fullscreen Exit Recorded',
        text: `Exiting fullscreen mode is recorded as a potential plagiarism indicator (violation #${violation?.esc_violations || violations + 1}). Please stay in fullscreen mode for contest integrity.`,
        confirmButtonText: 'OK, I understand',
        allowOutsideClick: false,
        allowEscapeKey: false,
      });
    } catch {
      // Even if API call fails, just show warning
      await showAlert({
        icon: 'warning',
        title: 'Fullscreen Exit Detected',
        text: 'Exiting fullscreen mode is recorded for integrity tracking. Please stay in fullscreen mode.',
        confirmButtonText: 'OK, I understand',
        allowOutsideClick: false,
        allowEscapeKey: false,
      });
    } finally {
      setIsEscBlocked(false);
    }
  }, [contestId, isEscBlocked, violations, isPracticeMode]);

  // Show initial full-screen prompt (placed AFTER all hooks to avoid Rules of Hooks violation)
  if (!isPracticeMode && showInitialPrompt && contest && contest.has_joined) {
    return (
      <div className="flex items-center justify-center h-screen bg-background-primary">
        <div className="card max-w-lg w-full text-center py-12 px-8">
          <div className="text-5xl mb-4">
            <svg
              width="64"
              height="64"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              style={{ margin: '0 auto', color: 'var(--accent-primary)' }}
            >
              <path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3"></path>
            </svg>
          </div>
          <h2 className="text-xl font-bold text-text-primary mb-2">Full-screen Mode Required</h2>
          <p className="text-text-secondary mb-6">
            This contest must be taken in full-screen mode for integrity purposes. Exiting
            fullscreen mode will be recorded as a potential plagiarism indicator.
          </p>
          <div className="bg-surface border border-border rounded-lg p-4 mb-6 text-left">
            <h4 className="text-text-primary font-semibold mb-2">Contest Rules:</h4>
            <ul className="text-text-secondary text-sm space-y-2">
              <li className="flex items-start gap-2">
                <span className="text-accent-warning">!</span>
                You must stay in full-screen mode during the contest
              </li>
              <li className="flex items-start gap-2">
                <span className="text-accent-warning">!</span>
                Exiting fullscreen is recorded for integrity tracking
              </li>
              <li className="flex items-start gap-2">
                <span className="text-accent-warning">!</span>
                Right-click and copy/paste are disabled
              </li>
              <li className="flex items-start gap-2">
                <span className="text-accent-warning">!</span>
                Violations are logged but you can continue participating
              </li>
            </ul>
          </div>
          <button onClick={requestFullscreen} className="btn btn-primary w-full">
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              style={{ marginRight: '8px' }}
            >
              <path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3"></path>
            </svg>
            Enter Full-screen & Continue
          </button>
          <p className="text-text-muted text-sm mt-4">
            You must enter full-screen mode to continue participating in this contest.
          </p>
        </div>
      </div>
    );
  }

  const handleLanguageChange = (newLangId: number) => {
    // Persist the current language's code before switching
    localStorage.setItem(storageKey(languageId), code);
    setCodeByLang((prev) => ({ ...prev, [languageId]: code }));

    const newCode = codeByLang[newLangId] ?? DEFAULT_CODE[newLangId] ?? '';
    setLanguageId(newLangId);
    setCode(newCode);
  };

  // Persist code to localStorage on every keystroke (debounce not needed — localStorage is synchronous & fast)
  const handleCodeChange = (newCode: string) => {
    setCode(newCode);
    localStorage.setItem(storageKey(languageId), newCode);
  };

  const handleRun = async () => {
    setRunning(true);
    setConsoleTab('result');
    setResults(null);
    try {
      if (!contestProblemId) {
        showError('Missing problem ID. Please refresh the page.');
        return;
      }
      const response = (await submissionsAPI.run(
        parseInt(contestProblemId, 10),
        languageId,
        code
      )) as {
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
      setResults({ error_message: (err as Error).message || 'Run failed', passed: false });
    } finally {
      setRunning(false);
    }
  };

  const handleSubmit = async () => {
    setSubmitting(true);
    setConsoleTab('result');
    setResults(null);
    try {
      if (!contestId || !contestProblemId) {
        showError('Missing contest or problem ID. Please refresh the page.');
        return;
      }
      const response = (await contestsAPI.submitSolution(
        contestId,
        contestProblemId,
        languageId,
        code
      )) as {
        job_id?: string;
      } & Record<string, unknown>;

      // Check if response contains a job_id (queue-based execution)
      if (response.job_id) {
        // Poll for job completion
        setResults({
          status: 'pending',
          message: (response as { message?: string }).message || 'Queued for execution...',
        });

        try {
          const jobResult = await pollJob(response.job_id, {
            onProgress: (status) => {
              setResults({
                status: (status as { status?: string }).status,
                message: (status as { message?: string }).message || 'Processing...',
                progress: (status as { progress?: unknown }).progress,
              });
            },
            timeout: 90000, // 90 seconds timeout for contest submissions
          });

          // Job completed - process results
          if (jobResult) {
            const finalResults = {
              all_passed: (jobResult as { all_passed?: boolean }).all_passed,
              passed: (jobResult as { all_passed?: boolean }).all_passed,
              total_tests: (jobResult as { total_tests?: number }).total_tests,
              passed_count: (jobResult as { passed_count?: number }).passed_count,
              score: (jobResult as { score?: number }).score,
              max_score: (jobResult as { max_score?: number }).max_score,
              test_results: (jobResult as { test_results?: unknown }).test_results,
            };
            setResults(finalResults);

            if (finalResults.all_passed || finalResults.passed) {
              setIsSolved(true);
              if (!isPracticeMode) {
                navigate(`/contests/${id}/problems`, { replace: true });
                return;
              }
            }
            // Refresh submissions list
            setSubmissionsRefreshKey((k) => k + 1);
          }
        } catch (pollError) {
          console.error('Job polling failed:', pollError);
          showError((pollError as Error).message || 'Execution timed out');
          setResults({
            error_message: (pollError as Error).message || 'Execution timed out',
            passed: false,
          });
        }
      } else {
        // Direct execution response (no job queue)
        setResults(response);
        if (
          (response as { all_passed?: boolean }).all_passed ||
          (response as { passed?: boolean }).passed
        ) {
          setIsSolved(true);
          if (!isPracticeMode) {
            navigate(`/contests/${id}/problems`, { replace: true });
            return;
          }
        }
        // Refresh submissions list
        setSubmissionsRefreshKey((k) => k + 1);
      }
    } catch (err) {
      console.error('Submission failed:', err);
      const redirectTo = (err as { originalResponse?: { redirect_to?: string } })?.originalResponse
        ?.redirect_to;
      if (redirectTo === 'contest_problems') {
        navigate(`/contests/${id}/problems`, { replace: true });
        return;
      }
      showError((err as Error).message || 'Submission failed');
      setResults({ error_message: (err as Error).message || 'Submission failed', passed: false });
    } finally {
      setSubmitting(false);
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

  // ── Loading ────────────────────────────────────────────────────────────────
  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <p className="text-text-muted">Loading problem...</p>
      </div>
    );
  }

  if (!problem) {
    return (
      <div className="flex items-center justify-center h-screen bg-background-primary">
        <div className="card max-w-md w-full text-center py-12 px-8">
          <div className="text-5xl mb-4">⚠️</div>
          <h2 className="text-xl font-bold text-accent-danger mb-2">Problem Not Found</h2>
          <p className="text-text-secondary mb-6">
            Unable to load this problem. Please go back and try again.
          </p>
          <Link to={`/contests/${id}`} className="btn btn-primary">
            ← Back to Contest
          </Link>
        </div>
      </div>
    );
  }

  // ── Main split-panel layout ────────────────────────────────────────────────
  return (
    <div className="problem-view no-navbar" style={{ position: 'relative' }}>
      {!isPracticeMode && isEscBlocked && (
        <div
          style={{
            position: 'fixed',
            inset: 0,
            background: 'rgba(0, 0, 0, 0.45)',
            zIndex: 10000,
            cursor: 'not-allowed',
          }}
        />
      )}
      {/* Problem Navigation Sidebar */}
      {showProblemNav && contest?.problems && (
        <div
          className="problem-nav-overlay"
          onClick={() => setShowProblemNav(false)}
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: 'rgba(8, 10, 14, 0.8)',
            zIndex: 1000,
          }}
        >
          <div
            className="problem-nav-sidebar"
            onClick={(e: React.MouseEvent<HTMLDivElement>) => e.stopPropagation()}
            style={{
              position: 'fixed',
              top: 0,
              left: 0,
              bottom: 0,
              width: '320px',
              backgroundColor: 'var(--background-primary, #111418)',
              borderRight: '1px solid var(--border)',
              boxShadow: '4px 0 20px rgba(0, 0, 0, 0.3)',
              overflowY: 'auto',
              zIndex: 1001,
            }}
          >
            {/* Sidebar Header */}
            <div
              style={{
                padding: '16px 20px',
                borderBottom: '1px solid var(--border)',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                backgroundColor: 'var(--background-primary, #111418)',
              }}
            >
              <h3 style={{ margin: 0, fontSize: '1.1rem', fontWeight: 600 }}>Contest Problems</h3>
              <button
                onClick={() => setShowProblemNav(false)}
                style={{
                  background: 'none',
                  border: 'none',
                  color: 'var(--text-muted)',
                  cursor: 'pointer',
                  padding: '4px',
                }}
              >
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                >
                  <line x1="18" y1="6" x2="6" y2="18"></line>
                  <line x1="6" y1="6" x2="18" y2="18"></line>
                </svg>
              </button>
            </div>

            {/* Progress Summary */}
            <div
              style={{
                padding: '16px 20px',
                borderBottom: '1px solid var(--border)',
                backgroundColor: 'var(--background-secondary, #171b22)',
              }}
            >
              <div
                style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '8px' }}
              >
                <span style={{ color: 'var(--text-muted)', fontSize: '0.85rem' }}>Progress</span>
                <span
                  style={{ color: 'var(--text-primary)', fontSize: '0.85rem', fontWeight: 600 }}
                >
                  {contest.problems.filter((p) => p.is_solved).length} / {contest.problems.length}{' '}
                  solved
                </span>
              </div>
              <div
                style={{
                  height: '6px',
                  backgroundColor: 'var(--bg-secondary)',
                  borderRadius: '3px',
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    height: '100%',
                    width: `${(contest.problems.filter((p) => p.is_solved).length / contest.problems.length) * 100}%`,
                    backgroundColor: 'var(--accent-success)',
                    borderRadius: '3px',
                    transition: 'width 0.3s ease',
                  }}
                />
              </div>
            </div>

            {/* Problems List */}
            <div style={{ padding: '12px 0' }}>
              {contest.problems.map((p, idx) => {
                const isActive = contestProblemId
                  ? p.problem_id === parseInt(contestProblemId, 10)
                  : false;

                return (
                  <div
                    key={p.problem_id}
                    onClick={() => {
                      if (!isActive) {
                        // Save current code before navigating away
                        localStorage.setItem(storageKey(languageId), code);
                        navigate(`/contests/${contestId}/problems/${p.problem_id}`);
                        setShowProblemNav(false);
                      }
                    }}
                    style={{
                      padding: '12px 20px',
                      display: 'flex',
                      alignItems: 'center',
                      gap: '12px',
                      cursor: isActive ? 'default' : 'pointer',
                      backgroundColor: isActive
                        ? 'rgba(59, 130, 246, 0.15)'
                        : 'var(--background-secondary, #171b22)',
                      borderLeft: isActive ? '3px solid var(--sky-500)' : '3px solid transparent',
                      transition: 'all 0.15s ease',
                      borderBottom: '1px solid var(--border-light, rgba(255,255,255,0.1))',
                    }}
                    onMouseEnter={(e: React.MouseEvent<HTMLDivElement>) => {
                      if (!isActive) {
                        e.currentTarget.style.backgroundColor = 'rgba(59, 130, 246, 0.08)';
                      }
                    }}
                    onMouseLeave={(e: React.MouseEvent<HTMLDivElement>) => {
                      if (!isActive) {
                        e.currentTarget.style.backgroundColor =
                          'var(--background-secondary, #171b22)';
                      }
                    }}
                  >
                    {/* Status Icon */}
                    <span
                      style={{
                        width: '28px',
                        height: '28px',
                        borderRadius: '50%',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        fontSize: '12px',
                        fontWeight: 'bold',
                        flexShrink: 0,
                        backgroundColor: p.is_solved
                          ? '#22c55e'
                          : p.is_attempted
                            ? '#f59e0b'
                            : 'var(--bg-secondary, #1f2937)',
                        color: p.is_solved || p.is_attempted ? '#fff' : 'var(--text-muted)',
                        border:
                          !p.is_solved && !p.is_attempted ? '1px solid var(--border)' : 'none',
                      }}
                    >
                      {p.is_solved ? '✓' : idx + 1}
                    </span>

                    {/* Problem Info */}
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div
                        style={{
                          fontWeight: isActive ? 600 : 500,
                          color: isActive ? 'var(--sky-500)' : 'var(--text-primary)',
                          whiteSpace: 'nowrap',
                          overflow: 'hidden',
                          textOverflow: 'ellipsis',
                          marginBottom: '2px',
                        }}
                      >
                        {p.title}
                      </div>
                      <div
                        style={{
                          display: 'flex',
                          gap: '8px',
                          fontSize: '0.75rem',
                          color: 'var(--text-muted)',
                        }}
                      >
                        <span
                          style={{
                            backgroundColor: 'var(--bg-secondary, #1f2937)',
                            padding: '1px 6px',
                            borderRadius: '3px',
                          }}
                        >
                          {p.points} pts
                        </span>
                        {p.difficulty && (
                          <span
                            style={{
                              color:
                                p.difficulty === 'easy'
                                  ? '#22c55e'
                                  : p.difficulty === 'medium'
                                    ? '#f59e0b'
                                    : '#ef4444',
                              fontWeight: 500,
                            }}
                          >
                            {p.difficulty}
                          </span>
                        )}
                      </div>
                    </div>

                    {/* Status Label */}
                    <span
                      style={{
                        fontSize: '0.65rem',
                        padding: '3px 8px',
                        borderRadius: '4px',
                        fontWeight: 600,
                        textTransform: 'uppercase',
                        letterSpacing: '0.5px',
                        backgroundColor: p.is_solved
                          ? 'rgba(34, 197, 94, 0.2)'
                          : p.is_attempted
                            ? 'rgba(245, 158, 11, 0.2)'
                            : 'var(--bg-secondary, #1f2937)',
                        color: p.is_solved
                          ? '#22c55e'
                          : p.is_attempted
                            ? '#f59e0b'
                            : 'var(--text-muted)',
                      }}
                    >
                      {p.is_solved ? 'Solved' : p.is_attempted ? 'Tried' : 'New'}
                    </span>
                  </div>
                );
              })}
            </div>

            {/* Back to Overview Button */}
            <div
              style={{
                padding: '16px 20px',
                borderTop: '1px solid var(--border)',
                position: 'sticky',
                bottom: 0,
                backgroundColor: 'var(--background-primary, #111418)',
              }}
            >
              <button
                onClick={() => navigate(`/contests/${contestId}/problems`)}
                className="btn btn-secondary"
                style={{ width: '100%' }}
              >
                <svg
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  style={{ marginRight: '8px' }}
                >
                  <line x1="19" y1="12" x2="5" y2="12"></line>
                  <polyline points="12 19 5 12 12 5"></polyline>
                </svg>
                Back to All Problems
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Breadcrumb Navigation */}
      <Breadcrumb
        items={[
          { label: contest?.title || 'Contest' },
          { label: 'Problems', to: `/contests/${contestId}/problems` },
          { label: problem?.title || 'Problem' },
        ]}
        backToPrevious
        actions={
          <div style={{ display: 'flex', gap: '8px' }}>
            {/* Problem Navigation Toggle */}
            <button
              onClick={() => setShowProblemNav(true)}
              className="btn btn-secondary flex items-center gap-2"
              style={{
                fontSize: '0.85rem',
                padding: '0.4rem 0.75rem',
                backgroundColor: 'var(--background-tertiary, #1b2130)',
                borderColor: 'var(--background-border, #2c3447)',
                opacity: 1,
              }}
            >
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
              >
                <line x1="3" y1="12" x2="21" y2="12"></line>
                <line x1="3" y1="6" x2="21" y2="6"></line>
                <line x1="3" y1="18" x2="21" y2="18"></line>
              </svg>
              Problems
              {contest?.problems && (
                <span
                  style={{
                    fontSize: '0.7rem',
                    backgroundColor: 'var(--accent-primary)',
                    color: '#fff',
                    padding: '2px 6px',
                    borderRadius: '10px',
                  }}
                >
                  {contest.problems.filter((p) => p.is_solved).length}/{contest.problems.length}
                </span>
              )}
            </button>
            <button
              onClick={() => navigate(`/contests/${contestId}/leaderboard`)}
              className="btn btn-secondary flex items-center gap-2"
              style={{ fontSize: '0.85rem', padding: '0.4rem 0.75rem' }}
            >
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
              >
                <rect x="4" y="14" width="4" height="8" rx="1"></rect>
                <rect x="10" y="6" width="4" height="16" rx="1"></rect>
                <rect x="16" y="10" width="4" height="12" rx="1"></rect>
              </svg>
              Leaderboard
            </button>
          </div>
        }
      />

      {/* Full-screen Warning Modal - Centered with light yellow theme */}
      {!isPracticeMode && showWarning && (
        <div
          style={{
            position: 'fixed',
            inset: 0,
            background: 'rgba(0, 0, 0, 0.6)',
            zIndex: 10001,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <div
            style={{
              background: '#fefce8', // Light yellow background
              borderRadius: '16px',
              padding: '32px 40px',
              maxWidth: '520px',
              width: '90%',
              textAlign: 'center',
              boxShadow: '0 20px 60px rgba(0, 0, 0, 0.4)',
              border: '2px solid #fbbf24', // Yellow border
            }}
          >
            {/* Warning Icon */}
            <div
              style={{
                width: '72px',
                height: '72px',
                borderRadius: '50%',
                background: '#fbbf24',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                margin: '0 auto 20px',
              }}
            >
              <svg
                width="36"
                height="36"
                viewBox="0 0 24 24"
                fill="none"
                stroke="white"
                strokeWidth="2.5"
              >
                <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
                <line x1="12" y1="9" x2="12" y2="13"></line>
                <line x1="12" y1="17" x2="12.01" y2="17"></line>
              </svg>
            </div>

            {/* Title */}
            <h2
              style={{
                color: '#92400e', // Dark amber/brown text
                fontSize: '1.4rem',
                fontWeight: '700',
                marginBottom: '10px',
              }}
            >
              Full-screen Mode Required
            </h2>

            {/* Message */}
            <p
              style={{
                color: '#78716c', // Warm gray
                fontSize: '1rem',
                marginBottom: '20px',
                lineHeight: '1.6',
              }}
            >
              You must stay in full-screen mode during the contest for integrity purposes.
              <br />
              <strong style={{ color: '#b45309' }}>
                Exiting fullscreen mode is recorded as a potential plagiarism indicator.
              </strong>
              <br />
              <span style={{ fontSize: '0.85rem', color: '#78716c' }}>
                Violation #{violations} recorded. Please re-enter fullscreen to continue.
              </span>
            </p>

            {/* Violation Counter */}
            <div
              style={{
                display: 'flex',
                justifyContent: 'center',
                gap: '10px',
                marginBottom: '24px',
              }}
            >
              <span
                style={{
                  padding: '6px 12px',
                  backgroundColor: '#fef3c7',
                  border: '1px solid #fbbf24',
                  borderRadius: '6px',
                  color: '#92400e',
                  fontSize: '0.85rem',
                  fontWeight: '600',
                }}
              >
                Fullscreen Exits: {violations} (logged for integrity)
              </span>
            </div>

            {/* Action Button */}
            <button
              onClick={requestFullscreen}
              style={{
                backgroundColor: '#f59e0b',
                color: 'white',
                border: 'none',
                padding: '12px 28px',
                borderRadius: '10px',
                fontWeight: '600',
                fontSize: '1rem',
                cursor: 'pointer',
                display: 'inline-flex',
                alignItems: 'center',
                gap: '8px',
                transition: 'all 0.2s ease',
              }}
              onMouseEnter={(e: React.MouseEvent<HTMLButtonElement>) => {
                e.currentTarget.style.backgroundColor = '#d97706';
                e.currentTarget.style.transform = 'scale(1.02)';
              }}
              onMouseLeave={(e: React.MouseEvent<HTMLButtonElement>) => {
                e.currentTarget.style.backgroundColor = '#f59e0b';
                e.currentTarget.style.transform = 'scale(1)';
              }}
            >
              <svg
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
              >
                <path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3"></path>
              </svg>
              Re-enter Full-screen
            </button>

            <p
              style={{
                color: '#a16207',
                fontSize: '0.85rem',
                marginTop: '16px',
              }}
            >
              Click the button above to return to full-screen mode and continue.
            </p>
          </div>
        </div>
      )}

      <div className="split-container" ref={splitRef}>
        {/* Left Panel: Problem Description + Submissions */}
        <div className="left-panel" style={{ width: `${leftWidthPercent}%` }}>
          <div className="panel-tabs">
            <button
              className={`tab-btn ${activeTab === 'description' ? 'active' : ''}`}
              onClick={() => setActiveTab('description')}
            >
              Description
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
                <div className="problem-detail">
                  {/* Title + meta badges */}
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '0.75rem',
                      marginBottom: '0.5rem',
                      flexWrap: 'wrap',
                    }}
                  >
                    <h2 style={{ margin: 0 }}>{problem?.title}</h2>
                    {isSolved && (
                      <span
                        style={{
                          display: 'inline-flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          width: '24px',
                          height: '24px',
                          borderRadius: '50%',
                          background: 'var(--accent-success, #22c55e)',
                          color: '#fff',
                          fontSize: '14px',
                          fontWeight: 'bold',
                          flexShrink: 0,
                        }}
                        title="Solved"
                      >
                        ✓
                      </span>
                    )}
                    {problem?.difficulty && (
                      <span className={`difficulty ${getDifficultyClass(problem.difficulty)}`}>
                        {problem.difficulty}
                      </span>
                    )}
                    {(problem?.points_in_contest ?? 0) > 0 && (
                      <span className="tag" style={{ marginLeft: '0.5rem' }}>
                        {problem.points_in_contest} pts
                      </span>
                    )}
                  </div>

                  {/* Description */}
                  {problem?.description && (
                    <div style={{ whiteSpace: 'pre-wrap', marginBottom: '1.5rem' }}>
                      {problem.description}
                    </div>
                  )}

                  {/* Input format */}
                  {problem?.input_format && (
                    <section style={{ marginBottom: '1.5rem' }}>
                      <h3
                        style={{
                          fontSize: '1rem',
                          fontWeight: 600,
                          marginBottom: '0.5rem',
                          color: 'var(--text-primary)',
                        }}
                      >
                        Input Format
                      </h3>
                      <div
                        style={{
                          background: 'var(--surface)',
                          borderRadius: '6px',
                          padding: '0.75rem',
                        }}
                      >
                        <pre style={{ margin: 0, whiteSpace: 'pre-wrap', fontSize: '0.9rem' }}>
                          {problem.input_format}
                        </pre>
                      </div>
                    </section>
                  )}

                  {/* Output format */}
                  {problem?.output_format && (
                    <section style={{ marginBottom: '1.5rem' }}>
                      <h3
                        style={{
                          fontSize: '1rem',
                          fontWeight: 600,
                          marginBottom: '0.5rem',
                          color: 'var(--text-primary)',
                        }}
                      >
                        Output Format
                      </h3>
                      <div
                        style={{
                          background: 'var(--surface)',
                          borderRadius: '6px',
                          padding: '0.75rem',
                        }}
                      >
                        <pre style={{ margin: 0, whiteSpace: 'pre-wrap', fontSize: '0.9rem' }}>
                          {problem.output_format}
                        </pre>
                      </div>
                    </section>
                  )}

                  {/* Constraints */}
                  {problem?.constraints && (
                    <section style={{ marginBottom: '1.5rem' }}>
                      <h3
                        style={{
                          fontSize: '1rem',
                          fontWeight: 600,
                          marginBottom: '0.5rem',
                          color: 'var(--text-primary)',
                        }}
                      >
                        Constraints
                      </h3>
                      <div
                        style={{
                          background: 'var(--surface)',
                          borderRadius: '6px',
                          padding: '0.75rem',
                        }}
                      >
                        <pre style={{ margin: 0, whiteSpace: 'pre-wrap', fontSize: '0.9rem' }}>
                          {problem.constraints}
                        </pre>
                      </div>
                    </section>
                  )}

                  {/* Sample test cases */}
                  {problem?.sample_test_cases && problem.sample_test_cases.length > 0 && (
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
                      {problem.sample_test_cases.map((tc, i) => (
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

                  {/* Limits */}
                  <div style={{ marginTop: '1.5rem', color: 'var(--text-muted)' }}>
                    <div>Time Limit: {problem?.time_limit || 2000}ms</div>
                    <div>
                      Memory Limit: {((problem?.memory_limit || 256000) / 1024).toFixed(0)}MB
                    </div>
                  </div>

                  {/* Tags */}
                  {problem?.tags && (
                    <div style={{ marginTop: '1rem' }}>
                      {problem.tags
                        .split(',')
                        .map((t) => t.trim())
                        .filter(Boolean)
                        .map((tag) => (
                          <span key={tag} className="tag" style={{ marginRight: '0.5rem' }}>
                            {tag}
                          </span>
                        ))}
                    </div>
                  )}
                </div>
              </div>
            )}

            {activeTab === 'submissions' && (
              <div className="tab-pane active" style={{ height: '100%' }}>
                <SubmissionsPanel
                  fetchSubmissions={fetchSubmissions}
                  fetchCode={fetchCode}
                  refreshKey={submissionsRefreshKey}
                />
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
            {/* Editor Header with Language Selector, Timer, and Submit */}
            <div className="editor-header">
              <LanguageSelector value={languageId} onChange={handleLanguageChange} />
              <div className="editor-actions">
                {contest && (
                  <div className="solve-timer-pill">
                    <SolveTimer
                      startTime={new Date(contest.start_time).toISOString()}
                      endTime={new Date(contest.end_time).toISOString()}
                    />
                  </div>
                )}
                {/* Violation Counter */}
                {!isPracticeMode && violations > 0 && (
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '4px',
                      padding: '4px 8px',
                      backgroundColor: '#fef3c7', // Light yellow
                      border: '1px solid #fbbf24',
                      color: '#92400e',
                      borderRadius: '4px',
                      fontSize: '0.75rem',
                      fontWeight: '600',
                    }}
                    title="Fullscreen exits are logged for integrity tracking"
                  >
                    <svg
                      width="14"
                      height="14"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="#f59e0b"
                      strokeWidth="2"
                    >
                      <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
                      <line x1="12" y1="9" x2="12" y2="13"></line>
                      <line x1="12" y1="17" x2="12.01" y2="17"></line>
                    </svg>
                    Exits: {violations} (logged)
                  </div>
                )}
                {/* Fullscreen Status Indicator */}
                {!isPracticeMode && !isFullscreen && !showWarning && (
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '4px',
                      padding: '4px 8px',
                      backgroundColor: '#fef3c7',
                      border: '1px solid #fbbf24',
                      color: '#92400e',
                      borderRadius: '4px',
                      fontSize: '0.75rem',
                      fontWeight: '600',
                      cursor: 'pointer',
                    }}
                    onClick={requestFullscreen}
                    title="Click to enter full-screen mode"
                  >
                    <svg
                      width="14"
                      height="14"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                    >
                      <path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3"></path>
                    </svg>
                    Enter Full-screen
                  </div>
                )}
                <button
                  className="btn btn-secondary"
                  onClick={handleRun}
                  disabled={running || submitting || (!isPracticeMode && isEscBlocked)}
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
                  {running ? 'Running...' : 'Run'}
                </button>
                <button
                  className="btn btn-success"
                  onClick={handleSubmit}
                  disabled={submitting || running || (!isPracticeMode && isEscBlocked)}
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
                  {submitting ? 'Submitting...' : 'Submit'}
                </button>
              </div>
            </div>

            <div style={{ position: 'relative', flex: 1, minHeight: 0 }}>
              <CodeEditor
                languageId={languageId}
                code={code}
                onChange={handleCodeChange}
                onEscape={!isPracticeMode ? handleEscAttempt : undefined}
                readOnly={!isPracticeMode && isEscBlocked}
              />
              {!isPracticeMode && isEscBlocked && (
                <div
                  style={{
                    position: 'absolute',
                    inset: 0,
                    background: 'rgba(0, 0, 0, 0.35)',
                    zIndex: 5,
                    cursor: 'not-allowed',
                  }}
                />
              )}
            </div>

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
                  <TestCasePanel testCases={problem?.sample_test_cases || []} />
                ) : (
                  <ResultsPanel
                    results={results as ExecutionResult}
                    loading={running || submitting || isPolling}
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
    </div>
  );
}

export default ContestProblemPage;
