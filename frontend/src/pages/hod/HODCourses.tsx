import { useState, useEffect, useCallback, type ChangeEvent } from 'react';
import api from '../../services/api';
import { showError } from '../../utils/showAlert';
import StudentProfileCard from '../../components/common/StudentProfileCard';
import type { StudentAnalytics } from '../../components/common/StudentProfileCard';

/**
 * HODCourses Page
 *
 * VIEWS (inline navigation, same URL):
 * 1. 'courses' - Grid of all courses
 * 2. 'course-students' - Student list for selected course with section filter
 * 3. 'student-profile' - Individual student analytics
 *
 * NAVIGATION:
 * - Courses → Click course → Course Students view
 * - Course Students → Click student → Student Profile view
 * - Back buttons return to previous view
 */

interface Course {
  id: number | string;
  course_code: string;
  course_name: string;
  lab?: boolean;
  theory?: boolean;
  program: string;
  semester: number | string;
  branches: string;
  enrolled_count?: number;
}

interface Section {
  section_id: number;
  section_name: string;
  student_count?: number;
}

interface CourseAnalyticsResponse {
  sections?: Section[];
}

interface SectionAnalyticsResponse {
  students?: Student[];
}

interface Student {
  regdno: string;
  name: string;
  roll_number?: string;
  problems_solved?: number;
  time_spent_seconds?: number;
  current_streak?: number;
  section_name?: string;
  section_id?: number;
}

type ViewState = 'courses' | 'course-students' | 'student-profile';

