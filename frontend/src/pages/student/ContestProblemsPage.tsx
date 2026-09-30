import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { contestsAPI } from '../../services/api';
import { showAlert, showError, showSuccess, showConfirm } from '../../utils/showAlert';
import Breadcrumb from '../../components/common/Breadcrumb';
import SolveTimer from '../../components/editor/SolveTimer';
import useContestFullscreen from '../../hooks/useContestFullscreen';
import { useContestMode } from '../../context/ContestModeContext';

interface Contest {
  id: number;
  title: string;
  start_time: string;
  end_time: string;
  is_practice_active?: boolean;
  has_joined?: boolean;
  has_finished?: boolean;
  has_disqualified?: boolean;
  disqualification_reason?: string;
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

function ContestProblemsPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const contestId = id ?? '';
  const [contest, setContest] = useState<Contest | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [finishing, setFinishing] = useState<boolean>(false);
  const [hasFinished, setHasFinished] = useState<boolean>(false);
  const isPracticeMode = Boolean(
    contest?.is_practice_active && contest?.end_time && new Date() > new Date(contest.end_time)
  );

  // Contest mode context
  const {
    isInContestMode: _isInContestMode,
    activeContestId: _activeContestId,
    setContestMode,
    exitContestMode,
    finishContest: finishContestFromContext,
    isAllowedPath: _isAllowedPath,
  } = useContestMode();

  // Full-screen enforcement for contest integrity - tracks violations for plagiarism records, does NOT disqualify
  const {
    isFullscreen,
    showInitialPrompt,
    showWarning,
    violations,
    maxViolations,
    requestFullscreen,
  } = useContestFullscreen({
    warningDuration: 60000, // 1 minute
    maxViolations: 3,
    promptKey: `contest_fullscreen_prompt_${id}`,
    enabled: !isPracticeMode,
  });

  // Load contest data
  const loadContest = useCallback(async () => {
    try {
      setLoading(true);
      if (!contestId) return;
      const data = (await contestsAPI.getById(contestId)) as Contest;
      setContest(data);
      // Note: has_disqualified may still be set for OTHER reasons (admin action, etc.)
      // but fullscreen exits no longer cause disqualification
      if (data?.has_disqualified) {
        await showAlert({
          icon: 'error',
          title: 'Disqualified',
          text: data.disqualification_reason || 'You have been disqualified from this contest.',
          confirmButtonText: 'OK, I understand',
          allowOutsideClick: false,
          allowEscapeKey: false,
        });
        exitContestMode();
        navigate('/contests', { replace: true });
        return;
      }
      // Check if user has finished the contest (from participant data or local state)
      setHasFinished(data?.has_finished || false);

      // Set contest mode if this is an active contest the user has joined
      const now = new Date();
      const start = new Date(data.start_time);
      const end = new Date(data.end_time);
      if (data.has_joined && now >= start && now <= end && !data.has_finished) {
        setContestMode(data.id, data.start_time, data.end_time, data.title);
      }
    } catch (err) {
      console.error('Failed to load contest:', err);
      showError((err as Error).message || 'Failed to load contest');
    } finally {
      setLoading(false);
    }
  }, [contestId, setContestMode, exitContestMode, navigate]);

  useEffect(() => {
    loadContest();
  }, [loadContest]);

  // Handle Finish Contest
  const handleFinishContest = async () => {
    if (!contest) return;

    const result = await showConfirm(
      'Are you sure you want to finish the contest?',
      'Once finished, you cannot make any more submissions. Your current submissions will be finalized for scoring.',
      'Yes, Finish Contest',
      'Cancel'
    );

    if (!result.isConfirmed) return;

    try {
      setFinishing(true);
      if (!contestId) return;
      await contestsAPI.finish(contestId);
      setHasFinished(true);
      finishContestFromContext(); // Exit contest mode via context
      showSuccess('Contest finished successfully! Your submissions have been finalized.');
      navigate(`/contests/${contestId}/leaderboard`);
    } catch (err) {
      console.error('Failed to finish contest:', err);
      showError((err as Error).message || 'Failed to finish contest');
    } finally {
      setFinishing(false);
    }
  };

