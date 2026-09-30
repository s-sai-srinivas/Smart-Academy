import { useState, useEffect } from 'react';
import { principalAnalyticsAPI } from '../../../services/api';
import { showError } from '../../../utils/showAlert';
import { Calendar, TrendingUp, Users } from 'lucide-react';
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
  Legend,
} from 'recharts';

interface CohortTrend {
  cohort_year: number;
  total_students: number;
  avg_marks: number;
  pass_pct: number;
  submissions: number;
}

interface HistoricalPass {
  year: number;
  pass_pct: number;
}

interface PlacementReadiness {
  cohort_year: number;
  coding_score: number;
  contest_participation: number;
  skill_assessments: number;
}

interface GrowthMetrics {
  improvement_rate: number;
  dept_growth: number;
}

const YearAnalytics = () => {
  const [loading, setLoading] = useState<boolean>(true);
  const [cohortTrends, setCohortTrends] = useState<CohortTrend[]>([]);
  const [historicalPass, setHistoricalPass] = useState<HistoricalPass[]>([]);
  const [placementReadiness, setPlacementReadiness] = useState<PlacementReadiness[]>([]);
  const [growthMetrics, setGrowthMetrics] = useState<GrowthMetrics | null>(null);

  useEffect(() => {
    fetchYearAnalytics();
  }, []);

  const fetchYearAnalytics = async () => {
    try {
      const response = (await principalAnalyticsAPI.getYearAnalytics()) as {
        cohort_trends: CohortTrend[];
        historical_pass_pct: HistoricalPass[];
        placement_readiness: PlacementReadiness[];
        growth_metrics: GrowthMetrics;
      };
      setCohortTrends(response.cohort_trends);
      setHistoricalPass(response.historical_pass_pct);
      setPlacementReadiness(response.placement_readiness);
      setGrowthMetrics(response.growth_metrics);
    } catch (error) {
      console.error('Error fetching year analytics:', error);
      showError('Failed to load year analytics');
    } finally {
      setLoading(false);
    }
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
          <span className="text-text-secondary">Loading year analytics...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-text-primary mb-1">Year-wise Analytics</h1>
        <p className="text-text-secondary text-sm">
          Cohort trends, historical pass rates, and placement readiness
        </p>
      </div>

      {/* Growth Metrics */}
      {growthMetrics && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="card-elevated border-l-4 border-accent-success">
            <div className="flex items-center gap-3 mb-2">
              <TrendingUp className="w-6 h-6 text-accent-success" />
              <h3 className="text-lg font-semibold text-text-primary">
                Year-over-Year Improvement
              </h3>
            </div>
            <p className="text-2xl font-bold text-accent-success">
              {growthMetrics.improvement_rate > 0 ? '+' : ''}
              {growthMetrics.improvement_rate.toFixed(1)}%
            </p>
            <p className="text-sm text-text-secondary mt-1">Academic performance change</p>
          </div>
          <div className="card-elevated border-l-4 border-accent-info">
            <div className="flex items-center gap-3 mb-2">
              <Users className="w-6 h-6 text-accent-info" />
              <h3 className="text-lg font-semibold text-text-primary">Department Growth</h3>
            </div>
            <p className="text-2xl font-bold text-accent-info">
              {growthMetrics.dept_growth > 0 ? '+' : ''}
              {growthMetrics.dept_growth.toFixed(1)}%
            </p>
            <p className="text-sm text-text-secondary mt-1">Student enrollment growth</p>
          </div>
        </div>
      )}

      {/* Cohort Comparison */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
          <Calendar className="w-5 h-5" />
          Cohort Performance Comparison
        </h2>
        <div className="h-64">
          {cohortTrends.length > 0 ? (
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={cohortTrends}>
                <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.1)" />
                <XAxis dataKey="cohort_year" stroke="rgba(255,255,255,0.5)" />
                <YAxis stroke="rgba(255,255,255,0.5)" />
                <Tooltip
                  contentStyle={{
                    backgroundColor: '#1f2937',
                    border: '1px solid #374151',
                    borderRadius: '8px',
                  }}
                />
                <Legend />
                <Bar dataKey="avg_marks" fill="#3b82f6" name="Avg Marks" radius={[4, 4, 0, 0]} />
                <Bar dataKey="pass_pct" fill="#10b981" name="Pass %" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-full text-text-muted">
              No cohort data available
            </div>
          )}
        </div>
      </div>

      {/* Historical Pass % */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4">Historical Pass Percentage</h2>
        <div className="h-64">
          {historicalPass.length > 0 ? (
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={historicalPass}>
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
                <Line
                  type="monotone"
                  dataKey="pass_pct"
                  stroke="#10b981"
                  strokeWidth={2}
                  dot={{ fill: '#10b981' }}
                  name="Pass %"
                />
              </LineChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-full text-text-muted">
              No historical data available
            </div>
          )}
        </div>
      </div>

      {/* Placement Readiness */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4">
          Placement Readiness by Cohort
        </h2>
        <div className="h-64">
          {placementReadiness.length > 0 ? (
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={placementReadiness}>
                <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.1)" />
                <XAxis dataKey="cohort_year" stroke="rgba(255,255,255,0.5)" />
                <YAxis stroke="rgba(255,255,255,0.5)" />
                <Tooltip
                  contentStyle={{
                    backgroundColor: '#1f2937',
                    border: '1px solid #374151',
                    borderRadius: '8px',
                  }}
                />
                <Legend />
                <Bar
                  dataKey="coding_score"
                  fill="#3b82f6"
                  name="Coding Score"
                  radius={[4, 4, 0, 0]}
                />
                <Bar
                  dataKey="contest_participation"
                  fill="#f59e0b"
                  name="Contest Participation"
                  radius={[4, 4, 0, 0]}
                />
                <Bar
                  dataKey="skill_assessments"
                  fill="#8b5cf6"
                  name="Skill Assessments"
                  radius={[4, 4, 0, 0]}
                />
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-full text-text-muted">
              No placement data available
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default YearAnalytics;