const HODCourses = () => {
  // ============== STATE ==============
  // Current view: 'courses' | 'course-students' | 'student-profile'
  const [view, setView] = useState<ViewState>('courses');

  // Data for each view
  const [courses, setCourses] = useState<Course[]>([]);
  const [selectedCourse, setSelectedCourse] = useState<Course | null>(null);
  const [selectedStudent, setSelectedStudent] = useState<Student | null>(null);

  // Course students data
  const [sections, setSections] = useState<Section[]>([]);
  const [allStudents, setAllStudents] = useState<Student[]>([]);
  const [filteredStudents, setFilteredStudents] = useState<Student[]>([]);
  const [selectedSection, setSelectedSection] = useState<string>('all');

  // Student profile data
  const [studentAnalytics, setStudentAnalytics] = useState<StudentAnalytics | null>(null);

  // Loading states
  const [loading, setLoading] = useState<boolean>(true);
  const [studentsLoading, setStudentsLoading] = useState<boolean>(false);
  const [profileLoading, setProfileLoading] = useState<boolean>(false);

  // ============== FETCH FUNCTIONS ==============

  // Fetch all courses on mount
  useEffect(() => {
    fetchCourses();
  }, []);

  // Filter students when section selection changes
  useEffect(() => {
    if (selectedSection === 'all') {
      setFilteredStudents(allStudents);
    } else {
      setFilteredStudents(allStudents.filter((s) => s.section_id === parseInt(selectedSection)));
    }
  }, [selectedSection, allStudents]);

  const fetchCourses = async () => {
    try {
      const response = (await api.get('/hod/courses')) as Course[];
      setCourses(Array.isArray(response) ? response : []);
    } catch (error) {
      console.error('Error fetching courses:', error);
      showError('Failed to load courses');
      setCourses([]);
    } finally {
      setLoading(false);
    }
  };

  // Fetch all students for the course (from all sections)
  const fetchCourseStudents = useCallback(async () => {
    if (!selectedCourse) return;
    setStudentsLoading(true);
    try {
      // Get course analytics which includes sections
      const analyticsResponse = (await api.get(
        `/hod/analytics/course/${selectedCourse.id}`
      )) as CourseAnalyticsResponse;
      setSections(analyticsResponse.sections || []);

      // Fetch students from each section
      const allStudentsData: Student[] = [];
      for (const section of analyticsResponse.sections || []) {
        try {
          const sectionResponse = (await api.get(
            `/hod/analytics/course/${selectedCourse.id}/section/${section.section_id}`
          )) as SectionAnalyticsResponse;
          const students = (sectionResponse.students || []).map((s: Student) => ({
            ...s,
            section_name: section.section_name,
            section_id: section.section_id,
          }));
          allStudentsData.push(...students);
        } catch (err) {
          console.error(`Failed to load students for section ${section.section_name}:`, err);
        }
      }
      setAllStudents(allStudentsData);
      setFilteredStudents(allStudentsData);
    } catch (error) {
      console.error('Error fetching course students:', error);
      showError('Failed to load course students');
    } finally {
      setStudentsLoading(false);
    }
  }, [selectedCourse]);

  // Fetch students when course is selected
  useEffect(() => {
    if (selectedCourse && view === 'course-students') {
      fetchCourseStudents();
    }
  }, [selectedCourse, view, fetchCourseStudents]);

  // Fetch individual student analytics
  const fetchStudentAnalytics = async (student: Student) => {
    setProfileLoading(true);
    try {
      const response = (await api.get(
        `/hod/analytics/student/${student.regdno}`
      )) as StudentAnalytics;
      setStudentAnalytics(response);
    } catch (error) {
      console.error('Error fetching student analytics:', error);
      showError('Failed to load student analytics');
    } finally {
      setProfileLoading(false);
    }
  };

  // ============== EVENT HANDLERS ==============

  const handleCourseClick = (course: Course) => {
    setSelectedCourse(course);
    setSelectedSection('all');
    setView('course-students');
  };

  const handleBackToCourses = () => {
    setView('courses');
    setSelectedCourse(null);
    setAllStudents([]);
    setFilteredStudents([]);
    setSections([]);
  };

  const handleStudentClick = (student: Student) => {
    setSelectedStudent(student);
    setView('student-profile');
    fetchStudentAnalytics(student);
  };

  const handleBackToStudents = () => {
    setView('course-students');
    setSelectedStudent(null);
    setStudentAnalytics(null);
  };

  const handleSectionChange = (e: ChangeEvent<HTMLSelectElement>) => {
    setSelectedSection(e.target.value);
  };

  // ============== HELPER FUNCTIONS ==============

  const formatTime = (seconds?: number): string => {
    if (!seconds) return '0m';
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  };

  // ============== RENDER FUNCTIONS ==============

  // Render View 1: Courses Grid
  const renderCoursesView = () => (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-text-primary mb-1">Courses</h1>
        <p className="text-text-secondary text-sm">Click on a course to view students</p>
      </div>

      {/* Courses Grid */}
      {courses.length === 0 ? (
        <div className="card text-center py-16">
          <p className="text-text-muted">No courses found</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
          {courses.map((course) => (
            <div
              key={course.id}
              onClick={() => handleCourseClick(course)}
              className="card p-4 cursor-pointer transition-all hover:border-accent-secondary/50 hover:shadow-lg"
            >
              <div className="flex items-start justify-between mb-3">
                <div>
                  <span className="text-xl font-bold text-accent-secondary">
                    {course.course_code}
                  </span>
                  <div className="text-text-primary font-medium mt-1">{course.course_name}</div>
                </div>
                <div className="flex gap-2">
                  {course.lab && <span className="badge badge-success text-xs">Lab</span>}
                  {course.theory && <span className="badge badge-info text-xs">Theory</span>}
                </div>
              </div>
              <div className="flex items-center gap-3 text-sm text-text-secondary">
                <span>{course.program}</span>
                <span>•</span>
                <span>Sem {course.semester}</span>
                <span>•</span>
                <span>{course.branches}</span>
              </div>
              <div className="text-sm text-text-secondary mt-2">
                Enrolled Students:{' '}
                <span className="font-semibold text-text-primary">
                  {course.enrolled_count ?? 0}
                </span>
              </div>
              <div className="mt-3 pt-3 border-t border-border-primary text-xs text-accent-primary">
                View students →
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );

  // Render View 2: Course Students List
  const renderCourseStudentsView = () => (
    <div className="space-y-6">
      {/* Header with back button */}
      <div className="flex items-center gap-4">
        <button
          onClick={handleBackToCourses}
          className="text-accent-primary hover:text-accent-secondary flex items-center gap-1"
        >
          ← Back to Courses
        </button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold text-text-primary">
            {selectedCourse?.course_code} - {selectedCourse?.course_name}
          </h1>
          <p className="text-text-secondary text-sm">
            {filteredStudents.length} students{' '}
            {selectedSection !== 'all' && `in Section ${selectedSection}`}
          </p>
        </div>
      </div>

      {/* Section Filter Dropdown */}
      <div className="flex items-center gap-3">
        <label className="text-text-secondary text-sm">Filter by Section:</label>
        <select
          value={selectedSection}
          onChange={handleSectionChange}
          className="px-3 py-2 bg-background-secondary border border-border-primary rounded-lg text-text-primary"
        >
          <option value="all">All Sections ({allStudents.length})</option>
          {sections.map((section) => (
            <option key={section.section_id} value={section.section_id}>
              Section {section.section_name} ({section.student_count} students)
            </option>
          ))}
        </select>
      </div>

      {/* Students Table */}
      {studentsLoading ? (
        <div className="text-center py-12">
          <div className="inline-block animate-spin h-8 w-8 border-4 border-accent-primary border-t-transparent rounded-full"></div>
          <p className="text-text-muted mt-3">Loading students...</p>
        </div>
      ) : filteredStudents.length === 0 ? (
        <div className="card text-center py-12">
          <p className="text-text-muted">No students found</p>
        </div>
      ) : (
        <div className="card overflow-hidden">
          <table className="w-full">
            <thead className="bg-background-secondary">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-medium text-text-secondary">
                  Name
                </th>
                <th className="px-4 py-3 text-left text-sm font-medium text-text-secondary">
                  Section
                </th>
                <th className="px-4 py-3 text-left text-sm font-medium text-text-secondary">
                  Roll Number
                </th>
                <th className="px-4 py-3 text-left text-sm font-medium text-text-secondary">
                  Problems Solved
                </th>
                <th className="px-4 py-3 text-left text-sm font-medium text-text-secondary">
                  Time Spent
                </th>
                <th className="px-4 py-3 text-left text-sm font-medium text-text-secondary">
                  Streak
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border-primary">
              {filteredStudents.map((student) => (
                <tr
                  key={student.regdno}
                  onClick={() => handleStudentClick(student)}
                  className="hover:bg-background-tertiary cursor-pointer transition-colors"
                >
                  <td className="px-4 py-3 text-text-primary">{student.name}</td>
                  <td className="px-4 py-3">
                    <span className="px-2 py-1 bg-accent-primary/10 text-accent-primary rounded text-xs">
                      {student.section_name}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-text-secondary">{student.roll_number || '-'}</td>
                  <td className="px-4 py-3 text-text-primary font-medium">
                    {student.problems_solved || 0}
                  </td>
                  <td className="px-4 py-3 text-text-secondary">
                    {formatTime(student.time_spent_seconds)}
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className={`px-2 py-1 rounded text-xs ${student.current_streak && student.current_streak > 0 ? 'bg-success/10 text-success' : 'bg-text-muted/10 text-text-muted'}`}
                    >
                      {student.current_streak || 0} days
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );

  // Render View 3: Student Profile (uses shared component)
  const renderStudentProfileView = () => (
    <StudentProfileCard
      studentAnalytics={studentAnalytics}
      studentInfo={{
        name: selectedStudent?.name,
        roll_number: selectedStudent?.roll_number,
        section_name: selectedStudent?.section_name,
      }}
      loading={profileLoading}
      onBack={handleBackToStudents}
      backLabel="← Back to Students"
    />
  );

  // ============== MAIN RENDER ==============

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="flex items-center gap-3">
          <div className="animate-spin h-5 w-5 border-4 border-accent-primary border-t-transparent rounded-full"></div>
          <span className="text-text-secondary">Loading courses...</span>
        </div>
      </div>
    );
  }

  // Render based on current view
  return (
    <div className="p-6">
      {view === 'courses' && renderCoursesView()}
      {view === 'course-students' && renderCourseStudentsView()}
      {view === 'student-profile' && renderStudentProfileView()}
    </div>
  );
};

export default HODCourses;
