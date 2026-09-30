import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { contestsAPI, referenceAPI } from '../../services/api';
import { showError, showSuccess } from '../../utils/showAlert';

interface Contest {
  contest_id: number | string;
  title: string;
  description?: string;
  start_time: string;
  end_time: string;
  is_public: boolean;
  participant_count?: number;
  target_cohort?: string | number;
  target_branch_id?: number | string;
}

interface Branch {
  branch_id: number | string;
  branch_name: string;
}

interface ContestStatus {
  status: string;
  label: string;
  class: string;
}

function HODContests() {
  const [contests, setContests] = useState<Contest[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [downloading, setDownloading] = useState<string | null>(null);

  // Reference data
  const [branches, setBranches] = useState<Branch[]>([]);

  useEffect(() => {
    loadContests();
    loadReferenceData();
  }, []);

  const loadContests = async () => {
    try {
      setLoading(true);
      // Always show all college contests (college-level)
      const data = (await contestsAPI.getHODContests()) as Contest[];
      setContests(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error('Failed to load contests:', err);
      showError('Failed to load contests');
    } finally {
      setLoading(false);
    }
  };

  const loadReferenceData = async () => {
    try {
      const branchesData = (await referenceAPI.getBranches()) as Branch[];
      setBranches(branchesData || []);
    } catch (err) {
      console.error('Failed to load reference data:', err);
    }
  };

  const handleDownloadCSV = async (contestId: number | string, type: string) => {
    setDownloading(String(contestId) + type);
    try {
      if (type === 'overview') {
        await contestsAPI.downloadResultsCSV(contestId);
      } else {
        await contestsAPI.downloadDetailedCSV(contestId);
      }
      showSuccess('CSV downloaded successfully!');
    } catch (err) {
      console.error('Failed to download CSV:', err);
      const error = err as Error;
      showError(error.message || 'Failed to download CSV');
    } finally {
      setDownloading(null);
    }
  };

  const getContestStatus = (contest: Contest): ContestStatus => {
    const now = new Date();
    const start = new Date(contest.start_time);
    const end = new Date(contest.end_time);

    if (now < start) return { status: 'upcoming', label: 'Upcoming', class: 'badge-info' };
    if (now > end) return { status: 'ended', label: 'Ended', class: 'badge-neutral' };
    return { status: 'active', label: 'Active', class: 'badge-success' };
  };

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

  const getBatchName = (cohort?: string | number): string => {
    if (!cohort) return 'All Batches';
    return `Batch ${cohort}`;
  };

  const getBranchName = (branchId?: number | string): string => {
    if (!branchId) return 'All Branches';
    const branch = branches.find((b) => b.branch_id === branchId);
    return branch?.branch_name || 'Unknown';
  };

  return (
    <div className="p-6 max-w-6xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-3xl font-bold text-text-primary">Contest Overview</h1>
          <p className="text-text-secondary text-sm mt-1">View all contests for your college</p>
        </div>
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <p className="text-text-muted">Loading contests...</p>
        </div>
      ) : contests.length === 0 ? (
        <div className="card text-center py-16">
          <div className="text-6xl mb-4">🏆</div>
          <h3 className="text-xl font-semibold text-text-primary mb-2">No Contests Found</h3>
          <p className="text-text-secondary mb-6">
            No contests have been created for your college yet.
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {contests.map((contest) => {
            const status = getContestStatus(contest);
            const isEnded = status.status === 'ended';
            const isLoading =
              downloading === String(contest.contest_id) + 'overview' ||
              downloading === String(contest.contest_id) + 'detailed';

            return (
              <div key={contest.contest_id} className="card">
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-3 mb-2">
                      <h3 className="text-xl font-semibold text-text-primary">{contest.title}</h3>
                      <span className={`badge ${status.class}`}>{status.label}</span>
                      {!contest.is_public && <span className="badge badge-neutral">Private</span>}
                    </div>
                    {contest.description && (
                      <p className="text-text-secondary text-sm mb-3">{contest.description}</p>
                    )}
                    <div className="flex items-center gap-4 text-sm text-text-muted">
                      <span>Start: {formatDateTime(contest.start_time)}</span>
                      <span>End: {formatDateTime(contest.end_time)}</span>
                      <span>Duration: {formatDuration(contest.start_time, contest.end_time)}</span>
                      {contest.participant_count !== undefined && (
                        <span>{contest.participant_count} participants</span>
                      )}
                    </div>
                    <div className="flex items-center gap-4 text-sm text-text-secondary mt-2">
                      <span>Target: {getBatchName(contest.target_cohort)}</span>
                      {contest.target_cohort && (
                        <span>Branch: {getBranchName(contest.target_branch_id)}</span>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {isEnded && (
                      <>
                        <button
                          onClick={() => handleDownloadCSV(contest.contest_id, 'overview')}
                          disabled={isLoading}
                          className="px-3 py-1 text-sm bg-accent-success text-white rounded hover:bg-accent-success/80 transition-colors disabled:opacity-50"
                          title="Download overview CSV with ranks, scores, and participation"
                        >
                          {downloading === String(contest.contest_id) + 'overview'
                            ? 'Downloading...'
                            : 'Overview CSV'}
                        </button>
                        <button
                          onClick={() => handleDownloadCSV(contest.contest_id, 'detailed')}
                          disabled={isLoading}
                          className="px-3 py-1 text-sm bg-accent-info text-white rounded hover:bg-accent-info/80 transition-colors disabled:opacity-50"
                          title="Download detailed per-problem CSV"
                        >
                          {downloading === String(contest.contest_id) + 'detailed'
                            ? 'Downloading...'
                            : 'Detailed CSV'}
                        </button>
                        <Link
                          to={`/hod/contests/${contest.contest_id}/plagiarism`}
                          className="px-3 py-1 text-sm border border-accent-warning text-accent-warning rounded hover:bg-accent-warning/10 transition-colors"
                          title="View plagiarism analysis"
                        >
                          Plagiarism
                        </Link>
                      </>
                    )}
                    <Link
                      to={`/hod/contests/${contest.contest_id}`}
                      className="px-3 py-1 text-sm border border-border-light rounded hover:bg-bg-secondary transition-colors"
                    >
                      View Details
                    </Link>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

export default HODContests;
