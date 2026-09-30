import { useState, useEffect } from 'react';
import { principalAnalyticsAPI } from '../../../services/api';
import { showError } from '../../../utils/showAlert';
import {
  Lightbulb,
  AlertTriangle,
  CheckCircle2,
  Info,
  TrendingUp,
  TrendingDown,
  Users,
  BookOpen,
  Trophy,
  FlaskConical,
} from 'lucide-react';

interface InsightItem {
  type: string;
  message: string;
  severity: string;
  branch_name?: string;
  metric?: string;
  value?: string;
}

const Insights = () => {
  const [loading, setLoading] = useState<boolean>(true);
  const [insights, setInsights] = useState<InsightItem[]>([]);

  useEffect(() => {
    fetchInsights();
  }, []);

  const fetchInsights = async () => {
    try {
      const response = (await principalAnalyticsAPI.getInsights()) as { insights: InsightItem[] };
      setInsights(response.insights);
    } catch (error) {
      console.error('Error fetching insights:', error);
      showError('Failed to load AI insights');
    } finally {
      setLoading(false);
    }
  };

  const getSeverityIcon = (severity: string) => {
    switch (severity) {
      case 'success':
        return <CheckCircle2 className="w-5 h-5 text-accent-success mt-0.5" />;
      case 'warning':
        return <AlertTriangle className="w-5 h-5 text-accent-warning mt-0.5" />;
      case 'info':
        return <Info className="w-5 h-5 text-accent-info mt-0.5" />;
      default:
        return <Lightbulb className="w-5 h-5 text-accent-primary mt-0.5" />;
    }
  };

  const getSeverityBorder = (severity: string) => {
    switch (severity) {
      case 'success':
        return 'border-l-4 border-accent-success bg-green-500/5';
      case 'warning':
        return 'border-l-4 border-accent-warning bg-yellow-500/5';
      case 'info':
        return 'border-l-4 border-accent-info bg-blue-500/5';
      default:
        return 'border-l-4 border-accent-primary bg-background-tertiary';
    }
  };

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'branch_performance':
        return <TrendingUp className="w-4 h-4" />;
      case 'lab_decline':
        return <FlaskConical className="w-4 h-4" />;
      case 'contest_engagement':
        return <Trophy className="w-4 h-4" />;
      case 'subject_difficulty':
        return <BookOpen className="w-4 h-4" />;
      case 'faculty_workload':
        return <Users className="w-4 h-4" />;
      case 'coding_consistency':
        return <TrendingDown className="w-4 h-4" />;
      default:
        return <Lightbulb className="w-4 h-4" />;
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
          <span className="text-text-secondary">Generating insights...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-text-primary mb-1">AI-Powered Insights</h1>
        <p className="text-text-secondary text-sm">
          Rule-based intelligence for academic decision-making
        </p>
      </div>

      {/* Insights Feed */}
      {insights.length > 0 ? (
        <div className="space-y-4">
          {insights.map((insight, index) => (
            <div key={index} className={`p-4 rounded-lg ${getSeverityBorder(insight.severity)}`}>
              <div className="flex items-start gap-3">
                {getSeverityIcon(insight.severity)}
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-1">
                    {getTypeIcon(insight.type)}
                    <span className="text-xs text-text-muted uppercase tracking-wider">
                      {insight.type.replace(/_/g, ' ')}
                    </span>
                    {insight.branch_name && (
                      <span className="text-xs text-text-muted">({insight.branch_name})</span>
                    )}
                  </div>
                  <p className="text-text-primary">{insight.message}</p>
                  {insight.metric && insight.value && (
                    <div className="mt-2 flex items-center gap-2">
                      <span className="text-xs text-text-muted">{insight.metric}:</span>
                      <span className="text-sm font-semibold text-accent-primary">
                        {insight.value}
                      </span>
                    </div>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="card text-center py-12">
          <Lightbulb className="w-12 h-12 text-text-muted mx-auto mb-4" />
          <p className="text-text-secondary">No insights available yet.</p>
          <p className="text-text-muted text-sm mt-1">
            Insights are generated based on recent academic data and trends.
          </p>
        </div>
      )}
    </div>
  );
};

export default Insights;
