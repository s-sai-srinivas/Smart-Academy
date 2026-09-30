import { useState, useEffect, useCallback } from 'react';
import { useParams, Link } from 'react-router-dom';
import { contestsAPI } from '../../services/api';
import { showError } from '../../utils/showAlert';
import Breadcrumb from '../../components/common/Breadcrumb';
import { useContestMode } from '../../context/ContestModeContext';

interface ContestData {
  title: string;
  is_frozen?: boolean;
  participant_count?: number;
}

interface LeaderboardEntry {
  rank: number;
  user_regdno?: string;
  name: string;
  total_score: number;
  best_total_score?: number;
  problems_solved: number;
  problems_attempted: number;
  last_submit_time?: string;
}

function ContestLeaderboardPage() {
  const { id } = useParams<{ id: string }>();
  const [contest, setContest] = useState<ContestData | null>(null);
  const [leaderboard, setLeaderboard] = useState<LeaderboardEntry[]>([]);
  const [loading, setLoading] = useState(true);

  // Contest mode context
  const { isInContestMode, activeContestId } = useContestMode();

  const loadData = useCallback(async () => {
    if (!id) return;
    try {
      setLoading(true);
      const [contestData, leaderboardData] = await Promise.all([
        contestsAPI.getById(id) as Promise<ContestData>,
        contestsAPI.getLeaderboard(id) as Promise<LeaderboardEntry[]>,
      ]);
      setContest(contestData);
      setLeaderboard(leaderboardData || []);
    } catch (err) {
      console.error('Failed to load leaderboard:', err);
      showError((err as Error).message || 'Failed to load leaderboard');
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const getRankBadge = (rank: number) => {
    if (rank === 1) return { emoji: '🥇', color: '#ffd700' };
    if (rank === 2) return { emoji: '🥈', color: '#c0c0c0' };
    if (rank === 3) return { emoji: '🥉', color: '#cd7f32' };
    return null;
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <p className="text-text-muted">Loading leaderboard...</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background-primary">
      {/* Header */}
      <div className="border-b border-border-light bg-background-secondary">
        <div className="max-w-5xl mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div>
              {/* In contest mode, show limited breadcrumb */}
              {isInContestMode && activeContestId ? (
                <Breadcrumb
                  items={[
                    { label: contest?.title || 'Contest', to: `/contests/${id}/problems` },
                    { label: 'Leaderboard' },
                  ]}
                />
              ) : (
                <Breadcrumb
                  items={[
                    { label: 'Dashboard', to: '/dashboard' },
                    { label: 'Contests', to: '/contests' },
                    { label: contest?.title || 'Contest', to: `/contests/${id}` },
                    { label: 'Leaderboard' },
                  ]}
                />
              )}
              <h1 className="text-2xl font-bold text-text-primary flex items-center gap-3">
                <svg
                  width="24"
                  height="24"
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
                {contest?.is_frozen && <span className="badge badge-warning text-sm">Frozen</span>}
              </h1>
              <p className="text-text-secondary text-sm mt-1">{contest?.title}</p>
            </div>
            <div className="flex items-center gap-4">
              {/* Back to Problems button when in contest mode */}
              {isInContestMode && activeContestId && (
                <Link
                  to={`/contests/${activeContestId}/problems`}
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
                    <line x1="19" y1="12" x2="5" y2="12"></line>
                    <polyline points="12 19 5 12 12 5"></polyline>
                  </svg>
                  Back to Problems
                </Link>
              )}
              <div className="text-right text-sm text-text-muted">
                <div>{contest?.participant_count || leaderboard.length} participants</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Leaderboard Table */}
      <div className="max-w-5xl mx-auto px-6 py-6">
        {leaderboard.length === 0 ? (
          <div className="card text-center py-16">
            <div className="text-6xl mb-4">📊</div>
            <h3 className="text-xl font-semibold text-text-primary mb-2">No Submissions Yet</h3>
            <p className="text-text-secondary">
              Be the first to solve a problem and claim the top spot!
            </p>
          </div>
        ) : (
          <div className="card overflow-hidden" style={{ padding: 0 }}>
            <table className="w-full">
              <thead>
                <tr style={{ background: 'var(--background-tertiary, #1e1e3f)' }}>
                  <th
                    className="text-left py-4 px-6 text-text-muted font-semibold text-sm"
                    style={{ width: '80px' }}
                  >
                    Rank
                  </th>
                  <th className="text-left py-4 px-6 text-text-muted font-semibold text-sm">
                    Participant
                  </th>
                  <th
                    className="text-right py-4 px-6 text-text-muted font-semibold text-sm"
                    style={{ width: '120px' }}
                  >
                    Score
                  </th>
                  <th
                    className="text-right py-4 px-6 text-text-muted font-semibold text-sm"
                    style={{ width: '140px' }}
                  >
                    Solved / Attempted
                  </th>
                  <th
                    className="text-right py-4 px-6 text-text-muted font-semibold text-sm"
                    style={{ width: '160px' }}
                  >
                    Last Submission
                  </th>
                </tr>
              </thead>
              <tbody>
                {leaderboard.map((entry, idx) => {
                  const badge = getRankBadge(entry.rank);
                  return (
                    <tr
                      key={entry.user_regdno || idx}
                      className="border-b border-border-light hover:bg-background-tertiary/30 transition-colors"
                      style={
                        entry.rank <= 3
                          ? { background: 'var(--background-tertiary, #1e1e3f)', opacity: 0.95 }
                          : {}
                      }
                    >
                      <td className="py-4 px-6">
                        <div className="flex items-center gap-2">
                          {badge ? (
                            <span style={{ fontSize: '1.25rem' }}>{badge.emoji}</span>
                          ) : (
                            <span className="text-text-muted font-mono font-bold">
                              #{entry.rank}
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="py-4 px-6">
                        <div>
                          <span
                            className={`font-semibold ${entry.rank <= 3 ? 'text-accent-primary' : 'text-text-primary'}`}
                          >
                            {entry.name}
                          </span>
                          {entry.user_regdno && (
                            <span className="text-text-muted text-xs ml-2">
                              {entry.user_regdno}
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="py-4 px-6 text-right">
                        <span className="font-bold text-accent-secondary text-lg">
                          {entry.total_score}
                        </span>
                        {entry.best_total_score && entry.best_total_score > entry.total_score && (
                          <span className="text-text-muted text-xs ml-1">
                            ({entry.best_total_score} partial)
                          </span>
                        )}
                      </td>
                      <td className="py-4 px-6 text-right">
                        <span className="text-text-primary font-medium">
                          {entry.problems_solved}
                        </span>
                        <span className="text-text-muted"> / {entry.problems_attempted}</span>
                      </td>
                      <td className="py-4 px-6 text-right text-text-muted text-sm">
                        {entry.last_submit_time
                          ? new Date(entry.last_submit_time).toLocaleTimeString('en-US', {
                              hour: '2-digit',
                              minute: '2-digit',
                              second: '2-digit',
                              hour12: true,
                            })
                          : '—'}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

export default ContestLeaderboardPage;
