import { useState, useEffect } from 'react';
import { principalAnalyticsAPI } from '../../../services/api';
import { showError } from '../../../utils/showAlert';
import {
  BookOpen,
  FlaskConical,
  Trophy,
  Users,
  GraduationCap,
  Activity,
  AlertTriangle,
  CheckCircle2,
  TrendingUp,
  TrendingDown,
  BarChart3,
} from 'lucide-react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
  Cell,
} from 'recharts';

interface OverviewData {
  total_active_courses: number;
  active_lab_sessions: number;
  contests_this_semester: number;
  student_participation_pct: number;
  avg_academic_performance: number;
  avg_attendance_pct: number;
  pass_fail_ratio: number;
  pending_evaluations: number;
  active_faculty_count: number;
  total_students: number;
  submissions_this_month: number;
  monthly_submission_trend: Array<{ month: string; count: number }>;
  branch_comparison: Array<{ branch_name: string; submissions: number }>;
}

interface AlertItem {
  type: string;
  message: string;
  severity: string;
  branch_name?: string;
  created_at: string;
}

const PrincipalOverview = () => {
  const [loading, setLoading] = useState<boolean>(true);
  const [data, setData] = useState<OverviewData | null>(null);
  const [alerts, setAlerts] = useState<AlertItem[]>([]);

  useEffect(() => {
    fetchOverview();
    fetchAlerts();
  }, []);

  const fetchOverview = async () => {
    try {
      const response = (await principalAnalyticsAPI.getOverview()) as OverviewData;
      setData(response);
    } catch (error) {
      console.error('Error fetching overview:', error);
      showError('Failed to load analytics overview');
    } finally {
      setLoading(false);
    }
  };

  const fetchAlerts = async () => {
    try {
      const response = (await principalAnalyticsAPI.getAlerts()) as { alerts: AlertItem[] };
      setAlerts(response.alerts.slice(0, 5));
    } catch (error) {
      console.error('Error fetching alerts:', error);
    }
  };

  const getTrendIcon = (value: number, threshold = 50) => {
    if (value >= threshold) {
      return <TrendingUp className="w-4 h-4 text-accent-success" />;
    }
    return <TrendingDown className="w-4 h-4 text-accent-danger" />;
  };

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
          <span className="text-text-secondary">Loading analytics...</span>
        </div>
      </div>
    );
  }

  if (!data) {
    return (
      <div className="text-text-secondary text-center py-12">No analytics data available.</div>
    );
  }

  const kpiCards = [
    {
      label: 'Active Courses',
      value: data.total_active_courses,
      icon: BookOpen,
      color: 'border-accent-secondary',
    },
    {
      label: 'Lab Sessions',
      value: data.active_lab_sessions,
      icon: FlaskConical,
      color: 'border-accent-info',
    },
    {
      label: 'Contests (Semester)',
      value: data.contests_this_semester,
      icon: Trophy,
      color: 'border-accent-warning',
    },
    {
      label: 'Student Participation',
      value: `${data.student_participation_pct.toFixed(1)}%`,
      icon: Users,
      color: 'border-accent-success',
      trend: getTrendIcon(data.student_participation_pct, 40),
    },
    {
      label: 'Avg Performance',
      value: `${data.avg_academic_performance.toFixed(1)}%`,
      icon: GraduationCap,
      color: 'border-accent-primary',
      trend: getTrendIcon(data.avg_academic_performance, 50),
    },
    {
      label: 'Pass/Fail Ratio',
      value: `${data.pass_fail_ratio.toFixed(1)}%`,
      icon: Activity,
      color: 'border-accent-secondary',
      trend: getTrendIcon(data.pass_fail_ratio, 60),
    },
    {
      label: 'Active Faculty',
      value: data.active_faculty_count,
      icon: Users,
      color: 'border-accent-info',
    },
    {
      label: 'Total Students',
      value: data.total_students,
      icon: GraduationCap,
      color: 'border-accent-primary',
    },
  ];

  const COLORS = ['#10b981', '#3b82f6', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899'];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-text-primary mb-1">Institution Overview</h1>
        <p className="text-text-secondary text-sm">
          High-level academic intelligence and performance metrics
        </p>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        {kpiCards.map((card, index) => (
          <div
            key={index}
            className={`card-elevated flex items-center gap-4 border-l-4 ${card.color}`}
          >
            <div className="flex-1">
              <div className="text-3xl font-bold text-text-primary">{card.value}</div>
              <div className="text-sm text-text-secondary mt-1 flex items-center gap-2">
                {card.label}
                {card.trend}
              </div>
            </div>
            <card.icon className="w-8 h-8 text-text-muted opacity-50" />
          </div>
        ))}
      </div>

      {/* Charts Row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Monthly Submission Trend */}
        <div className="card">
          <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
            <BarChart3 className="w-5 h-5" />
            Monthly Submission Trend
          </h2>
          <div className="h-64">
            {data.monthly_submission_trend && data.monthly_submission_trend.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={data.monthly_submission_trend}>
                  <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.1)" />
                  <XAxis dataKey="month" stroke="rgba(255,255,255,0.5)" />
                  <YAxis stroke="rgba(255,255,255,0.5)" />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: '#1f2937',
                      border: '1px solid #374151',
                      borderRadius: '8px',
                    }}
                  />
                  <Line
                    type="monotone"
                    dataKey="count"
                    stroke="#3b82f6"
                    strokeWidth={2}
                    dot={{ fill: '#3b82f6' }}
                  />
                </LineChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex items-center justify-center h-full text-text-muted">
                No trend data available
              </div>
            )}
          </div>
        </div>

        {/* Department Comparison */}
        <div className="card">
          <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
            <BarChart3 className="w-5 h-5" />
            Department Comparison (Submissions)
          </h2>
          <div className="h-64">
            {data.branch_comparison && data.branch_comparison.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={data.branch_comparison} layout="vertical">
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
                  <Bar dataKey="submissions" radius={[0, 4, 4, 0]}>
                    {data.branch_comparison.map((_, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex items-center justify-center h-full text-text-muted">
                No department data available
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Alerts Section */}
      {alerts.length > 0 && (
        <div className="card border-l-4 border-accent-danger">
          <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
            <AlertTriangle className="w-5 h-5 text-accent-danger" />
            Active Alerts ({alerts.length})
          </h2>
          <div className="space-y-3">
            {alerts.map((alert, index) => (
              <div
                key={index}
                className={`p-3 rounded-lg border ${
                  alert.severity === 'critical'
                    ? 'bg-red-500/10 border-red-500/30'
                    : alert.severity === 'warning'
                      ? 'bg-yellow-500/10 border-yellow-500/30'
                      : 'bg-blue-500/10 border-blue-500/30'
                }`}
              >
                <div className="flex items-start gap-3">
                  {alert.severity === 'critical' ? (
                    <AlertTriangle className="w-5 h-5 text-red-400 mt-0.5" />
                  ) : alert.severity === 'warning' ? (
                    <AlertTriangle className="w-5 h-5 text-yellow-400 mt-0.5" />
                  ) : (
                    <CheckCircle2 className="w-5 h-5 text-blue-400 mt-0.5" />
                  )}
                  <div className="flex-1">
                    <p className="text-text-primary text-sm">{alert.message}</p>
                    {alert.branch_name && (
                      <p className="text-text-muted text-xs mt-1">Branch: {alert.branch_name}</p>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default PrincipalOverview;
