import { useState, useEffect } from 'react';
import { principalAnalyticsAPI } from '../../../services/api';
import { showError } from '../../../utils/showAlert';
import { GitBranch, Trophy, AlertTriangle, TrendingUp } from 'lucide-react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  LineChart,
  Line,
  RadarChart,
  PolarGrid,
  PolarAngleAxis,
  PolarRadiusAxis,
  Radar,
  Legend,
} from 'recharts';

interface BranchData {
  branch_id: number;
  branch_name: string;
  short_name: string;
  total_students: number;
  total_faculty: number;
  avg_submission_score: number;
  pass_pct: number;
  contest_participation: number;
  assignment_completion: number;
}

interface BranchTrend {
  branch_name: string;
  year: number;
  avg_marks: number;
}

const BranchAnalytics = () => {
  const [loading, setLoading] = useState<boolean>(true);
  const [branches, setBranches] = useState<BranchData[]>([]);
  const [bestBranch, setBestBranch] = useState<string>('');
  const [weakestBranch, setWeakestBranch] = useState<string>('');
  const [trends, setTrends] = useState<BranchTrend[]>([]);

  useEffect(() => {
    fetchBranchAnalytics();
  }, []);

  const fetchBranchAnalytics = async () => {
    try {
      const response = (await principalAnalyticsAPI.getBranchAnalytics()) as {
        branches: BranchData[];
        best_branch: string;
        weakest_branch: string;
        branch_trends: BranchTrend[];
      };
      setBranches(response.branches);
      setBestBranch(response.best_branch);
      setWeakestBranch(response.weakest_branch);
      setTrends(response.branch_trends);
    } catch (error) {
      console.error('Error fetching branch analytics:', error);
      showError('Failed to load branch analytics');
    } finally {
      setLoading(false);
    }
  };

  const getPerformanceColor = (value: number) => {
    if (value >= 70) return 'text-accent-success';
    if (value >= 50) return 'text-accent-warning';
    return 'text-accent-danger';
  };

  const getPerformanceBadge = (value: number) => {
    if (value >= 70) return 'bg-green-500/20 text-green-400 border-green-500/30';
    if (value >= 50) return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30';
    return 'bg-red-500/20 text-red-400 border-red-500/30';
  };

  // Prepare radar data
  const radarData = branches.map((b) => ({
    branch: b.short_name || b.branch_name,
    performance: Math.min(b.avg_submission_score, 100),
    passRate: Math.min(b.pass_pct, 100),
    contest: Math.min(b.contest_participation, 100),
    assignment: Math.min(b.assignment_completion, 100),
  }));

  // Prepare trend data grouped by year
  const trendYears = [...new Set(trends.map((t) => t.year))].sort();
  const trendData = trendYears.map((year) => {
    const entry: Record<string, string | number> = { year };
    branches.forEach((b) => {
      const trend = trends.find((t) => t.year === year && t.branch_name === b.branch_name);
      entry[b.short_name || b.branch_name] = trend ? trend.avg_marks : 0;
    });
    return entry;
  });

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
          <span className="text-text-secondary">Loading branch analytics...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-text-primary mb-1">Branch Performance</h1>
        <p className="text-text-secondary text-sm">Comparative analytics across all departments</p>
      </div>

      {/* Best / Weakest Branch Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="card-elevated border-l-4 border-accent-success">
          <div className="flex items-center gap-3 mb-2">
            <Trophy className="w-6 h-6 text-accent-success" />
            <h3 className="text-lg font-semibold text-text-primary">Best Performing Branch</h3>
          </div>
          <p className="text-2xl font-bold text-accent-success">{bestBranch || 'N/A'}</p>
          <p className="text-sm text-text-secondary mt-1">Based on composite score</p>
        </div>
        <div className="card-elevated border-l-4 border-accent-danger">
          <div className="flex items-center gap-3 mb-2">
            <AlertTriangle className="w-6 h-6 text-accent-danger" />
            <h3 className="text-lg font-semibold text-text-primary">Needs Attention</h3>
          </div>
          <p className="text-2xl font-bold text-accent-danger">{weakestBranch || 'N/A'}</p>
          <p className="text-sm text-text-secondary mt-1">Requires intervention</p>
        </div>
      </div>

      {/* Radar Comparison Chart */}
      {radarData.length > 0 && (
        <div className="card">
          <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
            <GitBranch className="w-5 h-5" />
            Multi-Dimensional Comparison
          </h2>
          <div className="h-80">
            <ResponsiveContainer width="100%" height="100%">
              <RadarChart data={radarData}>
                <PolarGrid stroke="rgba(255,255,255,0.1)" />
                <PolarAngleAxis dataKey="branch" stroke="rgba(255,255,255,0.5)" />
                <PolarRadiusAxis stroke="rgba(255,255,255,0.2)" />
                <Radar
                  name="Performance"
                  dataKey="performance"
                  stroke="#10b981"
                  fill="#10b981"
                  fillOpacity={0.2}
                />
                <Radar
                  name="Pass Rate"
                  dataKey="passRate"
                  stroke="#3b82f6"
                  fill="#3b82f6"
                  fillOpacity={0.2}
                />
                <Legend />
                <Tooltip
                  contentStyle={{
                    backgroundColor: '#1f2937',
                    border: '1px solid #374151',
                    borderRadius: '8px',
                  }}
                />
              </RadarChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      {/* Branch Detail Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {branches.map((branch) => (
          <div key={branch.branch_id} className="card hover:shadow-lg transition-shadow">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-lg font-semibold text-text-primary">{branch.branch_name}</h3>
              <span
                className={`px-2 py-1 rounded-full text-xs border ${getPerformanceBadge(
                  branch.avg_submission_score
                )}`}
              >
                {branch.avg_submission_score.toFixed(1)}%
              </span>
            </div>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-text-secondary">Students</span>
                <span className="text-text-primary font-medium">{branch.total_students}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-text-secondary">Faculty</span>
                <span className="text-text-primary font-medium">{branch.total_faculty}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-text-secondary">Pass %</span>
                <span className={`font-medium ${getPerformanceColor(branch.pass_pct)}`}>
                  {branch.pass_pct.toFixed(1)}%
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-text-secondary">Contest Participation</span>
                <span
                  className={`font-medium ${getPerformanceColor(branch.contest_participation)}`}
                >
                  {branch.contest_participation.toFixed(1)}%
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-text-secondary">Assignment Completion</span>
                <span
                  className={`font-medium ${getPerformanceColor(branch.assignment_completion)}`}
                >
                  {branch.assignment_completion.toFixed(1)}%
                </span>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Performance Bar Chart */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4">Performance by Branch</h2>
        <div className="h-64">
          {branches.length > 0 ? (
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={branches}>
                <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.1)" />
                <XAxis dataKey="short_name" stroke="rgba(255,255,255,0.5)" />
                <YAxis stroke="rgba(255,255,255,0.5)" />
                <Tooltip
                  contentStyle={{
                    backgroundColor: '#1f2937',
                    border: '1px solid #374151',
                    borderRadius: '8px',
                  }}
                />
                <Bar dataKey="avg_submission_score" fill="#3b82f6" radius={[4, 4, 0, 0]} />
                <Bar dataKey="pass_pct" fill="#10b981" radius={[4, 4, 0, 0]} />
                <Legend />
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-full text-text-muted">
              No branch data available
            </div>
          )}
        </div>
      </div>

      {/* Year-over-Year Trend */}
      {trendData.length > 0 && (
        <div className="card">
          <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
            <TrendingUp className="w-5 h-5" />
            Year-over-Year Trends
          </h2>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={trendData}>
                <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.1)" />
                <XAxis dataKey="year" stroke="rgba(255,255,255,0.5)" />
                <YAxis stroke="rgba(255,255,255,0.5)" />
                <Tooltip
                  contentStyle={{
                    backgroundColor: '#1f2937',
                    border: '1px solid #374151',
                    borderRadius: '8px',
                  }}
                />
                <Legend />
                {branches.map((branch, index) => (
                  <Line
                    key={branch.branch_id}
                    type="monotone"
                    dataKey={branch.short_name || branch.branch_name}
                    stroke={COLORS[index % COLORS.length]}
                    strokeWidth={2}
                    dot={{ fill: COLORS[index % COLORS.length] }}
                  />
                ))}
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}
    </div>
  );
};

export default BranchAnalytics;
