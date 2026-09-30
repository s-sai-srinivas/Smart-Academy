import { useState, useEffect } from 'react';
import { principalAnalyticsAPI } from '../../../services/api';
import { showError } from '../../../utils/showAlert';
import {
  Award,
  Users,
  BookOpen,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  TrendingUp,
} from 'lucide-react';

interface OutcomeMetric {
  metric_name: string;
  value: number;
  target: number;
  status: string;
}

interface DeptReport {
  branch_name: string;
  total_students: number;
  total_faculty: number;
  avg_performance: number;
  pass_pct: number;
  contest_winners: number;
}

interface FacultyWorkload {
  name: string;
  courses: number;
  students: number;
  avg_class_size: number;
}

interface StudentAchievement {
  name: string;
  branch_name: string;
  achievement: string;
  score: number;
}

const Accreditation = () => {
  const [loading, setLoading] = useState<boolean>(true);
  const [outcomeMetrics, setOutcomeMetrics] = useState<OutcomeMetric[]>([]);
  const [deptReports, setDeptReports] = useState<DeptReport[]>([]);
  const [facultyWorkload, setFacultyWorkload] = useState<FacultyWorkload[]>([]);
  const [achievements, setAchievements] = useState<StudentAchievement[]>([]);

  useEffect(() => {
    fetchAccreditation();
  }, []);

  const fetchAccreditation = async () => {
    try {
      const response = (await principalAnalyticsAPI.getAccreditation()) as {
        outcome_metrics: OutcomeMetric[];
        department_reports: DeptReport[];
        faculty_workload: FacultyWorkload[];
        student_achievements: StudentAchievement[];
      };
      setOutcomeMetrics(response.outcome_metrics);
      setDeptReports(response.department_reports);
      setFacultyWorkload(response.faculty_workload);
      setAchievements(response.student_achievements);
    } catch (error) {
      console.error('Error fetching accreditation:', error);
      showError('Failed to load accreditation data');
    } finally {
      setLoading(false);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'achieved':
        return <CheckCircle2 className="w-5 h-5 text-accent-success" />;
      case 'near_target':
        return <AlertTriangle className="w-5 h-5 text-accent-warning" />;
      case 'below_target':
        return <XCircle className="w-5 h-5 text-accent-danger" />;
      default:
        return <TrendingUp className="w-5 h-5 text-text-muted" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'achieved':
        return 'text-accent-success bg-green-500/10 border-green-500/30';
      case 'near_target':
        return 'text-accent-warning bg-yellow-500/10 border-yellow-500/30';
      case 'below_target':
        return 'text-accent-danger bg-red-500/10 border-red-500/30';
      default:
        return 'text-text-muted bg-background-tertiary';
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
          <span className="text-text-secondary">Loading accreditation data...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-text-primary mb-1">Accreditation Reports</h1>
        <p className="text-text-secondary text-sm">NBA/NAAC support metrics and reports</p>
      </div>

      {/* Outcome Metrics */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
          <Award className="w-5 h-5" />
          Outcome-Based Education Metrics
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {outcomeMetrics.map((metric, index) => (
            <div key={index} className={`p-4 rounded-lg border ${getStatusColor(metric.status)}`}>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-medium">{metric.metric_name}</span>
                {getStatusIcon(metric.status)}
              </div>
              <div className="text-2xl font-bold">{metric.value.toFixed(1)}</div>
              <div className="text-xs mt-1 opacity-75">Target: {metric.target}</div>
              <div className="mt-2 w-full bg-background-border rounded-full h-2">
                <div
                  className="h-full rounded-full bg-current"
                  style={{
                    width: `${Math.min((metric.value / metric.target) * 100, 100)}%`,
                  }}
                />
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Department Reports */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
          <BookOpen className="w-5 h-5" />
          Department Performance Reports
        </h2>
        <div className="table-container overflow-x-auto">
          <table className="table w-full">
            <thead>
              <tr>
                <th>Department</th>
                <th>Students</th>
                <th>Faculty</th>
                <th>Avg Performance</th>
                <th>Pass %</th>
                <th>Contest Winners</th>
              </tr>
            </thead>
            <tbody>
              {deptReports.map((dept, index) => (
                <tr key={index}>
                  <td className="text-text-primary font-medium">{dept.branch_name}</td>
                  <td className="text-text-secondary">{dept.total_students}</td>
                  <td className="text-text-secondary">{dept.total_faculty}</td>
                  <td className="text-text-primary">{dept.avg_performance.toFixed(1)}%</td>
                  <td>
                    <span
                      className={`px-2 py-1 rounded-full text-xs border ${
                        dept.pass_pct >= 80
                          ? 'bg-green-500/20 text-green-400 border-green-500/30'
                          : dept.pass_pct >= 60
                            ? 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30'
                            : 'bg-red-500/20 text-red-400 border-red-500/30'
                      }`}
                    >
                      {dept.pass_pct.toFixed(1)}%
                    </span>
                  </td>
                  <td className="text-accent-success">{dept.contest_winners}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Faculty Workload */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
          <Users className="w-5 h-5" />
          Faculty Workload Report
        </h2>
        <div className="table-container overflow-x-auto">
          <table className="table w-full">
            <thead>
              <tr>
                <th>Faculty</th>
                <th>Courses</th>
                <th>Students</th>
                <th>Avg Class Size</th>
                <th>Load Status</th>
              </tr>
            </thead>
            <tbody>
              {facultyWorkload.map((fw, index) => (
                <tr key={index}>
                  <td className="text-text-primary font-medium">{fw.name}</td>
                  <td className="text-text-secondary">{fw.courses}</td>
                  <td className="text-text-secondary">{fw.students}</td>
                  <td className="text-text-primary">{fw.avg_class_size.toFixed(1)}</td>
                  <td>
                    <span
                      className={`px-2 py-1 rounded-full text-xs border ${
                        fw.courses > 5
                          ? 'bg-red-500/20 text-red-400 border-red-500/30'
                          : fw.courses > 3
                            ? 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30'
                            : 'bg-green-500/20 text-green-400 border-green-500/30'
                      }`}
                    >
                      {fw.courses > 5 ? 'High' : fw.courses > 3 ? 'Moderate' : 'Optimal'}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Student Achievements */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
          <Award className="w-5 h-5 text-accent-warning" />
          Student Achievements
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {achievements.map((achievement, index) => (
            <div
              key={index}
              className="p-4 bg-background-tertiary rounded-lg border border-background-border"
            >
              <div className="flex items-center gap-2 mb-2">
                <Award className="w-5 h-5 text-accent-warning" />
                <span className="text-xs text-text-muted uppercase tracking-wider">
                  {achievement.achievement}
                </span>
              </div>
              <h3 className="text-text-primary font-semibold">{achievement.name}</h3>
              <p className="text-text-secondary text-sm">{achievement.branch_name}</p>
              <div className="mt-2 text-lg font-bold text-accent-primary">
                {achievement.score.toFixed(1)} pts
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default Accreditation;
