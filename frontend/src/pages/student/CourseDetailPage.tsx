import { useState, useEffect, useCallback } from 'react';
import { useParams, Link } from 'react-router-dom';
import { coursesAPI, default as apiInstance } from '../../services/api';
import { showError } from '../../utils/showAlert';
import Breadcrumb from '../../components/common/Breadcrumb';
import StudentTheoryModulePage from './StudentTheoryModulePage';
import StudentTheoryQuizPage from './StudentTheoryQuizPage';

interface CourseData {
  course_name: string;
  course_code: string;
  course_type: string;
  credits: number;
  semester?: number;
}

interface LabSessionItem {
  id: number;
  session_name?: string;
  title?: string;
  topic_name?: string;
}

interface TheoryModuleItem {
  id: number;
  module_name?: string;
  title?: string;
  description?: string;
}

interface TheoryWeekItem {
  id: number;
  week_order?: number;
  week_name: string;
  modules?: TheoryModuleItem[];
}

interface TheoryPDF {
  id: number;
  theory_id: number;
  theory_module_id?: number;
  file_name: string;
  display_name?: string;
  file_size: number;
  created_at?: string;
}

function CourseDetailPage() {
  const { courseId } = useParams<{ courseId: string }>();
  const [course, setCourse] = useState<CourseData | null>(null);
  const [labSessions, setLabSessions] = useState<LabSessionItem[]>([]);
  const [theoryWeeks, setTheoryWeeks] = useState<TheoryWeekItem[]>([]);
  const [theoryId, setTheoryId] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [theoryPdfs, setTheoryPdfs] = useState<TheoryPDF[]>([]);
  const [pdfsLoading, setPdfsLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<'modules' | 'pdfs'>('modules');

  // State-based navigation (no new routes)
  const [currentView, setCurrentView] = useState<'list' | 'module' | 'quiz'>('list');
  const [selectedModule, setSelectedModule] = useState<TheoryModuleItem | null>(null);
  const [selectedWeek, setSelectedWeek] = useState<TheoryWeekItem | null>(null);

  const loadCourseData = useCallback(async () => {
    try {
      if (!courseId) {
        setCourse(null);
        setLabSessions([]);
        setTheoryWeeks([]);
        return;
      }
      const data = (await coursesAPI.getById(courseId)) as {
        course: CourseData;
        lab_sessions?: LabSessionItem[];
        theory_weeks?: TheoryWeekItem[];
        theory_id?: number;
      };
      setCourse(data.course);
      setLabSessions(data.lab_sessions || []);
      setTheoryWeeks(data.theory_weeks || []);
      setTheoryId(data.theory_id || null);
    } catch (err) {
      console.error('Failed to load course:', err);
      showError('Failed to load course');
    } finally {
      setLoading(false);
    }
  }, [courseId]);

  useEffect(() => {
    loadCourseData();
  }, [loadCourseData]);

  useEffect(() => {
    if (!theoryId) return;

    const fetchPDFs = async () => {
      setPdfsLoading(true);
      try {
        const response = (await coursesAPI.getTheoryPDFs(theoryId)) as { pdfs?: TheoryPDF[] };
        setTheoryPdfs(response.pdfs || []);
      } catch (err) {
        console.error('Failed to load PDFs:', err);
        showError('Failed to load PDFs');
      } finally {
        setPdfsLoading(false);
      }
    };

    fetchPDFs();
  }, [theoryId]);

  const openModule = (week: TheoryWeekItem, module: TheoryModuleItem) => {
    setSelectedWeek(week);
    setSelectedModule(module);
    setCurrentView('module');
  };

  const openQuiz = (week: TheoryWeekItem) => {
    setSelectedWeek(week);
    setCurrentView('quiz');
  };

  const goBack = () => {
    setCurrentView('list');
    setSelectedModule(null);
    setSelectedWeek(null);
  };

  const handlePdfOpen = async (pdfId: number) => {
    try {
      const blob = (await apiInstance.get(`/theory-pdfs/${pdfId}`, {
        responseType: 'blob',
      })) as Blob;
      const url = window.URL.createObjectURL(blob);
      window.open(url, '_blank');
      setTimeout(() => window.URL.revokeObjectURL(url), 120000);
    } catch (err) {
      console.error('Failed to open PDF:', err);
      showError('Failed to open PDF');
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <p className="text-text-muted">Loading course...</p>
      </div>
    );
  }

  if (!course) {
    return (
      <div className="p-6 max-w-4xl mx-auto">
        <div className="text-accent-danger mb-4">Course not found</div>
        <Link to="/courses" className="text-accent-secondary hover:underline">
          ← Back to Courses
        </Link>
      </div>
    );
  }

  const isLab = course.course_type === 'lab' || course.course_type === 'integrated';
  const courseTypeLabel = isLab ? 'Lab' : 'Theory';
  const courseTypeBadgeClass = isLab ? 'badge-success' : 'badge-info';
  const referencePdfs = theoryPdfs.filter((pdf) => !pdf.theory_module_id);

  const renderPdfList = (items: TheoryPDF[], emptyMessage: string) => {
    if (items.length === 0) {
      return <p className="text-text-muted text-sm italic">{emptyMessage}</p>;
    }

    return (
      <div className="space-y-3">
        {items.map((pdf) => (
          <button
            key={pdf.id}
            type="button"
            onClick={() => handlePdfOpen(pdf.id)}
            className="w-full flex items-center gap-3 p-3 bg-background-tertiary rounded-lg border border-background-border hover:border-accent-secondary/50 hover:bg-background-elevated transition-all text-left"
          >
            <div className="w-10 h-10 rounded-lg bg-red-500/10 flex items-center justify-center text-red-400 flex-shrink-0">
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
                />
              </svg>
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-text-primary text-sm font-medium truncate">
                {pdf.display_name || pdf.file_name}
              </p>
              <p className="text-text-muted text-xs">{(pdf.file_size / 1024).toFixed(1)} KB</p>
            </div>
            <svg
              className="w-4 h-4 text-text-muted"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
              />
            </svg>
          </button>
        ))}
      </div>
    );
  };

  // ── Module content view ──────────────────────────────────────────────────
  if (currentView === 'module' && selectedModule && selectedWeek) {
    return (
      <StudentTheoryModulePage
        course={course}
        courseId={courseId || ''}
        week={selectedWeek}
        module={selectedModule}
        theoryWeeks={theoryWeeks}
        theoryId={theoryId}
        onBack={goBack}
        onNavigateModule={openModule}
      />
    );
  }

  // ── Quiz view ────────────────────────────────────────────────────────────
  if (currentView === 'quiz' && selectedWeek) {
    return (
      <StudentTheoryQuizPage
        course={course}
        courseId={courseId || ''}
        week={selectedWeek}
        theoryWeeks={theoryWeeks}
        onBack={goBack}
        onNavigateModule={openModule}
      />
    );
  }

  // ── List view (default) ──────────────────────────────────────────────────
  return (
    <div className="p-6 max-w-4xl mx-auto">
      {/* Breadcrumb Navigation */}
      <Breadcrumb
        items={[
          { label: 'Dashboard', to: '/dashboard' },
          { label: 'Courses', to: '/courses' },
          { label: course.course_name },
        ]}
      />

      {/* Course Header */}
      <div className="card mb-6">
        <div className="flex items-start gap-5">
          <div className="w-20 h-20 rounded-xl bg-accent-secondary/10 flex items-center justify-center text-accent-secondary font-bold text-base">
            {course.course_code}
          </div>
          <div className="flex-1">
            <h1 className="text-2xl font-bold text-text-primary mb-3">{course.course_name}</h1>
            <div className="flex items-center gap-3 text-sm text-text-secondary">
              <span className={`badge ${courseTypeBadgeClass}`}>{courseTypeLabel}</span>
              <span>{course.credits} Credits</span>
              {course.semester && <span>Semester {course.semester}</span>}
            </div>
          </div>
        </div>
      </div>

      {/* Content based on course category */}
      <div className="card">
        {isLab ? (
          // Lab Course Content
          <div>
            <h3 className="text-lg font-semibold text-text-primary mb-4">Lab Sessions</h3>
            {labSessions.length === 0 ? (
              <div className="text-center py-16">
                <h3 className="text-lg font-semibold text-text-primary mb-2">No Lab Sessions</h3>
                <p className="text-text-secondary">No lab sessions available yet.</p>
              </div>
            ) : (
              <div className="space-y-3">
                {labSessions.map((session, index) => (
                  <Link
                    key={session.id}
                    to={`/courses/${courseId}/lab/${session.id}`}
                    className="group flex items-center gap-4 p-4 bg-background-tertiary rounded-lg border border-background-border hover:border-accent-secondary/50 hover:bg-background-elevated transition-all"
                  >
                    <div className="w-10 h-10 rounded-full bg-accent-secondary/10 flex items-center justify-center text-accent-secondary font-semibold text-sm">
                      {index + 1}
                    </div>
                    <div className="flex-1">
                      <h4 className="font-medium text-text-primary group-hover:text-accent-secondary transition-colors">
                        {session.session_name || session.title || `Session ${index + 1}`}
                      </h4>
                      {session.topic_name && (
                        <p className="text-sm text-text-secondary">{session.topic_name}</p>
                      )}
                    </div>
                    <div className="text-accent-secondary text-sm font-medium group-hover:translate-x-1 transition-transform">
                      View
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </div>
        ) : (
          // Theory Course Content
          <div>
            <div className="flex flex-col gap-4 border-b border-background-border pb-4 mb-6 sm:flex-row sm:items-center sm:justify-between">
              <h3 className="text-lg font-semibold text-text-primary">Theory Modules</h3>
              <div className="flex gap-1 bg-background-tertiary rounded-lg p-1 w-full sm:w-auto">
                <button
                  onClick={() => setActiveTab('modules')}
                  className={`flex-1 sm:flex-none px-4 py-2 text-sm font-medium rounded-md transition-colors ${
                    activeTab === 'modules'
                      ? 'bg-background-elevated text-text-primary shadow-sm'
                      : 'text-text-secondary hover:text-text-primary'
                  }`}
                >
                  Modules
                  <span className="ml-1.5 px-1.5 py-0.5 text-xs rounded-full bg-accent-primary/20 text-accent-primary">
                    {theoryWeeks.reduce((count, week) => count + (week.modules?.length || 0), 0)}
                  </span>
                </button>
                <button
                  onClick={() => setActiveTab('pdfs')}
                  className={`flex-1 sm:flex-none px-4 py-2 text-sm font-medium rounded-md transition-colors ${
                    activeTab === 'pdfs'
                      ? 'bg-background-elevated text-text-primary shadow-sm'
                      : 'text-text-secondary hover:text-text-primary'
                  }`}
                >
                  PDFs
                  <span className="ml-1.5 px-1.5 py-0.5 text-xs rounded-full bg-accent-primary/20 text-accent-primary">
                    {referencePdfs.length}
                  </span>
                </button>
              </div>
            </div>
            {theoryWeeks.length === 0 ? (
              <div className="text-center py-16">
                <h3 className="text-lg font-semibold text-text-primary mb-2">No Theory Content</h3>
                <p className="text-text-secondary">Theory modules will be available here soon.</p>
              </div>
            ) : activeTab === 'modules' ? (
              <div className="space-y-4">
                {theoryWeeks.map((week, index) => (
                  <div
                    key={week.id}
                    className="rounded-lg border border-background-border overflow-hidden"
                  >
                    {/* Week header */}
                    <div className="flex items-center justify-between px-4 py-3 bg-background-tertiary border-b border-background-border">
                      <h4 className="font-semibold text-text-primary">
                        Week {week.week_order || index + 1}: {week.week_name}
                      </h4>
                      <button
                        onClick={() => openQuiz(week)}
                        className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium
                                   bg-accent-secondary/10 text-accent-secondary border border-accent-secondary/20
                                   hover:bg-accent-secondary/20 transition-colors"
                      >
                        📝 Take Quiz
                      </button>
                    </div>

                    {/* Modules list */}
                    {week.modules && week.modules.length > 0 ? (
                      <div className="divide-y divide-background-border">
                        {week.modules.map((module) => (
                          <button
                            key={module.id}
                            onClick={() => openModule(week, module)}
                            className="w-full group flex items-center gap-4 px-4 py-3
                                       hover:bg-background-elevated transition-colors text-left"
                          >
                            <div className="w-8 h-8 rounded-md bg-accent-primary/10 flex items-center justify-center text-accent-primary flex-shrink-0">
                              📄
                            </div>
                            <div className="flex-1 min-w-0">
                              <div className="font-medium text-text-primary group-hover:text-accent-secondary transition-colors truncate">
                                {module.module_name || module.title}
                              </div>
                              {module.description && (
                                <p className="text-xs text-text-secondary truncate mt-0.5">
                                  {module.description}
                                </p>
                              )}
                            </div>
                            <div className="text-text-tertiary group-hover:text-accent-secondary group-hover:translate-x-0.5 transition-all text-sm flex-shrink-0">
                              →
                            </div>
                          </button>
                        ))}
                      </div>
                    ) : (
                      <p className="text-text-secondary text-sm italic px-4 py-3">No modules yet</p>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <div className="card">
                <h4 className="text-lg font-semibold text-text-primary mb-4 flex items-center gap-2">
                  <span>📎</span>
                  Reference PDFs
                </h4>
                {renderPdfList(referencePdfs, 'No reference PDFs available yet.')}
              </div>
            )}
          </div>
        )}

        {pdfsLoading && (
          <div className="card mt-6 text-center py-8">
            <div className="inline-block animate-spin rounded-full h-5 w-5 border-b-2 border-accent-primary"></div>
            <p className="text-text-secondary text-sm mt-2">Loading PDFs...</p>
          </div>
        )}
      </div>
    </div>
  );
}

export default CourseDetailPage;
