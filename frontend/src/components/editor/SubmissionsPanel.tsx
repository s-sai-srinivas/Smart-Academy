import { useState, useEffect, useCallback } from 'react';
import Editor from '@monaco-editor/react';
import { LANGUAGE_CONFIG, formatTimeAgo, formatMemory } from '../../constants/judge0';

export interface Submission {
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

export interface PaginationInfo {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
  has_next: boolean;
  has_prev: boolean;
}

export interface SubmissionsPanelProps {
  fetchSubmissions?: (
    page: number
  ) => Promise<{ submissions?: Submission[]; pagination?: PaginationInfo }>;
  fetchCode?: (submissionId: number) => Promise<string>;
  refreshKey?: number | string;
}

function SubmissionsPanel({ fetchSubmissions, fetchCode, refreshKey }: SubmissionsPanelProps) {
  const [submissions, setSubmissions] = useState<Submission[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedSubmission, setSelectedSubmission] = useState<Submission | null>(null);
  const [view, setView] = useState<'list' | 'detail'>('list');
  const [codeLoading, setCodeLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pagination, setPagination] = useState<PaginationInfo>({
    page: 1,
    limit: 20,
    total: 0,
    total_pages: 0,
    has_next: false,
    has_prev: false,
  });

  const loadSubmissions = useCallback(
    async (pageNum = 1) => {
      if (!fetchSubmissions) return;
      setLoading(true);
      try {
        const data = await fetchSubmissions(pageNum);
        setSubmissions(data.submissions || []);
        setPagination(
          data.pagination || {
            page: pageNum,
            limit: 20,
            total: 0,
            total_pages: 1,
            has_next: false,
            has_prev: false,
          }
        );
        setPage(pageNum);
      } catch (err) {
        console.error('Failed to load submissions:', err);
      } finally {
        setLoading(false);
      }
    },
    [fetchSubmissions]
  );

  useEffect(() => {
    loadSubmissions(1);
  }, [loadSubmissions, refreshKey]);

  const handleSubmissionClick = async (submission: Submission) => {
    setSelectedSubmission({ ...submission, source_code: null });
    setView('detail');
    if (fetchCode) {
      setCodeLoading(true);
      try {
        const code = await fetchCode(submission.id);
        setSelectedSubmission((prev) => (prev ? { ...prev, source_code: code } : prev));
      } catch {
        setSelectedSubmission((prev) =>
          prev ? { ...prev, source_code: '// Error loading code' } : prev
        );
      } finally {
        setCodeLoading(false);
      }
    }
  };

  const handleBack = () => {
    setSelectedSubmission(null);
    setView('list');
  };

  if (view === 'detail' && selectedSubmission) {
    const langInfo = LANGUAGE_CONFIG[selectedSubmission.language_id];
    return (
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        <div
          style={{
            padding: '0.75rem 1rem',
            background: 'var(--surface)',
            borderBottom: '1px solid var(--border)',
            display: 'flex',
            alignItems: 'center',
            gap: '0.75rem',
          }}
        >
          <button
            onClick={handleBack}
            style={{
              background: 'transparent',
              border: 'none',
              color: 'var(--text-secondary)',
              cursor: 'pointer',
              padding: '0.35rem',
              display: 'flex',
              alignItems: 'center',
              gap: '0.35rem',
              borderRadius: '4px',
              fontSize: '0.85rem',
            }}
          >
            <svg
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
            >
              <polyline points="15 18 9 12 15 6" />
            </svg>
            Back
          </button>
          <div
            style={{
              flex: 1,
              display: 'flex',
              alignItems: 'center',
              gap: '0.5rem',
              flexWrap: 'wrap',
            }}
          >
            <span
              style={{
                padding: '0.2rem 0.6rem',
                borderRadius: '4px',
                fontSize: '0.75rem',
                fontWeight: 600,
                background: selectedSubmission.passed
                  ? 'rgba(16,185,129,0.15)'
                  : 'rgba(239,68,68,0.15)',
                color: selectedSubmission.passed
                  ? 'var(--success, #10b981)'
                  : 'var(--error, #ef4444)',
              }}
            >
              {selectedSubmission.passed ? 'Accepted' : 'Wrong Answer'}
            </span>
            <span style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
              {langInfo?.name || `Lang ${selectedSubmission.language_id}`}
            </span>
            <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>
              {new Date(selectedSubmission.submitted_at).toLocaleString()}
            </span>
          </div>
        </div>
        <div style={{ flex: 1, minHeight: 0, position: 'relative' }}>
          {codeLoading && (
            <div
              style={{
                position: 'absolute',
                inset: 0,
                background: 'rgba(0,0,0,0.6)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                zIndex: 10,
                color: 'var(--text-muted)',
              }}
            >
              Loading code...
            </div>
          )}
          <Editor
            height="100%"
            language={langInfo?.monacoLang || 'python'}
            value={selectedSubmission.source_code || '// Loading code...'}
            theme="vs-dark"
            options={{
              readOnly: true,
              minimap: { enabled: false },
              fontSize: 14,
              scrollBeyondLastLine: false,
              padding: { top: 10 },
              contextmenu: false,
              lineNumbers: 'on',
            }}
          />
        </div>
        {selectedSubmission.passed && (
          <div
            style={{
              padding: '0.6rem 1rem',
              background: 'var(--surface)',
              borderTop: '1px solid var(--border)',
              display: 'flex',
              gap: '1.25rem',
              fontSize: '0.85rem',
              flexWrap: 'wrap',
            }}
          >
            {selectedSubmission.execution_time != null && (
              <div>
                <span style={{ color: 'var(--text-muted)' }}>Runtime: </span>
                <strong>{selectedSubmission.execution_time.toFixed(2)}ms</strong>
              </div>
            )}
            {selectedSubmission.memory_used != null && selectedSubmission.memory_used > 0 && (
              <div>
                <span style={{ color: 'var(--text-muted)' }}>Memory: </span>
                <strong>{formatMemory(selectedSubmission.memory_used)}</strong>
              </div>
            )}
            {selectedSubmission.total_tests != null && (
              <div>
                <span style={{ color: 'var(--text-muted)' }}>Tests: </span>
                <strong>
                  {selectedSubmission.passed_tests}/{selectedSubmission.total_tests} passed
                </strong>
              </div>
            )}
          </div>
        )}
      </div>
    );
  }

  return (
    <div style={{ height: '100%', overflowY: 'auto' }}>
      {loading ? (
        <div style={{ padding: '3rem', textAlign: 'center', color: 'var(--text-muted)' }}>
          Loading submissions...
        </div>
      ) : submissions.length === 0 ? (
        <div style={{ padding: '3rem', textAlign: 'center', color: 'var(--text-muted)' }}>
          <div style={{ fontSize: '2rem', marginBottom: '0.5rem' }}>📝</div>
          <p>No submissions yet. Submit your code to see it here!</p>
        </div>
      ) : (
        <>
          {submissions.map((sub, index) => {
            const langInfo = LANGUAGE_CONFIG[sub.language_id];
            return (
              <div
                key={sub.id}
                onClick={() => handleSubmissionClick(sub)}
                style={{
                  padding: '0.85rem 1rem',
                  borderBottom: index < submissions.length - 1 ? '1px solid var(--border)' : 'none',
                  cursor: 'pointer',
                  transition: 'background 0.15s',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '0.75rem',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = 'var(--surface)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'transparent';
                }}
              >
                <div
                  style={{
                    width: '36px',
                    height: '36px',
                    borderRadius: '8px',
                    background: sub.passed ? 'rgba(16,185,129,0.12)' : 'rgba(239,68,68,0.12)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    flexShrink: 0,
                  }}
                >
                  {sub.passed ? (
                    <svg
                      width="18"
                      height="18"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="var(--success, #10b981)"
                      strokeWidth="3"
                    >
                      <polyline points="20 6 9 17 4 12" />
                    </svg>
                  ) : (
                    <svg
                      width="18"
                      height="18"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="var(--error, #ef4444)"
                      strokeWidth="3"
                    >
                      <line x1="18" y1="6" x2="6" y2="18" />
                      <line x1="6" y1="6" x2="18" y2="18" />
                    </svg>
                  )}
                </div>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '0.5rem',
                      marginBottom: '0.15rem',
                    }}
                  >
                    <span
                      style={{
                        fontWeight: 600,
                        fontSize: '0.9rem',
                        color: sub.passed ? 'var(--success, #10b981)' : 'var(--error, #ef4444)',
                      }}
                    >
                      {sub.passed ? 'Accepted' : 'Wrong Answer'}
                    </span>
                    <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>
                      {langInfo?.name || `Lang ${sub.language_id}`}
                    </span>
                  </div>
                  <div style={{ fontSize: '0.7rem', color: 'var(--text-muted)' }}>
                    {formatTimeAgo(sub.submitted_at)}
                  </div>
                </div>
                {sub.passed && (
                  <div
                    style={{
                      display: 'flex',
                      gap: '0.75rem',
                      fontSize: '0.75rem',
                      color: 'var(--text-muted)',
                    }}
                  >
                    {sub.execution_time != null && <span>{sub.execution_time.toFixed(2)}ms</span>}
                    {sub.memory_used != null && sub.memory_used > 0 && (
                      <span>{formatMemory(sub.memory_used)}</span>
                    )}
                    {sub.total_tests != null && (
                      <span>
                        {sub.passed_tests}/{sub.total_tests}
                      </span>
                    )}
                  </div>
                )}
                <svg
                  width="14"
                  height="14"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="var(--text-muted)"
                  strokeWidth="2"
                  style={{ flexShrink: 0 }}
                >
                  <polyline points="9 18 15 12 9 6" />
                </svg>
              </div>
            );
          })}
          {pagination.total_pages > 1 && (
            <div
              style={{
                padding: '0.75rem 1rem',
                borderTop: '1px solid var(--border)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                gap: '0.75rem',
              }}
            >
              <button
                onClick={() => loadSubmissions(page - 1)}
                disabled={!pagination.has_prev || loading}
                style={{
                  padding: '0.4rem 0.75rem',
                  borderRadius: '4px',
                  border: '1px solid var(--border)',
                  background: 'var(--surface)',
                  color: 'var(--text-primary)',
                  fontSize: '0.8rem',
                  cursor: pagination.has_prev && !loading ? 'pointer' : 'not-allowed',
                  opacity: pagination.has_prev && !loading ? 1 : 0.5,
                }}
              >
                Prev
              </button>
              <span style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                {pagination.page} / {pagination.total_pages}
              </span>
              <button
                onClick={() => loadSubmissions(page + 1)}
                disabled={!pagination.has_next || loading}
                style={{
                  padding: '0.4rem 0.75rem',
                  borderRadius: '4px',
                  border: '1px solid var(--border)',
                  background: 'var(--surface)',
                  color: 'var(--text-primary)',
                  fontSize: '0.8rem',
                  cursor: pagination.has_next && !loading ? 'pointer' : 'not-allowed',
                  opacity: pagination.has_next && !loading ? 1 : 0.5,
                }}
              >
                Next
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

export default SubmissionsPanel;
