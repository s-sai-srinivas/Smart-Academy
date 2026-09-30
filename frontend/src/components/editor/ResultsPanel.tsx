export interface SingleTestResult {
  passed?: boolean;
  status_id?: number;
  status?: string;
  is_hidden?: boolean;
  input?: string;
  expected_output?: string;
  stdout?: string;
  actual_output?: string;
  stderr?: string;
  compile_output?: string;
  points?: number;
  earned_points?: number;
}

export interface ExecutionResult {
  total_tests?: number;
  totalTests?: number;
  passed_tests?: number;
  passedTests?: number;
  score?: number;
  max_score?: number;
  all_passed?: boolean;
  passed?: boolean;
  error_message?: string;
  execution_time?: number;
  memory_used?: number;
  test_results?: SingleTestResult[];
  runtime?: { display?: string; percentile?: number; faster_than?: string };
  memory?: { display?: string; percentile?: number; lower_than?: string };
}

interface ResultsPanelProps {
  results?: ExecutionResult;
  loading?: boolean;
  pollProgress?: { message?: string; status?: string; retry_count?: number } | null;
  pollError?: string | null;
}

function ResultsPanel({ results, loading, pollProgress, pollError }: ResultsPanelProps) {
  if (pollProgress) {
    return (
      <div className="results-content">
        <div
          style={{
            padding: '1.5rem',
            textAlign: 'center',
            background: 'var(--surface, #1e1e2e)',
            borderRadius: '8px',
            border: '1px solid var(--border)',
          }}
        >
          <div
            style={{
              width: '48px',
              height: '48px',
              margin: '0 auto 1rem',
              border: '3px solid var(--border)',
              borderTopColor: 'var(--accent-primary)',
              borderRadius: '50%',
              animation: 'spin 1s linear infinite',
            }}
          />
          <div
            style={{
              fontSize: '1rem',
              fontWeight: 600,
              color: 'var(--text-primary)',
              marginBottom: '0.5rem',
            }}
          >
            {pollProgress.message || 'Processing...'}
          </div>
          <div
            style={{
              display: 'inline-block',
              padding: '0.25rem 0.75rem',
              borderRadius: '20px',
              fontSize: '0.8rem',
              fontWeight: 500,
              background:
                pollProgress.status === 'running'
                  ? 'var(--accent-primary-soft)'
                  : 'var(--accent-secondary-soft)',
              color: pollProgress.status === 'running' ? 'var(--sky-500)' : 'var(--amber-500)',
            }}
          >
            {pollProgress.status === 'running' ? 'Running Tests' : 'Waiting in Queue'}
          </div>
          {(pollProgress.retry_count ?? 0) > 0 && (
            <div style={{ marginTop: '0.75rem', fontSize: '0.75rem', color: 'var(--text-muted)' }}>
              Retry attempt #{pollProgress.retry_count}
            </div>
          )}
          <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
        </div>
      </div>
    );
  }

  if (pollError) {
    return (
      <div className="results-content">
        <div
          style={{
            padding: '1rem',
            background: 'var(--error, #ef4444)15',
            borderRadius: '6px',
            border: '1px solid var(--error, #ef4444)',
          }}
        >
          <strong
            style={{
              color: 'var(--error, #ef4444)',
              display: 'flex',
              alignItems: 'center',
              gap: '0.5rem',
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
              <circle cx="12" cy="12" r="10"></circle>
              <line x1="12" y1="8" x2="12" y2="12"></line>
              <line x1="12" y1="16" x2="12.01" y2="16"></line>
            </svg>
            Execution Error
          </strong>
          <pre
            style={{
              marginTop: '0.5rem',
              whiteSpace: 'pre-wrap',
              fontSize: '0.85rem',
              color: 'var(--error, #ef4444)',
            }}
          >
            {pollError}
          </pre>
        </div>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="results-content">
        <div className="text-muted">Running your code...</div>
      </div>
    );
  }

  if (!results) {
    return (
      <div className="results-content">
        <p className="text-muted">Run your code to see results here...</p>
      </div>
    );
  }

  const res = results;
  const totalTests = res.total_tests ?? res.totalTests ?? 0;
  const passedTests = res.passed_tests ?? res.passedTests ?? 0;
  const score = res.score ?? 0;
  const maxScore = res.max_score ?? 0;
  const errorMessage = res.error_message;
  const testResults = res.test_results;

  const parseStatusId = (statusId: unknown): number | null => {
    const parsed = Number(statusId);
    return Number.isFinite(parsed) ? parsed : null;
  };

  const isAcceptedTest = (tr: SingleTestResult): boolean => {
    if (typeof tr?.passed === 'boolean') return tr.passed;
    const statusId = parseStatusId(tr?.status_id);
    if (statusId !== null) return statusId === 3;
    const statusText = String(tr?.status || '').toLowerCase();
    return statusText.includes('accepted') || statusText.includes('passed');
  };

  const getStatusInfo = (statusId: number | null, statusDescription?: string) => {
    const statusMap: Record<number, { label: string; color: string; bgColor: string }> = {
      1: { label: 'In Queue', color: '#f59e0b', bgColor: '#f59e0b15' },
      2: { label: 'Processing', color: '#f59e0b', bgColor: '#f59e0b15' },
      3: { label: 'Accepted', color: '#10b981', bgColor: '#10b98115' },
      4: { label: 'Wrong Answer', color: '#ef4444', bgColor: '#ef444415' },
      5: { label: 'Time Limit Exceeded', color: '#f59e0b', bgColor: '#f59e0b15' },
      6: { label: 'Compilation Error', color: '#ef4444', bgColor: '#ef444415' },
      7: { label: 'Runtime Error (SIGSEGV)', color: '#ef4444', bgColor: '#ef444415' },
      8: { label: 'Runtime Error (SIGXFSZ)', color: '#ef4444', bgColor: '#ef444415' },
      9: { label: 'Runtime Error (SIGFPE)', color: '#ef4444', bgColor: '#ef444415' },
      10: { label: 'Runtime Error (SIGABRT)', color: '#ef4444', bgColor: '#ef444415' },
      11: { label: 'Runtime Error (NZEC)', color: '#ef4444', bgColor: '#ef444415' },
      12: { label: 'Runtime Error (Other)', color: '#ef4444', bgColor: '#ef444415' },
      13: { label: 'Internal Error', color: '#ef4444', bgColor: '#ef444415' },
      14: { label: 'Exec Limit Exceeded', color: '#f59e0b', bgColor: '#f59e0b15' },
    };
    return (
      statusMap[statusId ?? -1] || {
        label: statusDescription || 'Unknown',
        color: '#6b7280',
        bgColor: '#6b728015',
      }
    );
  };

  const getTestBreakdown = (trs?: SingleTestResult[]) => {
    if (!trs) return { passed: 0, failed: 0, error: 0, tle: 0, total: 0 };
    return trs.reduce(
      (acc, tr) => {
        const statusId = parseStatusId(tr.status_id);
        const statusText = String(tr.status || '').toLowerCase();
        const passed = isAcceptedTest(tr);
        acc.total += 1;
        if (passed) {
          acc.passed += 1;
          return acc;
        }
        if (statusId === 4 || statusText.includes('wrong answer')) {
          acc.failed += 1;
          return acc;
        }
        if (statusId === 5 || statusId === 14 || statusText.includes('time limit')) {
          acc.tle += 1;
          return acc;
        }
        if (
          (statusId !== null && statusId >= 6 && statusId <= 13) ||
          statusText.includes('error')
        ) {
          acc.error += 1;
          return acc;
        }
        acc.failed += 1;
        return acc;
      },
      { passed: 0, failed: 0, error: 0, tle: 0, total: 0 }
    );
  };

  const breakdown = getTestBreakdown(testResults);
  const derivedTotalTests = breakdown.total;
  const derivedPassedTests = breakdown.passed;
  const normalizedTotalTests = derivedTotalTests > 0 ? derivedTotalTests : totalTests;
  const normalizedPassedTests = derivedTotalTests > 0 ? derivedPassedTests : passedTests;
  const normalizedAllPassed =
    derivedTotalTests > 0
      ? derivedPassedTests === derivedTotalTests
      : (res.all_passed ??
        res.passed ??
        (normalizedTotalTests > 0 && normalizedPassedTests === normalizedTotalTests));

  const compileError = testResults?.find((tr) => tr.compile_output)?.compile_output;

  return (
    <div className="results-content">
      <div
        style={{
          padding: '0.75rem',
          borderRadius: '6px',
          background: normalizedAllPassed ? 'var(--success, #10b981)' : 'var(--error, #ef4444)',
          color: 'white',
          marginBottom: '1rem',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <strong style={{ fontSize: '1rem', display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          {normalizedAllPassed ? (
            <>
              <svg
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="3"
              >
                <polyline points="20 6 9 17 4 12"></polyline>
              </svg>
              All Tests Passed
            </>
          ) : (
            <>
              <svg
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="3"
              >
                <line x1="18" y1="6" x2="6" y2="18"></line>
                <line x1="6" y1="6" x2="18" y2="18"></line>
              </svg>
              Some Tests Failed
            </>
          )}
        </strong>
        <span style={{ fontSize: '0.9rem', fontWeight: 500 }}>
          {normalizedPassedTests}/{normalizedTotalTests} tests passed
        </span>
      </div>
      {maxScore > 0 && (
        <div
          style={{
            marginBottom: '1rem',
            padding: '0.75rem',
            background: 'var(--surface, #1e1e2e)',
            borderRadius: '6px',
            border: '1px solid var(--border)',
          }}
        >
          <span className="text-muted">Score: </span>
          <span
            style={{
              fontSize: '1.25rem',
              fontWeight: 700,
              color: score === maxScore ? 'var(--success, #10b981)' : 'var(--warning, #f59e0b)',
            }}
          >
            {score}/{maxScore}
          </span>
          <span
            style={{
              marginLeft: '1rem',
              color: score === maxScore ? 'var(--success, #10b981)' : 'var(--text-muted)',
            }}
          >
            {score === maxScore ? '✓ Perfect Score!' : `${Math.round((score / maxScore) * 100)}%`}
          </span>
        </div>
      )}
      {testResults && testResults.length > 0 && (
        <div style={{ marginBottom: '1rem', display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
          <span
            style={{
              padding: '0.25rem 0.75rem',
              borderRadius: '20px',
              fontSize: '0.8rem',
              fontWeight: 600,
              background: '#10b98120',
              color: '#10b981',
            }}
          >
            ✓ {breakdown.passed} Passed
          </span>
          {breakdown.failed > 0 && (
            <span
              style={{
                padding: '0.25rem 0.75rem',
                borderRadius: '20px',
                fontSize: '0.8rem',
                fontWeight: 600,
                background: '#ef444420',
                color: '#ef4444',
              }}
            >
              ✗ {breakdown.failed} Wrong Answer
            </span>
          )}
          {breakdown.error > 0 && (
            <span
              style={{
                padding: '0.25rem 0.75rem',
                borderRadius: '20px',
                fontSize: '0.8rem',
                fontWeight: 600,
                background: '#ef444420',
                color: '#ef4444',
              }}
            >
              ⚠ {breakdown.error} Runtime Error
            </span>
          )}
          {breakdown.tle > 0 && (
            <span
              style={{
                padding: '0.25rem 0.75rem',
                borderRadius: '20px',
                fontSize: '0.8rem',
                fontWeight: 600,
                background: '#f59e0b20',
                color: '#f59e0b',
              }}
            >
              ⏱ {breakdown.tle} Time Limit
            </span>
          )}
        </div>
      )}
      {compileError && (
        <div
          style={{
            marginBottom: '1rem',
            padding: '1rem',
            background: 'var(--error, #ef4444)15',
            borderRadius: '6px',
            border: '1px solid var(--error, #ef4444)',
          }}
        >
          <strong
            style={{
              color: 'var(--error, #ef4444)',
              display: 'flex',
              alignItems: 'center',
              gap: '0.5rem',
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
              <circle cx="12" cy="12" r="10"></circle>
              <line x1="12" y1="8" x2="12" y2="12"></line>
              <line x1="12" y1="16" x2="12.01" y2="16"></line>
            </svg>
            Compilation Error
          </strong>
          <pre
            style={{
              marginTop: '0.5rem',
              whiteSpace: 'pre-wrap',
              fontSize: '0.85rem',
              color: 'var(--error, #ef4444)',
              background: 'var(--bg-secondary, #1a1a2e)',
              padding: '0.75rem',
              borderRadius: '4px',
              maxHeight: '200px',
              overflow: 'auto',
            }}
          >
            {compileError}
          </pre>
        </div>
      )}
      {errorMessage && !compileError && (
        <div
          style={{
            marginBottom: '1rem',
            padding: '1rem',
            background: 'var(--error, #ef4444)15',
            borderRadius: '6px',
            border: '1px solid var(--error, #ef4444)',
          }}
        >
          <strong style={{ color: 'var(--error, #ef4444)' }}>Error:</strong>
          <pre style={{ whiteSpace: 'pre-wrap', marginTop: '0.5rem' }}>{errorMessage}</pre>
        </div>
      )}
      <div style={{ marginBottom: '1rem', display: 'flex', gap: '1.5rem', flexWrap: 'wrap' }}>
        {res.runtime && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                style={{ color: 'var(--text-muted)' }}
              >
                <circle cx="12" cy="12" r="10"></circle>
                <polyline points="12 6 12 12 16 14"></polyline>
              </svg>
              <span className="text-muted">Runtime: </span>
              <span style={{ fontWeight: 600, color: '#10b981' }}>{res.runtime.display}</span>
            </div>
            {(res.runtime.percentile ?? 0) > 0 && (
              <span style={{ fontSize: '0.75rem', color: '#10b981', marginLeft: '1.5rem' }}>
                {res.runtime.faster_than}
              </span>
            )}
          </div>
        )}
        {res.memory && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                style={{ color: 'var(--text-muted)' }}
              >
                <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path>
              </svg>
              <span className="text-muted">Memory: </span>
              <span style={{ fontWeight: 600, color: 'var(--sky-500)' }}>{res.memory.display}</span>
            </div>
            {(res.memory.percentile ?? 0) > 0 && (
              <span style={{ fontSize: '0.75rem', color: 'var(--sky-500)', marginLeft: '1.5rem' }}>
                {res.memory.lower_than}
              </span>
            )}
          </div>
        )}
      </div>
      {testResults && testResults.length > 0 && (
        <div style={{ marginTop: '1rem' }}>
          <strong style={{ fontSize: '0.95rem', marginBottom: '0.75rem', display: 'block' }}>
            Test Results
          </strong>
          {testResults.map((tr, i) => {
            const statusInfo = getStatusInfo(parseStatusId(tr.status_id), tr.status);
            const isHidden = !!tr.is_hidden;
            const testPassed = isAcceptedTest(tr);
            return (
              <div
                key={i}
                style={{
                  marginTop: '0.5rem',
                  padding: '0.75rem',
                  background: 'var(--surface, #1e1e2e)',
                  borderRadius: '6px',
                  border: `1px solid ${testPassed ? 'var(--success, #10b981)' : 'var(--error, #ef4444)'}30`,
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    marginBottom: '0.5rem',
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                    <span
                      style={{
                        padding: '0.2rem 0.6rem',
                        borderRadius: '4px',
                        fontSize: '0.75rem',
                        fontWeight: 600,
                        background: statusInfo.bgColor,
                        color: statusInfo.color,
                      }}
                    >
                      {statusInfo.label}
                    </span>
                    <span style={{ fontWeight: 600, color: 'var(--text-primary)' }}>
                      {isHidden ? `Hidden Test ${i + 1}` : `Test ${i + 1}`}
                    </span>
                  </div>
                  {(tr.points ?? 0) > 0 && (
                    <span
                      style={{
                        fontSize: '0.8rem',
                        fontWeight: 600,
                        color: testPassed ? 'var(--success, #10b981)' : 'var(--error, #ef4444)',
                        background: `${testPassed ? 'var(--success, #10b981)' : 'var(--error, #ef4444)'}15`,
                        padding: '0.2rem 0.5rem',
                        borderRadius: '4px',
                      }}
                    >
                      {tr.earned_points ?? 0}/{tr.points} pts
                    </span>
                  )}
                </div>
                {!isHidden && tr.input && (
                  <div style={{ marginTop: '0.5rem', fontSize: '0.875rem' }}>
                    <span className="text-muted">Input: </span>
                    <pre
                      style={{
                        margin: '0.25rem 0',
                        padding: '0.4rem 0.6rem',
                        background: 'var(--bg-secondary, #1a1a2e)',
                        borderRadius: '4px',
                        fontSize: '0.8rem',
                        whiteSpace: 'pre-wrap',
                      }}
                    >
                      {tr.input}
                    </pre>
                  </div>
                )}
                {!isHidden && tr.expected_output && (
                  <div style={{ marginTop: '0.25rem', fontSize: '0.875rem' }}>
                    <span className="text-muted">Expected: </span>
                    <pre
                      style={{
                        margin: '0.25rem 0',
                        padding: '0.4rem 0.6rem',
                        background: 'var(--bg-secondary, #1a1a2e)',
                        borderRadius: '4px',
                        fontSize: '0.8rem',
                        whiteSpace: 'pre-wrap',
                      }}
                    >
                      {tr.expected_output}
                    </pre>
                  </div>
                )}
                {(tr.stdout || tr.actual_output) && (
                  <div style={{ marginTop: '0.25rem', fontSize: '0.875rem' }}>
                    <span className="text-muted">Your Output: </span>
                    <pre
                      style={{
                        margin: '0.25rem 0',
                        padding: '0.4rem 0.6rem',
                        background: 'var(--bg-secondary, #1a1a2e)',
                        borderRadius: '4px',
                        fontSize: '0.8rem',
                        whiteSpace: 'pre-wrap',
                        color: testPassed ? 'var(--success, #10b981)' : 'var(--error, #ef4444)',
                      }}
                    >
                      {tr.stdout || tr.actual_output}
                    </pre>
                  </div>
                )}
                {tr.stderr && (
                  <div style={{ marginTop: '0.25rem', fontSize: '0.875rem' }}>
                    <span className="text-muted" style={{ color: 'var(--error, #ef4444)' }}>
                      Stderr:{' '}
                    </span>
                    <pre
                      style={{
                        margin: '0.25rem 0',
                        padding: '0.4rem 0.6rem',
                        background: 'var(--bg-secondary, #1a1a2e)',
                        borderRadius: '4px',
                        fontSize: '0.8rem',
                        whiteSpace: 'pre-wrap',
                        color: 'var(--error, #ef4444)',
                      }}
                    >
                      {tr.stderr}
                    </pre>
                  </div>
                )}
                {isHidden && (
                  <div
                    style={{
                      marginTop: '0.5rem',
                      fontSize: '0.75rem',
                      color: 'var(--text-muted)',
                      fontStyle: 'italic',
                    }}
                  >
                    * Hidden test case - input and expected output are hidden
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

export default ResultsPanel;
