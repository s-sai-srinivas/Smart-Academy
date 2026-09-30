import { useState, useEffect, useRef, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { contestsAPI } from '../../services/api';
import { showError } from '../../utils/showAlert';
import Breadcrumb from '../../components/common/Breadcrumb';

interface ContestItem {
  contest_id: number;
  title: string;
  description?: string;
  start_time: string;
  end_time: string;
  status?: string;
  target_cohort?: string;
  target_branch_name?: string;
  participant_count?: number;
  has_joined?: boolean;
  is_practice_active?: boolean;
  problems?: unknown[];
}

function ContestsPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [contests, setContests] = useState<ContestItem[]>([]);
  const [selectedContest, setSelectedContest] = useState<ContestItem | null>(null);
  const [loading, setLoading] = useState(true);
  const [joining, setJoining] = useState<number | null>(null);
  const [countdown, setCountdown] = useState('');
  const countdownRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const loadContests = useCallback(async () => {
    try {
      setLoading(true);
      const data = (await contestsAPI.getAll()) as { contests?: ContestItem[] } | ContestItem[];
      const list = Array.isArray(data) ? data : data.contests || [];
      setContests(list);
    } catch (err) {
      console.error('Failed to load contests:', err);
      showError('Failed to load contests');
    } finally {
      setLoading(false);
    }
  }, []);

  const loadContestDetails = useCallback(async (contestId: string) => {
    try {
      setLoading(true);
      const data = (await contestsAPI.getById(contestId)) as ContestItem;
      setSelectedContest(data);
    } catch (err) {
      console.error('Failed to load contest details:', err);
      showError('Failed to load contest details');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (id) {
      loadContestDetails(id);
    } else {
      setSelectedContest(null);
      loadContests();
    }
  }, [id, loadContestDetails, loadContests]);

  const handleJoinContest = async (contestId: number) => {
    try {
      setJoining(contestId);
      await contestsAPI.join(contestId);
      // Reload contest details — now has_joined will be true
      const data = (await contestsAPI.getById(String(contestId))) as ContestItem;
      setSelectedContest(data);

      // If contest is active, redirect to problems page
      const now = new Date();
      const start = new Date(data.start_time);
      const end = new Date(data.end_time);
      if (now >= start && now <= end) {
        // Contest is active - force fullscreen prompt on entry
        localStorage.removeItem(`contest_fullscreen_prompt_${contestId}`);
        navigate(`/contests/${contestId}/problems`);
      }
    } catch (err) {
      console.error('Failed to join contest:', err);
      showError((err as Error).message || 'Failed to join contest');
    } finally {
      setJoining(null);
    }
  };

  // Countdown ticker for upcoming contests
  useEffect(() => {
    if (countdownRef.current) clearInterval(countdownRef.current);
    if (!selectedContest) return;
    const start = new Date(selectedContest.start_time);
    const tick = () => {
      const diff = start.getTime() - new Date().getTime();
      if (diff <= 0) {
        setCountdown('');
        if (countdownRef.current) clearInterval(countdownRef.current);
        // Reload to show active state
        loadContestDetails(String(selectedContest.contest_id));
        return;
      }
      const h = Math.floor(diff / 3600000);
      const m = Math.floor((diff % 3600000) / 60000);
      const s = Math.floor((diff % 60000) / 1000);
      setCountdown(`${h}h ${String(m).padStart(2, '0')}m ${String(s).padStart(2, '0')}s`);
    };
    tick();
    countdownRef.current = setInterval(tick, 1000);
    return () => {
      if (countdownRef.current) clearInterval(countdownRef.current);
    };
  }, [selectedContest, loadContestDetails]);

  const getContestStatus = (contest: ContestItem) => {
    const now = new Date();
    const start = new Date(contest.start_time);
    const end = new Date(contest.end_time);

    if (now < start) return { status: 'upcoming', label: 'Upcoming', class: 'badge-info' };
    if (now > end) return { status: 'ended', label: 'Ended', class: 'badge-neutral' };
    return { status: 'active', label: 'Active', class: 'badge-success' };
  };

  const formatDateTime = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      hour12: true,
    });
  };

  const formatDuration = (start: string, end: string) => {
    const startDate = new Date(start);
    const endDate = new Date(end);
    const diffMs = endDate.getTime() - startDate.getTime();
    const hours = Math.floor(diffMs / (1000 * 60 * 60));
    const minutes = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60));
    return `${hours}h ${minutes}m`;
  };

  // Detail View
  if (id && selectedContest) {
    const contestStatus = getContestStatus(selectedContest);

    return (
      <div className="p-6 max-w-6xl mx-auto">
        {/* Breadcrumb Navigation */}
        <Breadcrumb
          items={[{ label: 'Contests', to: '/contests' }, { label: selectedContest.title }]}
        />

        {/* Contest Header */}
        <div className="card mb-6">
          <div className="flex items-start justify-between mb-4">
            <div className="flex-1">
              <div className="flex items-center gap-3 mb-2">
                <h1 className="text-3xl font-bold text-text-primary">{selectedContest.title}</h1>
                <span className={`badge ${contestStatus.class}`}>{contestStatus.label}</span>
                {selectedContest.is_practice_active && (
                  <span className="badge bg-accent-primary/20 text-accent-primary border border-accent-primary/30">
                    Practice Mode
                  </span>
                )}
              </div>
              {selectedContest.description && (
                <p className="text-text-secondary mb-4">{selectedContest.description}</p>
              )}
            </div>
          </div>

          {/* Contest Info */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4">
            <div className="flex flex-col">
              <span className="text-text-muted text-sm">Start Time</span>
              <span className="text-text-primary font-medium">
                {formatDateTime(selectedContest.start_time)}
              </span>
            </div>
            <div className="flex flex-col">
              <span className="text-text-muted text-sm">End Time</span>
              <span className="text-text-primary font-medium">
                {formatDateTime(selectedContest.end_time)}
              </span>
            </div>
            <div className="flex flex-col">
              <span className="text-text-muted text-sm">Duration</span>
              <span className="text-text-primary font-medium">
                {formatDuration(selectedContest.start_time, selectedContest.end_time)}
              </span>
            </div>
            <div className="flex flex-col">
              <span className="text-text-muted text-sm">Participants</span>
              <span className="text-text-primary font-medium">
                {selectedContest.participant_count || 0}
              </span>
            </div>
          </div>

          {/* Eligibility Info */}
          {selectedContest.target_cohort && (
            <div className="flex items-center gap-2 text-sm text-text-secondary mb-2">
              <span className="badge badge-neutral">Batch {selectedContest.target_cohort}</span>
              {selectedContest.target_branch_name && (
                <span className="badge badge-neutral">{selectedContest.target_branch_name}</span>
              )}
            </div>
          )}

          {/* Join / Status Button */}
          {contestStatus.status === 'ended' ? (
            selectedContest.is_practice_active ? (
              <div className="flex flex-col md:flex-row md:items-center gap-3">
                <span className="text-accent-primary font-medium">
                  Practice mode is active. You can solve contest problems now.
                </span>
                <button
                  onClick={() => navigate(`/contests/${selectedContest.contest_id}/problems`)}
                  className="btn btn-primary"
                >
                  Go to Practice Problems →
                </button>
              </div>
            ) : selectedContest.has_joined ? (
              <div className="text-text-muted">You participated in this contest</div>
            ) : (
              <div className="text-accent-danger">This contest has ended</div>
            )
          ) : !selectedContest.has_joined ? (
            <button
              onClick={() => handleJoinContest(selectedContest.contest_id)}
              disabled={joining === selectedContest.contest_id}
              className="btn btn-primary"
            >
              {joining === selectedContest.contest_id ? 'Joining...' : 'Join Contest'}
            </button>
          ) : contestStatus.status === 'upcoming' ? (
            <div className="flex flex-col gap-2">
              <div className="flex items-center gap-2">
                <span className="inline-block w-2 h-2 rounded-full bg-accent-warning animate-pulse"></span>
                <span className="text-accent-warning font-medium">Contest has not started yet</span>
              </div>
              <p className="text-text-secondary text-sm">
                You are registered! The contest starts on{' '}
                <span className="text-text-primary font-semibold">
                  {formatDateTime(selectedContest.start_time)}
                </span>
                .
              </p>
              {countdown && (
                <div className="inline-flex items-center gap-2 bg-background-tertiary border border-border-light rounded-lg px-4 py-2 w-fit">
                  <span className="text-text-muted text-sm">Starts in</span>
                  <span className="font-mono font-bold text-accent-primary text-lg">
                    {countdown}
                  </span>
                </div>
              )}
            </div>
          ) : (
            <div className="flex items-center gap-4">
              <span className="text-accent-success font-medium">✓ You are participating</span>
              <button
                onClick={() => navigate(`/contests/${selectedContest.contest_id}/problems`)}
                className="btn btn-primary"
              >
                Go to Problems →
              </button>
            </div>
          )}
        </div>

        {/* Quick access to problems page for active contests */}
        {selectedContest.has_joined && contestStatus.status === 'active' && (
          <div className="card mb-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-xl font-bold text-text-primary">Contest Problems</h2>
                <p className="text-text-secondary text-sm">
                  {selectedContest.problems?.length || 0} problems available
                </p>
              </div>
              <div className="flex gap-3">
                <button
                  onClick={() => navigate(`/contests/${selectedContest.contest_id}/leaderboard`)}
                  className="btn btn-secondary flex items-center gap-2"
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
                <button
                  onClick={() => navigate(`/contests/${selectedContest.contest_id}/problems`)}
                  className="btn btn-primary"
                >
                  View Problems →
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Ended contest - show summary */}
        {(selectedContest.has_joined || selectedContest.is_practice_active) &&
          contestStatus.status === 'ended' && (
            <div className="card mb-6">
              <div className="text-center py-6">
                <h2 className="text-xl font-bold text-text-primary mb-2">Contest Ended</h2>
                {selectedContest.is_practice_active ? (
                  <p className="text-text-secondary mb-4">
                    Practice mode is enabled. You can solve problems for learning; practice
                    submissions do not affect final rankings.
                  </p>
                ) : (
                  <p className="text-text-secondary mb-4">
                    This contest has ended. Check the leaderboard for final results.
                  </p>
                )}
                <div className="flex flex-wrap items-center justify-center gap-3">
                  <button
                    onClick={() => navigate(`/contests/${selectedContest.contest_id}/leaderboard`)}
                    className="btn btn-secondary"
                  >
                    View Leaderboard
                  </button>
                  {selectedContest.is_practice_active && (
                    <button
                      onClick={() => navigate(`/contests/${selectedContest.contest_id}/problems`)}
                      className="btn btn-primary"
                    >
                      View Practice Problems
                    </button>
                  )}
                </div>
              </div>
            </div>
          )}

        {/* No problems placeholder */}
        {selectedContest.has_joined &&
          contestStatus.status === 'active' &&
          (!selectedContest.problems || selectedContest.problems.length === 0) && (
            <div className="card mb-6 text-center py-10">
              <div className="text-4xl mb-3">📭</div>
              <p className="text-text-muted">No problems have been added to this contest yet.</p>
            </div>
          )}
      </div>
    );
  }

  // List View
  return (
    <div className="p-6 max-w-6xl mx-auto">
      <h1 className="text-3xl font-bold text-text-primary mb-6">Contests</h1>

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <p className="text-text-muted">Loading contests...</p>
        </div>
      ) : contests.length === 0 ? (
        <div className="card text-center py-16">
          <div className="text-6xl mb-4">🏆</div>
          <h3 className="text-xl font-semibold text-text-primary mb-2">No Contests Available</h3>
          <p className="text-text-secondary">
            Check back later for coding competitions and challenges.
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {contests.map((contest) => {
            const status = getContestStatus(contest);
            return (
              <div
                key={contest.contest_id}
                className="card hover:border-accent-secondary/50 transition-all cursor-pointer"
                onClick={() => navigate(`/contests/${contest.contest_id}`)}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-3 mb-2">
                      <h3 className="text-xl font-semibold text-text-primary hover:text-accent-secondary transition-colors">
                        {contest.title}
                      </h3>
                      <span className={`badge ${status.class}`}>{status.label}</span>
                      {contest.target_cohort && (
                        <span className="badge badge-neutral text-xs">
                          Batch {contest.target_cohort}
                        </span>
                      )}
                    </div>
                    {contest.description && (
                      <p className="text-text-secondary text-sm mb-3 line-clamp-2">
                        {contest.description}
                      </p>
                    )}
                    <div className="flex items-center gap-4 text-sm text-text-muted">
                      <span>Start: {formatDateTime(contest.start_time)}</span>
                      <span>Duration: {formatDuration(contest.start_time, contest.end_time)}</span>
                      {contest.participant_count !== undefined && (
                        <span>{contest.participant_count} participants</span>
                      )}
                    </div>
                  </div>
                  <div className="text-accent-secondary text-xl">→</div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

export default ContestsPage;
