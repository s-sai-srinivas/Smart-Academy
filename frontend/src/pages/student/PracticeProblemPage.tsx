import { useState, useEffect, useRef, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import { practiceAPI } from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import { useNotification } from '../../context/NotificationContext';
import { showError } from '../../utils/showAlert';
import Breadcrumb from '../../components/common/Breadcrumb';
import CodeEditor from '../../components/editor/CodeEditor';
import { DEFAULT_CODE } from '../../components/editor/codeEditorConstants';
import LanguageSelector from '../../components/editor/LanguageSelector';
import TestCasePanel from '../../components/editor/TestCasePanel';
import ResultsPanel, { type ExecutionResult } from '../../components/editor/ResultsPanel';
import { STREAK_UPDATED_EVENT } from '../../components/common/Navbar';
import StreakPopup from '../../components/common/StreakPopup';
import useResizablePanels from '../../hooks/useResizablePanels';
import useJobPolling from '../../hooks/useJobPolling';
import type { Problem, TestCase } from '../../types/problem';

interface Submission {
  id: number;
  passed: boolean;
  passed_tests: number;
  total_tests: number;
  submitted_at: string;
}

interface ExtendedProblem extends Problem {
  test_cases?: TestCase[];
  tags?: string;
}

interface ProblemData {
  problem: ExtendedProblem;
  is_completed?: boolean;
}

function PracticeProblemPage() {
  const { problemId } = useParams<{ problemId: string }>();
  const { user } = useAuth();
  const notification = useNotification();

  const [problem, setProblem] = useState<ExtendedProblem | null>(null);
  const [testCases, setTestCases] = useState<TestCase[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [isCompleted, setIsCompleted] = useState<boolean>(false);

  // Timer state
  const [isSolving, setIsSolving] = useState<boolean>(false);
  const [timer, setTimer] = useState<number>(0);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Editor state
  const [languageId, setLanguageId] = useState<number>(71);
  const [code, setCode] = useState<string>(DEFAULT_CODE[71]);
  const [codeByLang, setCodeByLang] = useState<Record<number, string>>(() => {
    const initial: Record<number, string> = {};
    Object.keys(DEFAULT_CODE).forEach((id) => {
      initial[Number(id)] = DEFAULT_CODE[Number(id)];
    });
    return initial;
  });
  const [activeTab, setActiveTab] = useState<'description' | 'submissions'>('description');
  const [consoleTab, setConsoleTab] = useState<'testcase' | 'result'>('testcase');
  const [results, setResults] = useState<unknown>(null);
  const [running, setRunning] = useState<boolean>(false);
  const [submissions, setSubmissions] = useState<Submission[]>([]);

  // Job polling hook for async submissions
  const { pollJob, isPolling, pollProgress } = useJobPolling();

  // Streak popup state
  const [showStreakPopup, setShowStreakPopup] = useState<boolean>(false);
  const [currentStreak, setCurrentStreak] = useState<number>(0);

  // Resizable panels
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
    storageKey: `practice-split-${problemId}`,
    initialLeftPercent: 42,
    minLeftPx: 280,
    maxLeftPercent: 60,
    minRightPx: 360,
    initialConsoleHeight: 220,
    minConsoleHeight: 120,
    maxConsolePercent: 40,
    minEditorHeight: 220,
  });

  // Timer effect
  useEffect(() => {
    if (isSolving) {
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
  }, [isSolving]);

  const formatTime = (seconds: number): string => {
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;
    if (hrs > 0) {
      return `${hrs}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    }
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const getStorageKey = useCallback((): string => {
    return `practice_code_${problemId}_${user?.regdno || 'anonymous'}`;
  }, [problemId, user?.regdno]);

  const saveCodeToStorage = useCallback(() => {
    try {
      const key = getStorageKey();
      const data = {
        codeByLang,
        languageId,
        timer,
        lastSaved: new Date().toISOString(),
      };
      localStorage.setItem(key, JSON.stringify(data));
    } catch (e) {
      console.warn('Could not save code to localStorage:', e);
    }
  }, [codeByLang, languageId, timer, getStorageKey]);

  const loadSavedCode = useCallback(() => {
    try {
      const key = getStorageKey();
      const saved = localStorage.getItem(key);
      if (saved) {
        const data = JSON.parse(saved) as {
          codeByLang?: Record<number, string>;
          languageId?: number;
          timer?: number;
        };
        const mergedCodeByLang = { ...DEFAULT_CODE };
        if (data.codeByLang) {
          Object.keys(data.codeByLang).forEach((langId) => {
            mergedCodeByLang[Number(langId)] = data.codeByLang?.[Number(langId)] || '';
          });
        }
        setCodeByLang(mergedCodeByLang);
        setLanguageId(data.languageId || 71);
        setCode(mergedCodeByLang[data.languageId || 71] || DEFAULT_CODE[71]);
        setTimer(data.timer || 0);
        setIsSolving(true);
      } else {
        setIsSolving(true);
      }
    } catch (e) {
      console.warn('Could not load saved code:', e);
      setIsSolving(true);
    }
  }, [getStorageKey]);

  const clearSavedCode = () => {
    try {
      localStorage.removeItem(getStorageKey());
    } catch (e) {
      console.warn('Could not clear saved code:', e);
    }
  };

  const handleLanguageChange = (newLangId: number) => {
    const newCode = codeByLang[newLangId] || DEFAULT_CODE[newLangId] || '';
    setCodeByLang((prev) => ({ ...prev, [languageId]: code }));
    setLanguageId(newLangId);
    setCode(newCode);
  };

  const loadSubmissions = useCallback(async () => {
    try {
      const data = (await practiceAPI.getSubmissions(problemId || '')) as {
        submissions?: Submission[];
      };
      setSubmissions(data.submissions || []);
    } catch (err) {
      console.debug('Could not load submissions:', err);
    }
  }, [problemId]);

  const loadProblem = useCallback(async () => {
    try {
      const data = (await practiceAPI.getProblem(problemId || '')) as ProblemData | ExtendedProblem;
      const problemData: ExtendedProblem = 'problem' in data ? data.problem : data;
      setProblem(problemData);
      setIsCompleted(('is_completed' in data ? data.is_completed : false) || false);

      if (problemData.test_cases) {
        setTestCases(problemData.test_cases);
      }

      // Load submissions
      loadSubmissions();
    } catch (err) {
      showError((err as Error).message || 'Failed to load problem');
    } finally {
      setLoading(false);
    }
  }, [problemId, loadSubmissions]);

  useEffect(() => {
    loadProblem();
    loadSavedCode();

    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current);
      }
    };
  }, [problemId, loadProblem, loadSavedCode]);

  const handleRun = async () => {
    if (!user) {
      notification.error('Please login to run your code');
      return;
    }

    setRunning(true);
    setConsoleTab('result');
    setResults(null);

    try {
      const response = (await practiceAPI.run(parseInt(problemId || '0'), languageId, code)) as {
        job_id?: string;
      };

      // Check if response contains job_id (async job queue)
      if (response.job_id) {
        const jobResult = await pollJob(response.job_id, {
          onProgress: (status: unknown) =>
            setResults({
              status: (status as { status?: string }).status,
              message: (status as { message?: string }).message,
            }),
          timeout: 60000,
        });

        if (jobResult) {
          const resultObj = jobResult as Record<string, unknown>;
          setResults({
            all_passed: resultObj.all_passed,
            passed: resultObj.all_passed,
            test_results: resultObj.test_results,
            ...resultObj,
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

    setRunning(true);
    setConsoleTab('result');
    setResults(null);

    try {
      const response = (await practiceAPI.submit(
        parseInt(problemId || '0'),
        languageId,
        code,
        timer
      )) as { job_id?: string };

      // Check if response contains job_id (async job queue)
      if (response.job_id) {
        const jobResult = await pollJob(response.job_id, {
          onProgress: (status: unknown) =>
            setResults({
              status: (status as { status?: string }).status,
              message: (status as { message?: string }).message,
            }),
          timeout: 90000,
        });

        if (jobResult) {
          const resultObj = jobResult as Record<string, unknown>;
          setResults({
            all_passed: resultObj.all_passed,
            passed: resultObj.all_passed,
            test_results: resultObj.test_results,
            ...resultObj,
          });
          loadSubmissions();

          if ((jobResult as { all_passed?: boolean }).all_passed && !isCompleted) {
            setIsCompleted(true);
            setIsSolving(false);
            if (timerRef.current) {
              clearInterval(timerRef.current);
            }
            clearSavedCode();
            notification.success('Congratulations! Problem solved!');
            // Notify Navbar to refresh streak count
            window.dispatchEvent(new CustomEvent(STREAK_UPDATED_EVENT));

            // Fetch updated streak and show popup
            try {
              const token = localStorage.getItem('token');
              const authValue = token === 'httpOnly' ? 'httpOnly' : `Bearer ${token}`;
              const meResponse = await fetch('/api/me', {
                headers: { Authorization: authValue },
              });
              if (meResponse.ok) {
                const meData = (await meResponse.json()) as { streak?: number };
                setCurrentStreak(meData.streak || 1);
                setShowStreakPopup(true);
              }
            } catch (err) {
              console.warn('Could not fetch streak for popup:', err);
            }
          }
        }
      } else {
        // Direct response (no job queue)
        setResults(response);
        loadSubmissions();

        if ((response as { all_passed?: boolean }).all_passed && !isCompleted) {
          setIsCompleted(true);
          setIsSolving(false);
          if (timerRef.current) {
            clearInterval(timerRef.current);
          }
          clearSavedCode();
          notification.success('Congratulations! Problem solved!');
          // Notify Navbar to refresh streak count
          window.dispatchEvent(new CustomEvent(STREAK_UPDATED_EVENT));

          // Fetch updated streak and show popup
          try {
            const token = localStorage.getItem('token');
            const authValue = token === 'httpOnly' ? 'httpOnly' : `Bearer ${token}`;
            const meResponse = await fetch('/api/me', {
              headers: { Authorization: authValue },
            });
            if (meResponse.ok) {
              const meData = (await meResponse.json()) as { streak?: number };
              setCurrentStreak(meData.streak || 1);
              setShowStreakPopup(true);
            }
          } catch (err) {
            console.warn('Could not fetch streak for popup:', err);
          }
        }
      }
    } catch (err) {
      setResults({ error_message: (err as Error).message, passed: false });
    } finally {
      setRunning(false);
    }
  };

  const getDifficultyClass = (difficulty?: string): string => {
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

  // Save code periodically
  useEffect(() => {
    if (isSolving && code) {
      const interval = setInterval(saveCodeToStorage, 30000);
      return () => clearInterval(interval);
    }
  }, [code, isSolving, saveCodeToStorage]);

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
    <div className="problem-view">
      {/* Breadcrumb Navigation */}
      <Breadcrumb
        items={[
          { label: 'Dashboard', to: '/dashboard' },
          { label: 'Practice', to: '/practice' },
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
              className={`tab-btn ${activeTab === 'submissions' ? 'active' : ''}`}
              onClick={() => setActiveTab('submissions')}
            >
              Submissions ({submissions.length})
            </button>
          </div>

          <div className="tab-content">
            {activeTab === 'description' && (
              <div className="tab-pane active">
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
                          color: 'white',
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
                      </span>
                    )}
                  </div>
                  <div style={{ marginBottom: '1rem' }}>
                    <span className={`difficulty ${getDifficultyClass(problem.difficulty)}`}>
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

                  {/* Sample Test Cases */}
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

            {activeTab === 'submissions' && (
              <div className="tab-pane active">
                {!user ? (
                  <div style={{ padding: '3rem', textAlign: 'center', color: 'var(--text-muted)' }}>
                    Please login to view your submissions
                  </div>
                ) : submissions.length === 0 ? (
                  <div style={{ padding: '3rem', textAlign: 'center', color: 'var(--text-muted)' }}>
                    No submissions yet. Start solving!
                  </div>
                ) : (
                  <div className="submissions-list">
                    {submissions.map((sub, i) => (
                      <div
                        key={sub.id || i}
                        className="submission-item"
                        style={{
                          padding: '0.75rem',
                          borderBottom: '1px solid var(--border)',
                          display: 'flex',
                          justifyContent: 'space-between',
                          alignItems: 'center',
                        }}
                      >
                        <div>
                          <span
                            style={{
                              color: sub.passed ? 'var(--success)' : 'var(--error)',
                              fontWeight: 600,
                            }}
                          >
                            {sub.passed ? 'Accepted' : 'Failed'}
                          </span>
                          <span
                            style={{
                              marginLeft: '1rem',
                              color: 'var(--text-muted)',
                              fontSize: '0.85rem',
                            }}
                          >
                            {sub.passed_tests}/{sub.total_tests} tests
                          </span>
                        </div>
                        <div style={{ color: 'var(--text-muted)', fontSize: '0.85rem' }}>
                          {new Date(sub.submitted_at).toLocaleString()}
                        </div>
                      </div>
                    ))}
                  </div>
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
                <button className="btn btn-secondary" onClick={handleRun} disabled={running}>
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
                <button className="btn btn-success" onClick={handleSubmit} disabled={running}>
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
                {isSolving && (
                  <div className="solve-timer-pill" title="Time spent solving">
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

      {/* Streak Popup */}
      {showStreakPopup && (
        <StreakPopup streak={currentStreak} onClose={() => setShowStreakPopup(false)} />
      )}
    </div>
  );
}

export default PracticeProblemPage;
