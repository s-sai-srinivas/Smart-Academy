import { useState, useEffect } from 'react';
import { principalAnalyticsAPI } from '../../../services/api';
import { showError } from '../../../utils/showAlert';
import { FlaskConical, AlertTriangle, CheckCircle2 } from 'lucide-react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';

interface LabConductItem {
  lab_name: string;
  course_code: string;
  planned: number;
  conducted: number;
  missed: number;
  avg_attendance: number;
  avg_score: number;
}

interface LabPerformanceItem {
  lab_name: string;
  session_name: string;
  avg_score: number;
  completion_rate: number;
  total_submissions: number;
}

interface RiskItem {
  lab_name: string;
  branch_name: string;
  avg_score: number;
  attendance_pct: number;
  risk_level: string;
}

const LabAnalytics = () => {
  const [loading, setLoading] = useState<boolean>(true);
  const [conduct, setConduct] = useState<LabConductItem[]>([]);
  const [performance, setPerformance] = useState<LabPerformanceItem[]>([]);
  const [risks, setRisks] = useState<RiskItem[]>([]);

  useEffect(() => {
    fetchLabAnalytics();
  }, []);

  const fetchLabAnalytics = async () => {
    try {
      const response = (await principalAnalyticsAPI.getLabAnalytics()) as {
        lab_conduct: LabConductItem[];
        lab_performance: LabPerformanceItem[];
        risk_indicators: RiskItem[];
      };
      setConduct(response.lab_conduct);
      setPerformance(response.lab_performance);
      setRisks(response.risk_indicators);
    } catch (error) {
      console.error('Error fetching lab analytics:', error);
      showError('Failed to load lab analytics');
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
          <span className="text-text-secondary">Loading lab analytics...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-text-primary mb-1">Lab Session Analytics</h1>
        <p className="text-text-secondary text-sm">Lab conduct, performance, and risk monitoring</p>
      </div>

      {/* Risk Indicators */}
      {risks.length > 0 && (
        <div className="card border-l-4 border-accent-danger">
          <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
            <AlertTriangle className="w-5 h-5 text-accent-danger" />
            Risk Indicators
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {risks.map((risk, index) => (
              <div
                key={index}
                className={`p-4 rounded-lg border ${
                  risk.risk_level === 'high'
                    ? 'bg-red-500/10 border-red-500/30'
                    : 'bg-yellow-500/10 border-yellow-500/30'
                }`}
              >
                <h3 className="text-text-primary font-semibold">{risk.lab_name}</h3>
                <p className="text-text-secondary text-sm">{risk.branch_name}</p>
                <div className="mt-2 flex items-center gap-4 text-sm">
                  <span className="text-red-400">Score: {risk.avg_score.toFixed(1)}%</span>
                  <span className="text-yellow-400">
                    Attendance: {risk.attendance_pct.toFixed(1)}%
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Lab Conduct Stacked Bar */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
          <FlaskConical className="w-5 h-5" />
          Lab Conduct Tracking
        </h2>
        <div className="h-64">
          {conduct.length > 0 ? (
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={conduct}>
                <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.1)" />
                <XAxis dataKey="lab_name" stroke="rgba(255,255,255,0.5)" />
                <YAxis stroke="rgba(255,255,255,0.5)" />
                <Tooltip
                  contentStyle={{
                    backgroundColor: '#1f2937',
                    border: '1px solid #374151',
                    borderRadius: '8px',
                  }}
                />
                <Legend />
                <Bar dataKey="planned" fill="#3b82f6" stackId="a" />
                <Bar dataKey="conducted" fill="#10b981" stackId="a" />
                <Bar dataKey="missed" fill="#ef4444" stackId="a" />
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-full text-text-muted">
              No lab conduct data available
            </div>
          )}
        </div>
      </div>

      {/* Lab Performance Table */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
          <CheckCircle2 className="w-5 h-5" />
          Lab Session Performance
        </h2>
        <div className="table-container overflow-x-auto">
          <table className="table w-full">
            <thead>
              <tr>
                <th>Lab</th>
                <th>Session</th>
                <th>Avg Score</th>
                <th>Completion Rate</th>
                <th>Submissions</th>
              </tr>
            </thead>
            <tbody>
              {performance.map((perf, index) => (
                <tr key={index}>
                  <td className="text-text-primary font-medium">{perf.lab_name}</td>
                  <td className="text-text-secondary">{perf.session_name}</td>
                  <td className="text-text-primary">{perf.avg_score.toFixed(1)}%</td>
                  <td>
                    <div className="flex items-center gap-2">
                      <div className="w-20 h-2 bg-background-border rounded-full overflow-hidden">
                        <div
                          className="h-full bg-accent-secondary rounded-full"
                          style={{ width: `${Math.min(perf.completion_rate, 100)}%` }}
                        />
                      </div>
                      <span className="text-text-primary text-sm">
                        {perf.completion_rate.toFixed(1)}%
                      </span>
                    </div>
                  </td>
                  <td className="text-text-secondary">{perf.total_submissions}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

export default LabAnalytics;
