import { useState, useEffect } from 'react';
import { principalAnalyticsAPI } from '../../../services/api';
import { showError } from '../../../utils/showAlert';
import { AlertTriangle, CheckCircle2, Users } from 'lucide-react';
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

interface CourseItem {
  course_code: string;
  course_name: string;
  total_sessions: number;
  completed: number;
  pending: number;
  completion_pct: number;
  avg_marks: number;
  total_students: number;
}

interface DifficultyItem {
  course_code: string;
  course_name: string;
  avg_marks: number;
  failure_pct: number;
}

interface FacultyItem {
  regdno: string;
  name: string;
  courses: number;
  students: number;
  avg_marks: number;
  submissions: number;
}

const CourseAnalytics = () => {
  const [loading, setLoading] = useState<boolean>(true);
  const [courses, setCourses] = useState<CourseItem[]>([]);
  const [difficulties, setDifficulties] = useState<DifficultyItem[]>([]);
  const [faculty, setFaculty] = useState<FacultyItem[]>([]);

  useEffect(() => {
    fetchCourseAnalytics();
  }, []);

  const fetchCourseAnalytics = async () => {
    try {
      const response = (await principalAnalyticsAPI.getCourseAnalytics()) as {
        course_completion: CourseItem[];
        difficulty_indicators: DifficultyItem[];
        faculty_effectiveness: FacultyItem[];
      };
      setCourses(response.course_completion);
      setDifficulties(response.difficulty_indicators);
      setFaculty(response.faculty_effectiveness);
    } catch (error) {
      console.error('Error fetching course analytics:', error);
      showError('Failed to load course analytics');
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
          <span className="text-text-secondary">Loading course analytics...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-text-primary mb-1">Course Analytics</h1>
        <p className="text-text-secondary text-sm">
          Course completion, difficulty, and faculty effectiveness
        </p>
      </div>

      {/* Difficulty Indicators */}
      <div className="card border-l-4 border-accent-danger">
        <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
          <AlertTriangle className="w-5 h-5 text-accent-danger" />
          Hardest Courses
        </h2>
        <div className="h-64">
          {difficulties.length > 0 ? (
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={difficulties} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.1)" />
                <XAxis type="number" stroke="rgba(255,255,255,0.5)" />
                <YAxis
                  dataKey="course_code"
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
                <Bar dataKey="avg_marks" radius={[0, 4, 4, 0]}>
                  {difficulties.map((_, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-full text-text-muted">
              No difficulty data available
            </div>
          )}
        </div>
      </div>

      {/* Course Completion Table */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
          <CheckCircle2 className="w-5 h-5" />
          Course Completion
        </h2>
        <div className="table-container overflow-x-auto">
          <table className="table w-full">
            <thead>
              <tr>
                <th>Course</th>
                <th>Students</th>
                <th>Sessions</th>
                <th>Completed</th>
                <th>Pending</th>
                <th>Completion %</th>
                <th>Avg Marks</th>
              </tr>
            </thead>
            <tbody>
              {courses.map((course) => (
                <tr key={course.course_code}>
                  <td className="text-text-primary font-medium">
                    {course.course_code}
                    <div className="text-xs text-text-secondary">{course.course_name}</div>
                  </td>
                  <td className="text-text-secondary">{course.total_students}</td>
                  <td className="text-text-secondary">{course.total_sessions}</td>
                  <td className="text-accent-success">{course.completed}</td>
                  <td className="text-accent-warning">{course.pending}</td>
                  <td>
                    <div className="flex items-center gap-2">
                      <div className="w-20 h-2 bg-background-border rounded-full overflow-hidden">
                        <div
                          className="h-full bg-accent-secondary rounded-full"
                          style={{ width: `${Math.min(course.completion_pct, 100)}%` }}
                        />
                      </div>
                      <span className="text-text-primary text-sm">
                        {course.completion_pct.toFixed(1)}%
                      </span>
                    </div>
                  </td>
                  <td className="text-text-primary">{course.avg_marks.toFixed(1)}%</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Faculty Effectiveness */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4 flex items-center gap-2">
          <Users className="w-5 h-5" />
          Faculty Effectiveness
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {faculty.map((f) => (
            <div
              key={f.regdno}
              className="p-4 bg-background-tertiary rounded-lg border border-background-border"
            >
              <h3 className="text-text-primary font-semibold">{f.name}</h3>
              <div className="mt-2 space-y-1 text-sm">
                <div className="flex justify-between">
                  <span className="text-text-secondary">Courses</span>
                  <span className="text-text-primary">{f.courses}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-text-secondary">Students</span>
                  <span className="text-text-primary">{f.students}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-text-secondary">Avg Marks</span>
                  <span className="text-text-primary">{f.avg_marks.toFixed(1)}%</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-text-secondary">Submissions</span>
                  <span className="text-text-primary">{f.submissions}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default CourseAnalytics;
