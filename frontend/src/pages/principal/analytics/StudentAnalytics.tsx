import { useState, useEffect } from 'react';
import { principalAnalyticsAPI } from '../../../services/api';
import { showError } from '../../../utils/showAlert';
import { AlertTriangle, Award, BookOpen } from 'lucide-react';
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

interface SubjectPerformance {
  subject_name: string;
  avg_marks: number;
  pass_pct: number;
  total_students: number;
}

interface StudentItem {
  regdno: string;
  name: string;
  branch_name: string;
  avg_marks: number;
  total_submissions: number;
  status: string;
}

const StudentAnalytics = () => {
  const [loading, setLoading] = useState<boolean>(true);
  const [subjects, setSubjects] = useState<SubjectPerformance[]>([]);
  const [weakStudents, setWeakStudents] = useState<StudentItem[]>([]);
  const [topPerformers, setTopPerformers] = useState<StudentItem[]>([]);
  const [atRiskCount, setAtRiskCount] = useState<number>(0);

  useEffect(() => {
    fetchStudentAnalytics();
  }, []);

  const fetchStudentAnalytics = async () => {
    try {
      const response = (await principalAnalyticsAPI.getStudentAnalytics()) as {
        subject_performance: SubjectPerformance[];
        weak_students: StudentItem[];
        top_performers: StudentItem[];
        at_risk_count: number;
      };
      setSubjects(response.subject_performance);
      setWeakStudents(response.weak_students);
      setTopPerformers(response.top_performers);
      setAtRiskCount(response.at_risk_count);
    } catch (error) {
      console.error('Error fetching student analytics:', error);
      showError('Failed to load student analytics');
    } finally {
      setLoading(false);
    }
  };

  const COLORS = ['#ef4444', '#f59e0b', '#3b82f6', '#10b981', '#8b5cf6'];

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
          <span className="text-text-secondary">Loading student analytics...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-text-primary mb-1">Student Performance</h1>
        <p className="text-text-secondary text-sm">
          Subject analysis, at-risk students, and top performers
        </p>
      </div>

      {/* At Risk Summary */}
      {atRiskCount > 0 && (
        <div className="card-elevated border-l-4 border-accent-danger">
          <div className="flex items-center gap-3">
            <AlertTriangle className="w-8 h-8 text-accent-danger" />
            <div>
              <div className="text-2xl font-bold text-accent-danger">{atRiskCount}</div>
              <div className="text-sm text-text-secondary">
                Students below 40% average performance
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Subject Performance */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
          <BookOpen className="w-5 h-5" />
          Subject Performance
        </h2>
        <div className="h-64 mb-4">
          {subjects.length > 0 ? (
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={subjects} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.1)" />
                <XAxis type="number" stroke="rgba(255,255,255,0.5)" />
                <YAxis
                  dataKey="subject_name"
                  type="category"
                  stroke="rgba(255,255,255,0.5)"
                  width={120}
                />
                <Tooltip
                  contentStyle={{
                    backgroundColor: '#1f2937',
                    border: '1px solid #374151',
                    borderRadius: '8px',
                  }}
                />
                <Bar dataKey="avg_marks" radius={[0, 4, 4, 0]}>
                  {subjects.map((_, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-full text-text-muted">
              No subject data available
            </div>
          )}
        </div>
        <div className="table-container overflow-x-auto">
          <table className="table w-full">
            <thead>
              <tr>
                <th>Subject</th>
                <th>Avg Marks</th>
                <th>Pass %</th>
                <th>Total Students</th>
              </tr>
            </thead>
            <tbody>
              {subjects.map((subject, index) => (
                <tr key={index}>
                  <td className="text-text-primary font-medium">{subject.subject_name}</td>
                  <td className="text-text-primary">{subject.avg_marks.toFixed(1)}%</td>
                  <td>
                    <span
                      className={`px-2 py-1 rounded-full text-xs border ${
                        subject.pass_pct >= 70
                          ? 'bg-green-500/20 text-green-400 border-green-500/30'
                          : subject.pass_pct >= 50
                            ? 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30'
                            : 'bg-red-500/20 text-red-400 border-red-500/30'
                      }`}
                    >
                      {subject.pass_pct.toFixed(1)}%
                    </span>
                  </td>
                  <td className="text-text-secondary">{subject.total_students}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Weak Students */}
      {weakStudents.length > 0 && (
        <div className="card border-l-4 border-accent-danger">
          <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
            <AlertTriangle className="w-5 h-5 text-accent-danger" />
            At-Risk Students
          </h2>
          <div className="table-container overflow-x-auto">
            <table className="table w-full">
              <thead>
                <tr>
                  <th>Regd No</th>
                  <th>Name</th>
                  <th>Branch</th>
                  <th>Avg Marks</th>
                  <th>Submissions</th>
                </tr>
              </thead>
              <tbody>
                {weakStudents.map((student, index) => (
                  <tr key={index}>
                    <td className="text-text-secondary">{student.regdno}</td>
                    <td className="text-text-primary font-medium">{student.name}</td>
                    <td className="text-text-secondary">{student.branch_name}</td>
                    <td className="text-accent-danger">{student.avg_marks.toFixed(1)}%</td>
                    <td className="text-text-secondary">{student.total_submissions}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Top Performers */}
      {topPerformers.length > 0 && (
        <div className="card border-l-4 border-accent-success">
          <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
            <Award className="w-5 h-5 text-accent-success" />
            Top Performers
          </h2>
          <div className="table-container overflow-x-auto">
            <table className="table w-full">
              <thead>
                <tr>
                  <th>Rank</th>
                  <th>Name</th>
                  <th>Branch</th>
                  <th>Avg Marks</th>
                  <th>Submissions</th>
                </tr>
              </thead>
              <tbody>
                {topPerformers.map((student, index) => (
                  <tr key={index}>
                    <td className="text-text-secondary">#{index + 1}</td>
                    <td className="text-text-primary font-medium">{student.name}</td>
                    <td className="text-text-secondary">{student.branch_name}</td>
                    <td className="text-accent-success">{student.avg_marks.toFixed(1)}%</td>
                    <td className="text-text-secondary">{student.total_submissions}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
};

export default StudentAnalytics;
