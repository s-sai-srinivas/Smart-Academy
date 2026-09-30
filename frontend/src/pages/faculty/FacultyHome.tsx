import { useState, useEffect } from 'react';
import { useAuth } from '../../context/AuthContext';
import { dashboardAPI } from '../../services/api';
import { showError } from '../../utils/showAlert';
import { Link } from 'react-router-dom';
import { BookOpen, FileQuestion, Calendar } from 'lucide-react';

interface Course {
  id: number;
  ID?: number;
  course_code?: string;
  CourseCode?: string;
  course_name?: string;
  CourseName?: string;
  course_type?: string;
  CourseType?: string;
  section_name?: string;
  SectionName?: string;
  academic_year?: string;
  AcademicYear?: string;
  credits?: number;
  Credits?: number;
}

function FacultyHome() {
  const { user } = useAuth();
  const [loading, setLoading] = useState<boolean>(true);
  const [courses, setCourses] = useState<Course[]>([]);

  useEffect(() => {
    if (user) {
      loadDashboardData();
    } else {
      setLoading(false);
    }
  }, [user]);

  const loadDashboardData = async () => {
    try {
      const data = (await dashboardAPI.getData()) as { courses?: Course[] };
      // Get unique courses (deduplicate by course_id)
      const uniqueCourses: Course[] = [];
      const seenCourseIds = new Set<number>();

      (data.courses || []).forEach((course: Course) => {
        if (!seenCourseIds.has(course.id)) {
          seenCourseIds.add(course.id);
          uniqueCourses.push(course);
        }
      });

      setCourses(uniqueCourses);
    } catch (err) {
      console.error('Failed to load dashboard:', err);
      showError('Failed to load dashboard data');
    } finally {
      setLoading(false);
    }
  };

  if (!user) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <h2 className="text-2xl font-semibold text-text-primary mb-2">Welcome to CodePlatform</h2>
          <p className="text-text-muted">Please login to view your dashboard</p>
        </div>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <div className="w-8 h-8 border-2 border-background-border border-t-gray-500 rounded-full animate-spin mx-auto mb-3" />
          <p className="text-text-muted">Loading your courses...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {/* Welcome Header */}
      <div>
        <h1 className="text-3xl font-bold text-text-primary mb-2">Welcome back, {user.name}!</h1>
        <p className="text-text-secondary">Manage your courses and track student progress</p>
      </div>

      {/* Quick Actions */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Link
          to="/faculty/quizzes"
          className="group relative p-5 bg-gradient-to-br from-purple-500/10 to-purple-500/5 rounded-xl border-2 border-purple-500/20 hover:border-purple-500/40 transition-all"
        >
          <div className="flex items-center gap-3 mb-3">
            <div className="w-10 h-10 rounded-lg bg-purple-500/20 flex items-center justify-center group-hover:bg-purple-500/30 transition-colors">
              <FileQuestion className="w-5 h-5 text-purple-400" />
            </div>
            <span className="text-lg font-semibold text-text-primary">Quizzes</span>
          </div>
          <p className="text-sm text-text-secondary">Create and manage AI-powered quizzes</p>
        </Link>
        <Link
          to="/faculty/contests"
          className="group relative p-5 bg-gradient-to-br from-blue-500/10 to-blue-500/5 rounded-xl border-2 border-blue-500/20 hover:border-blue-500/40 transition-all"
        >
          <div className="flex items-center gap-3 mb-3">
            <div className="w-10 h-10 rounded-lg bg-blue-500/20 flex items-center justify-center group-hover:bg-blue-500/30 transition-colors">
              <Calendar className="w-5 h-5 text-blue-400" />
            </div>
            <span className="text-lg font-semibold text-text-primary">Contests</span>
          </div>
          <p className="text-sm text-text-secondary">Manage coding contests and competitions</p>
        </Link>
      </div>

      {/* My Courses */}
      <div className="bg-background-secondary rounded-xl border-2 border-background-border p-6">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl font-semibold text-text-primary">My Courses</h2>
          <span className="text-text-muted text-sm">
            {courses.length} course{courses.length !== 1 ? 's' : ''} assigned
          </span>
        </div>

        {courses.length === 0 ? (
          <div className="text-center py-16">
            <BookOpen className="w-16 h-16 mx-auto mb-4 text-text-muted" />
            <h3 className="text-xl font-semibold text-text-primary mb-2">No courses assigned</h3>
            <p className="text-text-muted max-w-md mx-auto">
              You haven't been assigned to any courses yet. Please contact your administrator or HOD
              to get course assignments.
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {courses.map((course) => (
              <Link
                key={course.id || course.ID}
                to={`/faculty/course/${course.id || course.ID}/dashboard`}
                className="group relative p-5 bg-background-secondary rounded-xl border-2 border-background-border hover:border-background-elevated transition-all overflow-hidden"
              >
                <div className="relative">
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex items-center gap-2">
                      <span className="px-2.5 py-1 bg-blue-500/10 text-blue-400 rounded-md text-xs font-semibold border border-blue-500/20">
                        {course.course_code || course.CourseCode}
                      </span>
                      <span
                        className={`px-2 py-1 text-[10px] font-medium rounded-md border ${
                          (course.course_type || course.CourseType) === 'theory'
                            ? 'bg-purple-500/10 text-purple-400 border-purple-500/20'
                            : 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                        }`}
                      >
                        {course.course_type || course.CourseType}
                      </span>
                    </div>
                    <div className="w-8 h-8 rounded-lg bg-background-tertiary flex items-center justify-center group-hover:bg-background-elevated transition-colors">
                      <span className="text-blue-400 text-sm">→</span>
                    </div>
                  </div>
                  <h4 className="text-lg font-semibold text-text-primary mb-3 group-hover:text-blue-400 transition-colors">
                    {course.course_name || course.CourseName}
                  </h4>
                  <div className="space-y-1.5 text-sm text-text-secondary">
                    {(course.section_name || course.SectionName) && (
                      <div className="flex items-center gap-2">
                        <span className="text-text-muted">Section:</span>
                        <span className="text-text-secondary">
                          {course.section_name || course.SectionName}
                        </span>
                      </div>
                    )}
                    {(course.academic_year || course.AcademicYear) && (
                      <div className="flex items-center gap-2">
                        <span className="text-text-muted">Year:</span>
                        <span className="text-text-secondary">
                          {course.academic_year || course.AcademicYear}
                        </span>
                      </div>
                    )}
                    <div className="flex items-center gap-2">
                      <span className="text-text-muted">Credits:</span>
                      <span className="text-text-secondary">
                        {course.credits || course.Credits}
                      </span>
                    </div>
                  </div>
                </div>
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export default FacultyHome;
