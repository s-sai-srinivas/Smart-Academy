import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { coursesAPI } from '../../services/api';
import { showError } from '../../utils/showAlert';

interface CourseItem {
  id: number;
  course_code: string;
  course_name: string;
  course_type: string;
  credits: number;
  semester?: number;
}

function CoursesPage() {
  const [courses, setCourses] = useState<CourseItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadCourses();
  }, []);

  const loadCourses = async () => {
    try {
      const data = (await coursesAPI.getAll()) as CourseItem[];
      setCourses(data || []);
    } catch (err) {
      console.error('Failed to load courses:', err);
      showError('Failed to load courses');
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <p className="text-text-muted">Loading courses...</p>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-7xl mx-auto">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-text-primary mb-2">My Courses</h1>
        <p className="text-text-secondary">Select a course to view theory and lab content</p>
      </div>

      {/* Courses Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
        {courses.length === 0 ? (
          <div className="col-span-full text-center py-16">
            <div className="text-6xl mb-4">📚</div>
            <h3 className="text-xl font-semibold text-text-primary mb-2">No Courses Available</h3>
            <p className="text-text-secondary">Check back later for new courses</p>
          </div>
        ) : (
          courses.map((course) => (
            <Link key={course.id} to={`/courses/${course.id}`} className="group block">
              <div className="card h-full hover:border-accent-secondary/50 transition-all">
                <div className="flex items-start gap-4">
                  {/* Course Icon */}
                  <div className="w-16 h-16 rounded-xl bg-accent-secondary/10 flex items-center justify-center text-accent-secondary font-bold text-sm group-hover:bg-accent-secondary group-hover:text-white transition-all">
                    {course.course_code}
                  </div>

                  {/* Course Content */}
                  <div className="flex-1 min-w-0">
                    <h3 className="text-lg font-semibold text-text-primary mb-2 group-hover:text-accent-secondary transition-colors line-clamp-1">
                      {course.course_name}
                    </h3>
                    <div className="flex items-center gap-3 text-sm mb-2">
                      <span
                        className={`badge ${course.course_type === 'lab' ? 'badge-success' : 'badge-info'}`}
                      >
                        {course.course_type === 'lab' ? 'Lab' : 'Theory'}
                      </span>
                      <span className="text-text-secondary">{course.credits} Credits</span>
                    </div>
                    {course.semester && (
                      <p className="text-text-tertiary text-sm">Semester {course.semester}</p>
                    )}
                  </div>

                  {/* Arrow */}
                  <div className="text-accent-secondary text-xl group-hover:translate-x-1 transition-transform">
                    →
                  </div>
                </div>
              </div>
            </Link>
          ))
        )}
      </div>
    </div>
  );
}

export default CoursesPage;