  // Check contest status
  const getContestStatus = () => {
    if (!contest) return { status: 'unknown', label: 'Unknown', class: 'badge-neutral' };
    const now = new Date();
    const start = new Date(contest.start_time);
    const end = new Date(contest.end_time);

    if (now < start) return { status: 'upcoming', label: 'Upcoming', class: 'badge-info' };
    if (now > end) return { status: 'ended', label: 'Ended', class: 'badge-neutral' };
    return { status: 'active', label: 'Active', class: 'badge-success' };
  };

  // Get problem status badge
  const getProblemStatusBadge = (problem: ContestProblem) => {
    if (problem.is_solved) {
      return { label: 'Solved', class: 'status-solved', icon: '✓' };
    }
    if (problem.is_attempted) {
      return { label: 'Attempted', class: 'status-attempted', icon: '◐' };
    }
    return { label: 'Not Attempted', class: 'status-not-attempted', icon: '○' };
  };

  // Get difficulty class
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

  // Format time
  const formatDateTime = (dateStr: string): string => {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      hour12: true,
    });
  };

  const formatDuration = (start: string, end: string): string => {
    const startDate = new Date(start);
    const endDate = new Date(end);
    const diffMs = endDate.getTime() - startDate.getTime();
    const hours = Math.floor(diffMs / (1000 * 60 * 60));
    const minutes = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60));
    return `${hours}h ${minutes}m`;
  };

  // Calculate progress stats
  const getProgressStats = () => {
    if (!contest?.problems) return { solved: 0, attempted: 0, total: 0, score: 0, maxScore: 0 };

    const total = contest.problems.length;
    const solved = contest.problems.filter((p) => p.is_solved).length;
    const attempted = contest.problems.filter((p) => p.is_attempted && !p.is_solved).length;

    // Calculate score from solved problems
    const score = contest.problems
      .filter((p) => p.is_solved)
      .reduce((sum, p) => sum + (p.points || 0), 0);

    const maxScore = contest.problems.reduce((sum, p) => sum + (p.points || 0), 0);

    return { solved, attempted, total, score, maxScore };
  };

  // Loading state
  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen bg-background-primary">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-accent-primary mx-auto mb-4"></div>
          <p className="text-text-muted">Loading contest...</p>
        </div>
      </div>
    );
  }

  // Show initial full-screen prompt (for active contests only)
  const contestStatusForPrompt = getContestStatus();
  if (
    !isPracticeMode &&
    showInitialPrompt &&
    contest &&
    contest.has_joined &&
    contestStatusForPrompt.status === 'active' &&
    !hasFinished
  ) {
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
            Enter Full-screen & Start Contest
          </button>
          <button
            onClick={() => navigate('/contests')}
            className="block mt-4 text-text-muted hover:text-text-secondary"
          >
            Cancel and leave contest
          </button>
        </div>
      </div>
    );
  }

  // Contest not found or not joined
  if (!contest) {
    return (
      <div className="flex items-center justify-center h-screen bg-background-primary">
        <div className="card max-w-md w-full text-center py-12 px-8">
          <div className="text-5xl mb-4">⚠️</div>
          <h2 className="text-xl font-bold text-accent-danger mb-2">Contest Not Found</h2>
          <p className="text-text-secondary mb-6">
            Unable to load this contest. Please go back and try again.
          </p>
          <Link to="/contests" className="btn btn-primary">
            ← Back to Contests
          </Link>
        </div>
      </div>
    );
  }

  const contestStatus = getContestStatus();
  const progressStats = getProgressStats();

  // Check if contest hasn't started yet
  if (contestStatus.status === 'upcoming') {
    return (
      <div className="p-6 max-w-4xl mx-auto">
        <Breadcrumb
          items={[
            { label: 'Contests', to: '/contests' },
            { label: contest.title, to: `/contests/${id}` },
            { label: 'Problems' },
          ]}
        />
        <div className="card text-center py-12">
          <div className="text-5xl mb-4">⏳</div>
          <h2 className="text-xl font-bold text-text-primary mb-2">Contest Has Not Started</h2>
          <p className="text-text-secondary mb-4">
            The contest begins at {formatDateTime(contest.start_time)}
          </p>
          <div className="inline-flex items-center gap-2 bg-background-tertiary border border-border-light rounded-lg px-4 py-2">
            <SolveTimer
              startTime={new Date(contest.start_time).toISOString()}
              endTime={new Date(contest.end_time).toISOString()}
            />
          </div>
          <Link to="/contests" className="btn btn-secondary mt-6">
            ← Back to Contests
          </Link>
        </div>
      </div>
    );
  }

  // Contest ended
  if (contestStatus.status === 'ended') {
    // Check if practice mode is active
    if (contest.is_practice_active) {
      // Practice mode is active - allow access
      // This will be handled in the main view below with a practice mode banner
    } else {
      return (
        <div className="p-6 max-w-4xl mx-auto">
          <Breadcrumb
            items={[
              { label: 'Contests', to: '/contests' },
              { label: contest.title, to: `/contests/${id}` },
              { label: 'Problems' },
            ]}
          />
          <div className="card text-center py-12">
            <div className="text-5xl mb-4">🏁</div>
            <h2 className="text-xl font-bold text-text-primary mb-2">Contest Has Ended</h2>
            <p className="text-text-secondary mb-4">
              This contest ended at {formatDateTime(contest.end_time)}
            </p>
            <div className="grid grid-cols-2 gap-4 max-w-xs mx-auto mb-6">
              <div className="bg-surface border border-border rounded-lg p-3">
                <div className="text-2xl font-bold text-accent-success">{progressStats.solved}</div>
                <div className="text-sm text-text-muted">Solved</div>
              </div>
              <div className="bg-surface border border-border rounded-lg p-3">
                <div className="text-2xl font-bold text-text-primary">{progressStats.score}</div>
                <div className="text-sm text-text-muted">Points</div>
              </div>
            </div>
            <button
              onClick={() => navigate(`/contests/${id}/leaderboard`)}
              className="btn btn-primary mb-4"
            >
              View Leaderboard
            </button>
            <Link to="/contests" className="btn btn-secondary block mx-auto">
              ← Back to Contests
            </Link>
          </div>
        </div>
      );
    }
  }

  // Not joined yet (except when practice mode is active for ended contests)
  const canAccessViaPractice = contestStatus.status === 'ended' && contest.is_practice_active;
  if (!contest.has_joined && !canAccessViaPractice) {
    return (
      <div className="p-6 max-w-4xl mx-auto">
        <Breadcrumb
          items={[
            { label: 'Contests', to: '/contests' },
            { label: contest.title, to: `/contests/${id}` },
            { label: 'Problems' },
          ]}
        />
        <div className="card text-center py-12">
          <div className="text-5xl mb-4">🔒</div>
          <h2 className="text-xl font-bold text-text-primary mb-2">
            Join Contest to View Problems
          </h2>
          <p className="text-text-secondary mb-6">
            You need to join this contest to see the problems and participate.
          </p>
          <Link to={`/contests/${id}`} className="btn btn-primary">
            Go to Contest Details
          </Link>
        </div>
      </div>
    );
  }

  // Main Problems Overview (Active Contest, Joined)
  return (
    <div className="p-4 md:p-6 max-w-5xl mx-auto">
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

            {/* Violation Counter Circles */}
            <div
              style={{
                display: 'flex',
                justifyContent: 'center',
                gap: '10px',
                marginBottom: '24px',
              }}
            >
              {Array.from({ length: maxViolations }, (_, i) => (
                <div
                  key={i}
                  style={{
                    width: '36px',
                    height: '36px',
                    borderRadius: '50%',
                    background: i < violations ? '#fbbf24' : '#fef9c3',
                    border: '2px solid ' + (i < violations ? '#f59e0b' : '#fde047'),
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    color: i < violations ? 'white' : '#a16207',
                    fontWeight: '600',
                    fontSize: '0.85rem',
                    transition: 'all 0.3s ease',
                  }}
                >
                  {i < violations ? '✗' : i + 1}
                </div>
              ))}
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

      {/* Violation Counter - Top Right */}
      {!isPracticeMode && !showWarning && violations > 0 && (
        <div
          style={{
            position: 'fixed',
            top: '12px',
            right: '20px',
            display: 'flex',
            alignItems: 'center',
            gap: '4px',
            padding: '6px 12px',
            backgroundColor: '#fef3c7', // Light yellow
            border: '1px solid #fbbf24',
            color: '#92400e',
            borderRadius: '6px',
            fontSize: '0.85rem',
            fontWeight: '600',
            zIndex: 9998,
          }}
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
          Fullscreen Exits: {violations} (logged for integrity)
        </div>
      )}

      {/* Breadcrumb Navigation */}
      <Breadcrumb
        items={[
          { label: 'Contests', to: '/contests' },
          { label: contest.title, to: `/contests/${id}` },
          { label: 'Problems' },
        ]}
        actions={
          <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
            {/* Fullscreen Status Indicator */}
            {!isPracticeMode && !isFullscreen && !showWarning && !showInitialPrompt && (
              <button
                onClick={requestFullscreen}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '4px',
                  padding: '4px 8px',
                  backgroundColor: '#f59e0b',
                  color: 'white',
                  borderRadius: '4px',
                  fontSize: '0.75rem',
                  fontWeight: '600',
                  cursor: 'pointer',
                  border: 'none',
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
                  <path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3"></path>
                </svg>
                Enter Full-screen
              </button>
            )}
            <button
              onClick={() => navigate(`/contests/${id}/leaderboard`)}
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

      {/* Simplified Contest Header */}
      <div className="mb-5 flex flex-col md:flex-row md:items-center md:justify-between gap-3">
        <div>
          <div className="flex items-center gap-3 mb-1">
            <h1 className="text-2xl font-bold text-text-primary">{contest.title}</h1>
            <span className={`badge ${contestStatus.class}`}>{contestStatus.label}</span>
            {/* Practice Mode Badge */}
            {contest.is_practice_active && (
              <span className="badge bg-accent-primary/20 text-accent-primary border border-accent-primary/30">
                Practice Mode
              </span>
            )}
            {hasFinished && <span className="badge badge-success">Finished</span>}
          </div>
          <div className="text-sm text-text-muted">
            Duration: {formatDuration(contest.start_time, contest.end_time)} | Ends:{' '}
            {formatDateTime(contest.end_time)}
          </div>
        </div>

        <div className="solve-timer-pill bg-surface border border-border rounded-lg px-4 py-2 w-fit">
          <SolveTimer
            startTime={new Date(contest.start_time).toISOString()}
            endTime={new Date(contest.end_time).toISOString()}
          />
        </div>
      </div>

      {/* Practice Mode Notice Banner */}
      {contest.is_practice_active && contestStatus.status === 'ended' && (
        <div className="mb-4 bg-gradient-to-r from-amber-50 to-yellow-50 border border-amber-200 rounded-lg p-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-full bg-amber-100 flex items-center justify-center">
              <svg
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                className="text-amber-700"
              >
                <path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3 3H2z"></path>
                <path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3 3h7z"></path>
              </svg>
            </div>
            <div>
              <h3 className="text-lg font-semibold text-amber-900">Practice Mode Active</h3>
              <p className="text-sm text-amber-800">
                You can solve problems for practice. Submissions during practice mode do not count
                toward the leaderboard.
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Problems List */}
      <div className="card mb-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-bold text-text-primary">Problems</h2>
          <span className="text-sm text-text-muted">{contest.problems?.length || 0} questions</span>
        </div>

        {!contest.problems || contest.problems.length === 0 ? (
          <div className="text-center py-10">
            <div className="text-4xl mb-3">📭</div>
            <p className="text-text-muted">No problems have been added to this contest yet.</p>
          </div>
        ) : (
          <div className="space-y-3">
            {contest.problems.map((problem, idx) => {
              const statusBadge = getProblemStatusBadge(problem);
              const isDisabled =
                (!isPracticeMode && hasFinished) ||
                (contestStatus.status !== 'active' && !contest.is_practice_active) ||
                (!isPracticeMode && problem.is_solved);

              return (
                <div
                  key={problem.problem_id}
                  className={`border rounded-lg p-4 transition-all ${
                    isDisabled
                      ? 'border-border-light bg-bg-secondary cursor-not-allowed opacity-70'
                      : 'border-border-light hover:border-accent-secondary/50 cursor-pointer group'
                  }`}
                  onClick={() => {
                    if (!isDisabled) {
                      navigate(`/contests/${id}/problems/${problem.problem_id}`);
                    }
                  }}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-4 flex-1">
                      {/* Problem Number */}
                      <span className="text-text-muted text-sm font-mono w-8 text-center">
                        #{idx + 1}
                      </span>

                      {/* Status Indicator */}
                      <span
                        className={`status-indicator ${statusBadge.class}`}
                        title={statusBadge.label}
                      >
                        {statusBadge.icon}
                      </span>

                      {/* Title */}
                      <span
                        className={`font-semibold ${
                          problem.is_solved
                            ? 'text-accent-success'
                            : 'text-text-primary group-hover:text-accent-secondary transition-colors'
                        }`}
                      >
                        {problem.title}
                      </span>

                      {/* Difficulty */}
                      {problem.difficulty && (
                        <span className={`difficulty ${getDifficultyClass(problem.difficulty)}`}>
                          {problem.difficulty}
                        </span>
                      )}

                      {/* Points */}
                      {problem.points && (
                        <span className="text-sm text-text-muted">{problem.points} pts</span>
                      )}
                    </div>

                    {/* Arrow, Lock, or Solved */}
                    <div className="text-text-muted">
                      {problem.is_solved ? (
                        <span className="text-accent-success">✓</span>
                      ) : isDisabled ? (
                        <svg
                          width="20"
                          height="20"
                          viewBox="0 0 24 24"
                          fill="none"
                          stroke="currentColor"
                          strokeWidth="2"
                        >
                          <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
                          <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
                        </svg>
                      ) : (
                        <span className="text-accent-secondary group-hover:translate-x-1 transition-transform">
                          →
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Finish Contest Button */}
      {!hasFinished && contestStatus.status === 'active' && !contest.is_practice_active && (
        <div className="flex justify-end">
          <button
            onClick={handleFinishContest}
            disabled={finishing}
            className="btn btn-success w-full md:w-auto"
          >
            {finishing ? (
              <>
                <span className="animate-spin rounded-full h-4 w-4 border-b-2 border-white inline-block mr-2"></span>
                Finishing...
              </>
            ) : (
              <>
                <svg
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  style={{ marginRight: '8px' }}
                >
                  <polyline points="20 6 9 17 4 12"></polyline>
                </svg>
                Finish Contest
              </>
            )}
          </button>
        </div>
      )}

      {/* Already Finished Message */}
      {hasFinished && !isPracticeMode && (
        <div className="card bg-accent-success/10 border-accent-success">
          <div className="flex items-center gap-3">
            <svg
              width="24"
              height="24"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              className="text-accent-success"
            >
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
              <polyline points="22 4 12 14.01 9 11.01"></polyline>
            </svg>
            <div>
              <h3 className="text-lg font-semibold text-accent-success">Contest Finished</h3>
              <p className="text-sm text-text-secondary">
                Your submissions have been finalized. No further edits allowed.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default ContestProblemsPage;
