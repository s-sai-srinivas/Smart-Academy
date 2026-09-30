import { useState, useEffect, useCallback } from 'react';
import { useParams, Link } from 'react-router-dom';
import { contestsAPI } from '../../services/api';
import { showError, showSuccess } from '../../utils/showAlert';

interface Contest {
  contest_id: number;
  title: string;
  start_time: string;
  end_time: string;
  is_public: boolean;
  participant_count?: number;
  problems?: Array<{ problem_id: number; title: string; difficulty?: string }>;
}

interface LeaderboardEntry {
  rank: number;
  name: string;
  user_regdno?: string;
  total_score: number;
  best_total_score?: number;
  problems_solved: number;
  problems_attempted: number;
  last_submit_time?: string;
}

interface NonParticipant {
  user_regdno: string;
  name: string;
  branch_name?: string;
  section_name?: string;
  cohort_year: number;
}

interface NonParticipantsData {
  target_cohort_years?: number[];
  target_cohort?: number;
  target_branch_names?: string[];
  target_branch_id?: number;
  total_eligible?: number;
  total_joined?: number;
  total_not_joined?: number;
  non_joined_students?: NonParticipant[];
}

function ContestDetailView() {
  const { id } = useParams<{ id: string }>();
  const numericId = Number(id);
  const [contest, setContest] = useState<Contest | null>(null);
  const [leaderboard, setLeaderboard] = useState<LeaderboardEntry[]>([]);
  const [leaderboardLoading, setLeaderboardLoading] = useState<boolean>(false);
  const [leaderboardLoaded, setLeaderboardLoaded] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(true);
  const [downloading, setDownloading] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<string>('problems');

  // Non-participants state
  const [nonParticipants, setNonParticipants] = useState<NonParticipantsData | null>(null);
  const [nonParticipantsLoading, setNonParticipantsLoading] = useState<boolean>(false);

  const loadContestDetails = useCallback(async () => {
    try {
      setLoading(true);
      const data = (await contestsAPI.getById(numericId)) as Contest;
      setContest(data);
    } catch (err) {
      console.error('Failed to load contest details:', err);
      showError('Failed to load contest details');
    } finally {
      setLoading(false);
    }
  }, [numericId]);

  const loadLeaderboard = useCallback(async () => {
    try {
      setLeaderboardLoading(true);
      const data = (await contestsAPI.getLeaderboard(numericId)) as LeaderboardEntry[];
      setLeaderboard(data || []);
      setLeaderboardLoaded(true);
    } catch (err: unknown) {
      console.error('Failed to load leaderboard:', err);
      const message = err instanceof Error ? err.message : 'Failed to load leaderboard';
      showError(message);
    } finally {
      setLeaderboardLoading(false);
    }
  }, [numericId]);

  const loadNonParticipants = useCallback(async () => {
    try {
      setNonParticipantsLoading(true);
      const data = (await contestsAPI.getNonParticipants(numericId)) as NonParticipantsData;
      setNonParticipants(data);
    } catch (err: unknown) {
      console.error('Failed to load non-participants:', err);
      const message = err instanceof Error ? err.message : 'Failed to load non-participants';
      showError(message);
    } finally {
      setNonParticipantsLoading(false);
    }
  }, [numericId]);

  useEffect(() => {
    loadContestDetails();
    setLeaderboard([]);
    setLeaderboardLoaded(false);
    setNonParticipants(null);
  }, [loadContestDetails, id]);

  useEffect(() => {
    if (activeTab === 'leaderboard' && !leaderboardLoaded) {
      loadLeaderboard();
    }
    if (activeTab === 'non-participants' && !nonParticipants) {
      loadNonParticipants();
    }
  }, [activeTab, leaderboardLoaded, nonParticipants, loadLeaderboard, loadNonParticipants]);

  const handleDownloadCSV = async (type: string) => {
    setDownloading(type);
    try {
      if (type === 'overview') {
        await contestsAPI.downloadResultsCSV(numericId);
      } else {
        await contestsAPI.downloadDetailedCSV(numericId);
      }
      showSuccess('CSV downloaded successfully!');
    } catch (err: unknown) {
      console.error('Failed to download CSV:', err);
      const message = err instanceof Error ? err.message : 'Failed to download CSV';
      showError(message);
    } finally {
      setDownloading(null);
    }
  };

  const getContestStatus = () => {
    if (!contest) return { status: 'unknown', label: 'Unknown', class: 'badge-neutral' };
    const now = new Date();
    const start = new Date(contest.start_time);
    const end = new Date(contest.end_time);

    if (now < start) return { status: 'upcoming', label: 'Upcoming', class: 'badge-info' };
    if (now > end) return { status: 'ended', label: 'Ended', class: 'badge-neutral' };
    return { status: 'active', label: 'Active', class: 'badge-success' };
  };

  const formatDateTime = (dateStr: string) => {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      hour12: true,
    });
  };

  const formatDuration = () => {
    if (!contest) return '-';
    const start = new Date(contest.start_time);
    const end = new Date(contest.end_time);
    const diffMs = end.getTime() - start.getTime();
    const hours = Math.floor(diffMs / (1000 * 60 * 60));
    const minutes = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60));
    return `${hours}h ${minutes}m`;
  };

  const formatLastSubmission = (dateStr: string | undefined) => {
    if (!dateStr) return '-';
    return new Date(dateStr).toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: true,
    });
  };

  const copyAllRegdNos = () => {
    if (!nonParticipants?.non_joined_students?.length) return;
    const allRegdNos = nonParticipants.non_joined_students
      .map((student) => student.user_regdno)
      .filter(Boolean)
      .join('\n');
    navigator.clipboard
      .writeText(allRegdNos)
      .then(() => showSuccess('All register numbers copied to clipboard!'))
      .catch(() => showError('Failed to copy register numbers'));
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <p className="text-text-muted">Loading contest details...</p>
      </div>
    );
  }

  if (!contest) {
    return (
      <div className="card text-center py-16">
        <h3 className="text-xl font-semibold text-text-primary mb-2">Contest Not Found</h3>
        <p className="text-text-secondary mb-6">The requested contest could not be found.</p>
        <Link to=".." relative="path" className="btn btn-primary">
          Back to Contests
        </Link>
      </div>
    );
  }

  const status = getContestStatus();
  const isEnded = status.status === 'ended';

  return (
    <div className="p-6 max-w-6xl mx-auto">
      <div className="flex items-center gap-3 mb-6">
        <Link to=".." relative="path" className="text-text-muted hover:text-text-primary">
          ← Back to Contests
        </Link>
      </div>

      <div className="card mb-6">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="text-3xl font-bold text-text-primary">{contest.title}</h1>
            <span className={`badge ${status.class}`}>{status.label}</span>
            {!contest.is_public && <span className="badge badge-neutral">Private</span>}
          </div>
          <div className="flex flex-wrap items-center gap-4 text-sm text-text-secondary">
            <span className="inline-flex items-center gap-2">
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                aria-hidden="true"
              >
                <rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect>
                <line x1="16" y1="2" x2="16" y2="6"></line>
                <line x1="8" y1="2" x2="8" y2="6"></line>
                <line x1="3" y1="10" x2="21" y2="10"></line>
              </svg>
              {formatDateTime(contest.start_time)}
            </span>
            <span className="inline-flex items-center gap-2">
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                aria-hidden="true"
              >
                <rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect>
                <line x1="16" y1="2" x2="16" y2="6"></line>
                <line x1="8" y1="2" x2="8" y2="6"></line>
                <line x1="3" y1="10" x2="21" y2="10"></line>
              </svg>
              {formatDateTime(contest.end_time)}
            </span>
            <span className="inline-flex items-center gap-2">
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                aria-hidden="true"
              >
                <circle cx="12" cy="12" r="10"></circle>
                <polyline points="12 6 12 12 16 14"></polyline>
              </svg>
              {formatDuration()}
            </span>
            <span className="inline-flex items-center gap-2">
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                aria-hidden="true"
              >
                <path d="M20 21a8 8 0 1 0-16 0"></path>
                <circle cx="12" cy="7" r="4"></circle>
              </svg>
              {contest.participant_count || 0} participants
            </span>
          </div>
        </div>

        {isEnded && (
          <div className="mt-4 flex flex-wrap items-center gap-2">
            <button
              onClick={() => handleDownloadCSV('overview')}
              disabled={downloading === 'overview'}
              className="btn btn-primary"
            >
              {downloading === 'overview' ? 'Downloading...' : 'Download Overview CSV'}
            </button>
            <button
              onClick={() => handleDownloadCSV('detailed')}
              disabled={downloading === 'detailed'}
              className="btn btn-secondary"
            >
              {downloading === 'detailed' ? 'Downloading...' : 'Download Detailed CSV'}
            </button>
            <Link to="plagiarism" relative="path" className="btn btn-secondary">
              View Plagiarism
            </Link>
          </div>
        )}
      </div>

      <div className="flex border-b border-border-light mb-6">
        <button
          onClick={() => setActiveTab('problems')}
          className={`px-4 py-2 font-medium transition-colors ${
            activeTab === 'problems'
              ? 'text-accent-secondary border-b-2 border-accent-secondary'
              : 'text-text-muted hover:text-text-primary'
          }`}
        >
          Problems ({contest.problems?.length || 0})
        </button>
        <button
          onClick={() => setActiveTab('leaderboard')}
          className={`px-4 py-2 font-medium transition-colors ${
            activeTab === 'leaderboard'
              ? 'text-accent-secondary border-b-2 border-accent-secondary'
              : 'text-text-muted hover:text-text-primary'
          }`}
        >
          Leaderboard
        </button>
        <button
          onClick={() => setActiveTab('non-participants')}
          className={`px-4 py-2 font-medium transition-colors ${
            activeTab === 'non-participants'
              ? 'text-accent-secondary border-b-2 border-accent-secondary'
              : 'text-text-muted hover:text-text-primary'
          }`}
        >
          Non-Joined Students
        </button>
      </div>

      {activeTab === 'problems' && (
        <div className="space-y-4">
          {!contest.problems || contest.problems.length === 0 ? (
            <div className="card text-center py-8">
              <p className="text-text-muted">No problems added to this contest yet.</p>
            </div>
          ) : (
            contest.problems.map((problem, index) => (
              <div key={problem.problem_id} className="card">
                <div className="flex items-center justify-between gap-4 py-1">
                  <div className="flex min-w-0 items-center gap-3">
                    <span className="text-base font-bold text-accent-secondary">
                      {String.fromCharCode(65 + index)}
                    </span>
                    <h3 className="truncate text-base font-semibold text-text-primary">
                      {problem.title}
                    </h3>
                  </div>
                  <span className="text-sm font-medium capitalize text-text-muted whitespace-nowrap">
                    {problem.difficulty || 'unknown'}
                  </span>
                </div>
              </div>
            ))
          )}
        </div>
      )}

      {activeTab === 'leaderboard' && (
        <div>
          {leaderboardLoading ? (
            <div className="card text-center py-10">
              <p className="text-text-muted">Loading leaderboard...</p>
            </div>
          ) : leaderboard.length === 0 ? (
            <div className="card text-center py-10">
              <p className="text-text-muted">No submissions yet.</p>
            </div>
          ) : (
            <div className="card overflow-x-auto" style={{ padding: 0 }}>
              <table className="w-full">
                <thead>
                  <tr className="border-b border-border-light">
                    <th className="text-left py-3 px-4 text-sm text-text-muted font-semibold">
                      Rank
                    </th>
                    <th className="text-left py-3 px-4 text-sm text-text-muted font-semibold">
                      Participant
                    </th>
                    <th className="text-right py-3 px-4 text-sm text-text-muted font-semibold">
                      Score
                    </th>
                    <th className="text-right py-3 px-4 text-sm text-text-muted font-semibold">
                      Solved / Attempted
                    </th>
                    <th className="text-right py-3 px-4 text-sm text-text-muted font-semibold">
                      Last Submission
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {leaderboard.map((entry, idx) => (
                    <tr key={entry.user_regdno || idx} className="border-b border-border-light/60">
                      <td className="py-3 px-4 text-text-primary font-semibold">#{entry.rank}</td>
                      <td className="py-3 px-4 text-text-primary">
                        {entry.name}
                        {entry.user_regdno && (
                          <span className="ml-2 text-xs text-text-muted">
                            ({entry.user_regdno})
                          </span>
                        )}
                      </td>
                      <td className="py-3 px-4 text-right text-text-primary font-semibold">
                        {entry.total_score}
                        {entry.best_total_score && entry.best_total_score > entry.total_score && (
                          <span className="text-text-muted text-xs ml-1">
                            ({entry.best_total_score} partial)
                          </span>
                        )}
                      </td>
                      <td className="py-3 px-4 text-right text-text-primary">
                        {entry.problems_solved}
                        <span className="text-text-muted"> / {entry.problems_attempted}</span>
                      </td>
                      <td className="py-3 px-4 text-right text-text-muted">
                        {formatLastSubmission(entry.last_submit_time)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {activeTab === 'non-participants' && (
        <div>
          {nonParticipantsLoading ? (
            <div className="card text-center py-10">
              <p className="text-text-muted">Loading student data...</p>
            </div>
          ) : !nonParticipants ? (
            <div className="card text-center py-10">
              <p className="text-text-muted">Failed to load data.</p>
            </div>
          ) : // Check both new array format and deprecated single field
          !nonParticipants.target_cohort_years?.length && !nonParticipants.target_cohort ? (
            <div className="card text-center py-10">
              <p className="text-text-muted">
                This contest is open to all students (no specific batch/branch targeting).
              </p>
              <p className="text-text-secondary text-sm mt-2">
                Non-joined students cannot be determined for open contests.
              </p>
            </div>
          ) : (
            <div>
              {/* Target Info */}
              {(nonParticipants.target_cohort_years?.length || nonParticipants.target_cohort) && (
                <div className="card mb-4">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-sm text-text-muted">Target Batches:</span>
                    {nonParticipants.target_cohort_years?.length
                      ? nonParticipants.target_cohort_years.map((year) => (
                          <span key={year} className="badge badge-neutral">
                            Batch {year}
                          </span>
                        ))
                      : nonParticipants.target_cohort && (
                          <span className="badge badge-neutral">
                            Batch {nonParticipants.target_cohort}
                          </span>
                        )}
                    {(nonParticipants.target_branch_names?.length ||
                      nonParticipants.target_branch_id) && (
                      <>
                        <span className="text-sm text-text-muted ml-2">Target Branches:</span>
                        {nonParticipants.target_branch_names?.length ? (
                          nonParticipants.target_branch_names.map((name) => (
                            <span key={name} className="badge badge-neutral">
                              {name}
                            </span>
                          ))
                        ) : (
                          <span className="badge badge-neutral">Specific Branch</span>
                        )}
                      </>
                    )}
                  </div>
                </div>
              )}

              {/* Summary Stats */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
                <div className="card text-center">
                  <p className="text-3xl font-bold text-text-primary">
                    {nonParticipants.total_eligible || 0}
                  </p>
                  <p className="text-sm text-text-muted">Eligible Students</p>
                </div>
                <div className="card text-center">
                  <p className="text-3xl font-bold text-accent-success">
                    {nonParticipants.total_joined || 0}
                  </p>
                  <p className="text-sm text-text-muted">Joined</p>
                </div>
                <div className="card text-center">
                  <p className="text-3xl font-bold text-accent-warning">
                    {nonParticipants.total_not_joined || 0}
                  </p>
                  <p className="text-sm text-text-muted">Not Joined</p>
                </div>
              </div>

              {/* Non-Joined Students List */}
              {nonParticipants.total_not_joined === 0 ? (
                <div className="card text-center py-10">
                  <p className="text-accent-success font-medium">
                    All eligible students have joined the contest!
                  </p>
                </div>
              ) : (
                <div className="card overflow-x-auto" style={{ padding: 0 }}>
                  <table className="w-full">
                    <thead>
                      <tr className="border-b border-border-light">
                        <th className="text-left py-3 px-4 text-sm text-text-muted font-semibold">
                          <div className="flex items-center gap-2">
                            Register No.
                            <button
                              onClick={copyAllRegdNos}
                              className="inline-flex items-center gap-1 text-xs px-2 py-1 rounded bg-background hover:bg-background-light text-text-muted hover:text-text-primary transition-colors"
                              title="Copy all register numbers"
                            >
                              <svg
                                width="14"
                                height="14"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                strokeWidth="2"
                                strokeLinecap="round"
                                strokeLinejoin="round"
                              >
                                <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
                              </svg>
                              Copy All
                            </button>
                          </div>
                        </th>
                        <th className="text-left py-3 px-4 text-sm text-text-muted font-semibold">
                          Name
                        </th>
                        <th className="text-left py-3 px-4 text-sm text-text-muted font-semibold">
                          Branch
                        </th>
                        <th className="text-left py-3 px-4 text-sm text-text-muted font-semibold">
                          Section
                        </th>
                        <th className="text-left py-3 px-4 text-sm text-text-muted font-semibold">
                          Batch
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {nonParticipants.non_joined_students?.map((student, idx) => (
                        <tr
                          key={student.user_regdno || idx}
                          className="border-b border-border-light/60"
                        >
                          <td className="py-3 px-4 text-text-primary font-mono">
                            {student.user_regdno}
                          </td>
                          <td className="py-3 px-4 text-text-primary">{student.name}</td>
                          <td className="py-3 px-4 text-text-secondary">
                            {student.branch_name || '-'}
                          </td>
                          <td className="py-3 px-4 text-text-secondary">
                            {student.section_name || '-'}
                          </td>
                          <td className="py-3 px-4 text-text-secondary">
                            Batch {student.cohort_year}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default ContestDetailView;
