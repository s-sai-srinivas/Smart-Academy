import { useState, useEffect } from 'react';
import { principalAnalyticsAPI } from '../../../services/api';
import { showError } from '../../../utils/showAlert';
import { Trophy, Users, Target } from 'lucide-react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from 'recharts';

interface ContestActivity {
  total: number;
  internal: number;
  external: number;
  avg_participants: number;
}

interface BranchParticipation {
  branch_name: string;
  participation_pct: number;
  avg_score: number;
}

interface TopPerformer {
  name: string;
  branch_name: string;
  contests: number;
  avg_score: number;
  problems_solved: number;
}

interface EngagementStats {
  active_students: number;
  inactive_students: number;
  repeat_participants: number;
}

const ContestAnalytics = () => {
  const [loading, setLoading] = useState<boolean>(true);
  const [activity, setActivity] = useState<ContestActivity | null>(null);
  const [branchParticipation, setBranchParticipation] = useState<BranchParticipation[]>([]);
  const [topPerformers, setTopPerformers] = useState<TopPerformer[]>([]);
  const [engagement, setEngagement] = useState<EngagementStats | null>(null);

  useEffect(() => {
    fetchContestAnalytics();
  }, []);

  const fetchContestAnalytics = async () => {
    try {
      const response = (await principalAnalyticsAPI.getContestAnalytics()) as {
        contest_activity: ContestActivity;
        branch_participation: BranchParticipation[];
        top_performers: TopPerformer[];
        engagement_stats: EngagementStats;
      };
      setActivity(response.contest_activity);
      setBranchParticipation(response.branch_participation);
      setTopPerformers(response.top_performers);
      setEngagement(response.engagement_stats);
    } catch (error) {
      console.error('Error fetching contest analytics:', error);
      showError('Failed to load contest analytics');
    } finally {
      setLoading(false);
    }
  };

  const COLORS = ['#10b981', '#3b82f6', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899'];

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="flex items-center gap-3">
          <svg
            className="animate-spin h-5 w-5 text-accent-primary"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              className="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="4"
            ></circle>
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            ></path>
          </svg>
          <span className="text-text-secondary">Loading contest analytics...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-text-primary mb-1">Contest Analytics</h1>
        <p className="text-text-secondary text-sm">
          Coding contest activity, participation, and performance
        </p>
      </div>

      {/* Activity Cards */}
      {activity && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
          <div className="card-elevated flex items-center gap-4 border-l-4 border-accent-secondary">
            <div className="flex-1">
              <div className="text-3xl font-bold text-text-primary">{activity.total}</div>
              <div className="text-sm text-text-secondary mt-1">Total Contests</div>
            </div>
            <Trophy className="w-8 h-8 text-text-muted opacity-50" />
          </div>
          <div className="card-elevated flex items-center gap-4 border-l-4 border-accent-success">
            <div className="flex-1">
              <div className="text-3xl font-bold text-text-primary">{activity.internal}</div>
              <div className="text-sm text-text-secondary mt-1">Internal Contests</div>
            </div>
            <Target className="w-8 h-8 text-text-muted opacity-50" />
          </div>
          <div className="card-elevated flex items-center gap-4 border-l-4 border-accent-warning">
            <div className="flex-1">
              <div className="text-3xl font-bold text-text-primary">
                {activity.avg_participants.toFixed(1)}
              </div>
              <div className="text-sm text-text-secondary mt-1">Avg Participants</div>
            </div>
            <Users className="w-8 h-8 text-text-muted opacity-50" />
          </div>
          <div className="card-elevated flex items-center gap-4 border-l-4 border-accent-info">
            <div className="flex-1">
              <div className="text-3xl font-bold text-text-primary">
                {engagement?.repeat_participants || 0}
              </div>
              <div className="text-sm text-text-secondary mt-1">Repeat Participants</div>
            </div>
            <Users className="w-8 h-8 text-text-muted opacity-50" />
          </div>
        </div>
      )}

      {/* Branch Participation */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="card">
          <h2 className="text-xl font-semibold text-text-primary mb-4">Branch Participation</h2>
          <div className="h-64">
            {branchParticipation.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={branchParticipation} layout="vertical">
                  <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.1)" />
                  <XAxis type="number" stroke="rgba(255,255,255,0.5)" />
                  <YAxis
                    dataKey="branch_name"
                    type="category"
                    stroke="rgba(255,255,255,0.5)"
                    width={80}
                  />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: '#1f2937',
                      border: '1px solid #374151',
                      borderRadius: '8px',
                    }}
                  />
                  <Bar dataKey="participation_pct" radius={[0, 4, 4, 0]}>
                    {branchParticipation.map((_, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex items-center justify-center h-full text-text-muted">
                No participation data available
              </div>
            )}
          </div>
        </div>

        <div className="card">
          <h2 className="text-xl font-semibold text-text-primary mb-4">Avg Score by Branch</h2>
          <div className="h-64">
            {branchParticipation.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={branchParticipation}>
                  <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.1)" />
                  <XAxis dataKey="branch_name" stroke="rgba(255,255,255,0.5)" />
                  <YAxis stroke="rgba(255,255,255,0.5)" />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: '#1f2937',
                      border: '1px solid #374151',
                      borderRadius: '8px',
                    }}
                  />
                  <Bar dataKey="avg_score" fill="#3b82f6" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex items-center justify-center h-full text-text-muted">
                No score data available
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Top Performers */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
          <Trophy className="w-5 h-5 text-accent-warning" />
          Top Performers
        </h2>
        <div className="table-container overflow-x-auto">
          <table className="table w-full">
            <thead>
              <tr>
                <th>Rank</th>
                <th>Student</th>
                <th>Branch</th>
                <th>Contests</th>
                <th>Avg Score</th>
                <th>Problems Solved</th>
              </tr>
            </thead>
            <tbody>
              {topPerformers.map((performer, index) => (
                <tr key={index}>
                  <td className="text-text-secondary">#{index + 1}</td>
                  <td className="text-text-primary font-medium">{performer.name}</td>
                  <td className="text-text-secondary">{performer.branch_name}</td>
                  <td className="text-text-primary">{performer.contests}</td>
                  <td className="text-accent-success">{performer.avg_score.toFixed(1)}</td>
                  <td className="text-text-primary">{performer.problems_solved}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

export default ContestAnalytics;
