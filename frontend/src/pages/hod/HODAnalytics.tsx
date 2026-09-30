import { useState, useEffect } from 'react';
import api from '../../services/api';
import { showError } from '../../utils/showAlert';

interface TopPerformer {
  rank: number;
  name: string;
  solved: number;
}

interface CourseCompletion {
  course_code: string;
  course_name: string;
  total_enrolled: number;
  has_lab: boolean;
  has_theory: boolean;
}

interface WeeklySubmissionData {
  week: number;
  count: number;
}

interface AnalyticsData {
  top_performers: TopPerformer[];
  course_completion: CourseCompletion[];
  weekly_submission_data: WeeklySubmissionData[];
}

const HODAnalytics = () => {
  const [loading, setLoading] = useState<boolean>(true);
  const [analytics, setAnalytics] = useState<AnalyticsData>({
    top_performers: [],
    course_completion: [],
    weekly_submission_data: [],
  });

  useEffect(() => {
    fetchAnalytics();
  }, []);

  const fetchAnalytics = async () => {
    try {
      const response = (await api.get('/hod/analytics')) as AnalyticsData;
      setAnalytics(response);
    } catch (error) {
      console.error('Error fetching analytics:', error);
      showError('Failed to load branch analytics');
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
          <span className="text-text-secondary">Loading analytics...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-text-primary mb-1">Branch Analytics</h1>
        <p className="text-text-secondary text-sm">
          Overall branch performance metrics and insights
        </p>
      </div>

      {/* Top Performers */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4">Top Performers</h2>
        {analytics.top_performers.length === 0 ? (
          <p className="text-text-muted text-center py-8">No performance data available yet</p>
        ) : (
          <div className="table-container">
            <table className="table">
              <thead>
                <tr>
                  <th>Rank</th>
                  <th>Student</th>
                  <th>Problems Solved</th>
                </tr>
              </thead>
              <tbody>
                {analytics.top_performers.map((performer) => (
                  <tr key={performer.rank}>
                    <td className="text-text-secondary">#{performer.rank}</td>
                    <td className="text-text-primary font-medium">{performer.name}</td>
                    <td>
                      <span className="badge badge-success">{performer.solved}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Course Completion */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4">Course Completion</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {analytics.course_completion.map((course) => (
            <div
              key={course.course_code}
              className="p-4 bg-background-tertiary rounded-lg border border-background-border"
            >
              <div className="flex items-center justify-between mb-3">
                <span className="font-semibold text-text-primary">{course.course_code}</span>
                <div className="flex gap-2">
                  {course.has_lab && <span className="badge badge-success text-[10px]">Lab</span>}
                  {course.has_theory && (
                    <span className="badge badge-info text-[10px]">Theory</span>
                  )}
                </div>
              </div>
              <div className="text-sm text-text-secondary mb-1">{course.course_name}</div>
              <div className="text-sm text-text-primary">Enrolled: {course.total_enrolled}</div>
            </div>
          ))}
        </div>
      </div>

      {/* Weekly Submission Trend */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4">Weekly Submission Trend</h2>
        {analytics.weekly_submission_data.length === 0 ? (
          <p className="text-text-muted text-center py-8">No submission data available</p>
        ) : (
          <div className="h-48 flex items-end gap-2">
            {analytics.weekly_submission_data.map((data) => (
              <div key={data.week} className="flex-1 flex flex-col items-center">
                <div
                  className="w-full bg-accent-secondary rounded-t"
                  style={{
                    height: `${Math.max(10, (data.count / Math.max(...analytics.weekly_submission_data.map((d) => d.count), 1)) * 100)}%`,
                  }}
                />
                <span className="text-xs text-text-tertiary mt-2">W{data.week}</span>
                <span className="text-xs text-text-primary">{data.count}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default HODAnalytics;
