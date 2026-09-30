import { useState, useEffect, ChangeEvent, FormEvent, MouseEvent } from 'react';
import api from '../../services/api';
import { showSuccess, showError } from '../../utils/showAlert';
import { getErrorMessage } from '../../utils/error';
import './CourseCreationDashboard.css';

// ─── Interfaces ──────────────────────────────────────────────────────────────

interface Program {
  program_id: number;
  program_name: string;
}

interface Branch {
  branch_id: number;
  branch_name: string;
  program_id: number;
}

interface Course {
  id: number;
  course_code: string;
  course_name: string;
  course_type: string;
  semester?: {
    program?: {
      program_name: string;
    };
  };
  branch?: {
    branch_name: string;
  };
}

interface Lab {
  lab_id: number;
  lab_name: string;
  lab_code: string;
  description?: string;
  course?: {
    course_name: string;
  };
  course_id?: number;
}

interface Theory {
  theory_id: number;
  theory_name: string;
  theory_code: string;
  course?: {
    course_name: string;
  };
  weeks?: Week[];
}

interface Session {
  session_id: number;
  title: string;
  topic: string;
  start_time: string;
}

interface Problem {
  id: number;
  title: string;
  difficulty: string;
}

interface ModuleItem {
  module_id: number;
  title: string;
  description?: string;
  content?: string;
  order_index?: number;
}

interface Week {
  week_id: number;
  week_number: number;
  title: string;
  description?: string;
  modules?: ModuleItem[];
}

interface TestCase {
  input: string;
  expected_output: string;
  is_sample: boolean;
  points: number;
}

interface CourseFormData {
  course_name: string;
  course_code: string;
  program_id: string;
  branch_ids: number[];
  course_type: string;
  credits: number;
  semester_id: number;
}

interface LabFormData {
  lab_name: string;
  lab_code: string;
  description: string;
  course_id: string;
}

interface TheoryFormData {
  theory_name: string;
  theory_code: string;
  course_id: string;
}

interface SessionFormData {
  title: string;
  topic: string;
  lab_id: number | undefined;
  course_id: number | undefined;
  start_time: string;
  end_time: string;
  max_attempts: number;
}

interface WeekFormData {
  week_number: number;
  title: string;
  description: string;
}

interface ModuleFormData {
  title: string;
  description: string;
  content: string;
  order_index: number;
}

interface ProblemFormData {
  title: string;
  description: string;
  difficulty: string;
  time_limit: number;
  memory_limit: number;
  test_cases: TestCase[];
}

// ─── Main Component ──────────────────────────────────────────────────────────

const CourseCreationDashboard = () => {
  const [programs, setPrograms] = useState<Program[]>([]);
  const [branches, setBranches] = useState<Branch[]>([]);
  const [courses, setCourses] = useState<Course[]>([]);
  const [labs, setLabs] = useState<Lab[]>([]);
  const [theories, setTheories] = useState<Theory[]>([]);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [problems, setProblems] = useState<Problem[]>([]);

  // Currently selected items for drilling down
  const [, setSelectedCourse] = useState<Lab | null>(null);
  const [selectedLab, setSelectedLab] = useState<Lab | null>(null);
  const [selectedTheory, setSelectedTheory] = useState<Theory | null>(null);
  const [selectedWeek, setSelectedWeek] = useState<Week | null>(null);
  const [selectedSession, setSelectedSession] = useState<Session | null>(null);

  // View state: 'courses', 'labs', 'theory', 'lab-detail', 'theory-detail', etc.
  const [currentView, setCurrentView] = useState<string>('courses');

  // Listen for navigation events from parent AdminDashboard
  useEffect(() => {
    const handleNavChange = (event: CustomEvent) => {
      const view = event.detail;
      if (['courses', 'labs', 'theory'].includes(view)) {
        setCurrentView(view);
        window.dispatchEvent(new CustomEvent('admin-view-change', { detail: view }));
      }
    };

    window.addEventListener('admin-nav-change', handleNavChange as EventListener);
    return () => window.removeEventListener('admin-nav-change', handleNavChange as EventListener);
  }, []);

  // Helper function to change view and notify parent
  const changeView = (view: string) => {
    setCurrentView(view);
    if (['courses', 'labs', 'theory'].includes(view)) {
      window.dispatchEvent(new CustomEvent('admin-view-change', { detail: view }));
    }
  };

  const [showModal, setShowModal] = useState(false);
  const [modalType, setModalType] = useState<string>('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetchPrograms();
    fetchBranches();
    fetchCourses();
    fetchLabs();
    fetchTheories();
  }, []);

  const fetchPrograms = async () => {
    try {
      const response = (await api.get('/programs')) as Program[];
      setPrograms(response);
    } catch (error) {
      console.error('Error fetching programs:', error);
    }
  };

  const fetchBranches = async () => {
    try {
      const response = (await api.get('/branches')) as Branch[];
      setBranches(response);
    } catch (error) {
      console.error('Error fetching branches:', error);
    }
  };

  const fetchCourses = async () => {
    try {
      const response = (await api.get('/admin/courses')) as Course[];
      setCourses(response);
    } catch (error) {
      console.error('Error fetching courses:', error);
    }
  };

  const fetchLabs = async () => {
    try {
      const response = (await api.get('/admin/courses')) as Lab[]; // Will fetch from course details
      setLabs(response);
    } catch (error) {
      console.error('Error fetching labs:', error);
    }
  };

  const fetchTheories = async () => {
    try {
      const response = (await api.get('/admin/theories')) as Theory[];
      setTheories(response);
    } catch (error) {
      console.error('Error fetching theories:', error);
    }
  };

  const fetchSessions = async (labId: number) => {
    try {
      const response = (await api.get(`/admin/labs/${labId}/sessions`)) as Session[];
      setSessions(response);
    } catch (error) {
      console.error('Error fetching sessions:', error);
    }
  };

  const fetchTheoryWeeks = async (theoryId: number) => {
    try {
      const response = (await api.get(`/admin/theories/${theoryId}/weeks`)) as Week[];
      return response;
    } catch (error) {
      console.error('Error fetching theory weeks:', error);
      return [];
    }
  };

  const fetchSessionProblems = async (sessionId: number) => {
    try {
      const response = (await api.get(`/admin/lab-sessions/${sessionId}/problems`)) as Problem[];
      setProblems(response);
    } catch (error) {
      console.error('Error fetching problems:', error);
    }
  };

  const openModal = (type: string) => {
    setModalType(type);
    setShowModal(true);
  };

  const closeModal = () => {
    setShowModal(false);
    setModalType('');
  };

  const handleCreate = async (
    formData:
      | CourseFormData
      | LabFormData
      | TheoryFormData
      | SessionFormData
      | WeekFormData
      | ModuleFormData
      | ProblemFormData
  ) => {
    setLoading(true);
    try {
      switch (modalType) {
        case 'course':
          await api.post('/admin/courses', formData);
          fetchCourses();
          break;
        case 'lab':
          await api.post('/admin/labs', formData);
          fetchLabs();
          break;
        case 'theory':
          await api.post('/admin/theories', formData);
          fetchTheories();
          break;
        case 'session':
          await api.post('/admin/lab-sessions', formData);
          if (selectedLab) {
            fetchSessions(selectedLab.lab_id);
          }
          break;
        case 'week':
          if (!selectedTheory) {
            showError('Please select a theory before adding a week');
            return;
          }
          await api.post(`/admin/theories/${selectedTheory.theory_id}/weeks`, formData);
          {
            const weeks = await fetchTheoryWeeks(selectedTheory.theory_id);
            setSelectedTheory({ ...selectedTheory, weeks });
          }
          break;
        case 'module':
          if (!selectedWeek || !selectedTheory) {
            showError('Please select a week before adding a module');
            return;
          }
          await api.post(`/admin/weeks/${selectedWeek.week_id}/modules`, formData);
          {
            const weeks = await fetchTheoryWeeks(selectedTheory.theory_id);
            const updatedWeek = weeks.find((w) => w.week_id === selectedWeek.week_id);
            if (updatedWeek) {
              setSelectedWeek(updatedWeek);
            }
          }
          break;
        case 'problem':
          if (!selectedSession) {
            showError('Please select a session before adding a problem');
            return;
          }
          await api.post('/admin/problems', formData);
          fetchSessionProblems(selectedSession.session_id);
          break;
      }
      closeModal();
      showSuccess(
        `${modalType.charAt(0).toUpperCase() + modalType.slice(1)} created successfully!`
      );
    } catch (error: unknown) {
      const message = getErrorMessage(error, `Failed to create ${modalType}`);
      console.error(`Error creating ${modalType}:`, error);
      showError(message);
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (type: string, id: number) => {
    if (!confirm(`Are you sure you want to delete this ${type}?`)) return;
    try {
      await api.delete(`/admin/${type}s/${id}`);
      switch (type) {
        case 'course':
          fetchCourses();
          setSelectedCourse(null);
          break;
        case 'lab':
          fetchLabs();
          setSelectedLab(null);
          break;
        case 'theory':
          fetchTheories();
          setSelectedTheory(null);
          break;
        case 'session':
          if (selectedLab) {
            fetchSessions(selectedLab.lab_id);
          }
          setSelectedSession(null);
          break;
      }
    } catch (error) {
      console.error(`Error deleting ${type}:`, error);
      showError(`Failed to delete ${type}`);
    }
  };

  // Render Courses view
  const renderCoursesView = () => (
    <div className="courses-view">
      <div className="section-header">
        <h2>Courses</h2>
        <button className="btn-primary" onClick={() => openModal('course')}>
          Create Course
        </button>
      </div>

      {courses.length === 0 ? (
        <div className="empty-state">
          <div className="empty-icon"></div>
          <h3>No courses yet</h3>
          <p>Create your first course to get started</p>
          <button className="btn-primary" onClick={() => openModal('course')}>
            Create Course
          </button>
        </div>
      ) : (
        <div className="courses-grid">
          {courses.map((course) => (
            <div key={course.id} className="course-card">
              <div className="course-header">
                <span className="course-code">{course.course_code}</span>
                <span className={`badge badge-${course.course_type}`}>{course.course_type}</span>
              </div>
              <h3 className="course-title">{course.course_name}</h3>
              <div className="course-meta">
                <span>Program: {course.semester?.program?.program_name || 'N/A'}</span>
                <span>Branch: {course.branch?.branch_name || 'N/A'}</span>
              </div>
            </div>
          ))}
        </div>
      )}

      {showModal && modalType === 'course' && (
        <CourseFormModal
          onClose={closeModal}
          onSubmit={handleCreate}
          programs={programs}
          branches={branches}
          loading={loading}
        />
      )}
    </div>
  );

  // Render Labs view
  const renderLabsView = () => (
    <div className="labs-view">
      <div className="section-header">
        <h2>Labs</h2>
        <button className="btn-primary" onClick={() => openModal('lab')}>
          Create Lab
        </button>
      </div>

      {labs.length === 0 ? (
        <div className="empty-state">
          <div className="empty-icon"></div>
          <h3>No labs yet</h3>
          <p>Create labs for your courses</p>
          <button className="btn-primary" onClick={() => openModal('lab')}>
            Create Lab
          </button>
        </div>
      ) : (
        <div className="items-list">
          {labs.map((lab) => (
            <div key={lab.lab_id} className="item-card">
              <div className="item-header">
                <h3>{lab.lab_name}</h3>
                <span className="item-code">{lab.lab_code}</span>
              </div>
              {lab.description && <p className="item-description">{lab.description}</p>}
              <div className="item-meta">
                <span>Course: {lab.course?.course_name || 'N/A'}</span>
              </div>
              <button
                className="btn-view-detail"
                onClick={() => {
                  setSelectedLab(lab);
                  changeView('lab-detail');
                }}
              >
                View Lab
              </button>
              <button className="btn-icon" onClick={() => handleDelete('lab', lab.lab_id)}>
                Delete
              </button>
            </div>
          ))}
        </div>
      )}

      {showModal && modalType === 'lab' && (
        <LabFormModal
          onClose={closeModal}
          onSubmit={handleCreate}
          courses={courses}
          loading={loading}
        />
      )}
    </div>
  );

  // Render Theory view
  const renderTheoryView = () => (
    <div className="theory-view">
      <div className="section-header">
        <h2>Theory</h2>
        <button className="btn-primary" onClick={() => openModal('theory')}>
          Create Theory
        </button>
      </div>

      {theories.length === 0 ? (
        <div className="empty-state">
          <div className="empty-icon"></div>
          <h3>No theories yet</h3>
          <p>Create theory content for your courses</p>
          <button className="btn-primary" onClick={() => openModal('theory')}>
            Create Theory
          </button>
        </div>
      ) : (
        <div className="items-list">
          {theories.map((theory) => (
            <div key={theory.theory_id} className="item-card">
              <div className="item-header">
                <h3>{theory.theory_name}</h3>
                <span className="item-code">{theory.theory_code}</span>
              </div>
              <div className="item-meta">
                <span>Course: {theory.course?.course_name || 'N/A'}</span>
              </div>
              <button
                className="btn-view-detail"
                onClick={() => {
                  setSelectedTheory(theory);
                  changeView('theory-detail');
                }}
              >
                View Theory
              </button>
              <button className="btn-icon" onClick={() => handleDelete('theory', theory.theory_id)}>
                Delete
              </button>
            </div>
          ))}
        </div>
      )}

      {showModal && modalType === 'theory' && (
        <TheoryFormModal
          onClose={closeModal}
          onSubmit={handleCreate}
          courses={courses}
          loading={loading}
        />
      )}
    </div>
  );

  // Render Lab Detail view
  const renderLabDetailView = () => {
    if (!selectedLab) return null;
    return (
      <div className="lab-detail-view">
        <div className="breadcrumb">
          <button
            onClick={() => {
              changeView('labs');
              setSelectedLab(null);
            }}
          >
            Back to Labs
          </button>
          <span>{selectedLab.lab_name}</span>
        </div>

        <div className="detail-header">
          <h1>{selectedLab.lab_name}</h1>
          <span className="detail-code">{selectedLab.lab_code}</span>
          {selectedLab.description && (
            <p className="detail-description">{selectedLab.description}</p>
          )}
          <div className="detail-meta">
            <span>Course: {selectedLab.course?.course_name || 'N/A'}</span>
          </div>
        </div>

        <div className="sessions-section">
          <div className="section-header">
            <h2>Sessions</h2>
            <button className="btn-primary" onClick={() => openModal('session')}>
              Add Session
            </button>
          </div>

          {sessions.length === 0 ? (
            <div className="empty-state">
              <div className="empty-icon"></div>
              <h3>No sessions yet</h3>
              <p>Create sessions for this lab</p>
              <button className="btn-primary" onClick={() => openModal('session')}>
                Add Session
              </button>
            </div>
          ) : (
            <div className="sessions-list">
              {sessions.map((session) => (
                <div key={session.session_id} className="session-item">
                  <div className="session-header">
                    <h4>{session.title}</h4>
                    <span className="session-topic">Topic: {session.topic}</span>
                  </div>
                  <div className="session-meta">
                    <span>Start: {new Date(session.start_time).toLocaleString()}</span>
                  </div>
                  <button
                    className="btn-view-problems"
                    onClick={() => {
                      setSelectedSession(session);
                      changeView('session-detail');
                    }}
                  >
                    View Problems
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        {showModal && modalType === 'session' && (
          <SessionFormModal
            onClose={closeModal}
            onSubmit={handleCreate}
            labId={selectedLab.lab_id}
            courseId={selectedLab.course_id}
            loading={loading}
          />
        )}
      </div>
    );
  };

  // Render Theory Detail view
  const renderTheoryDetailView = () => {
    if (!selectedTheory) return null;
    return (
      <div className="theory-detail-view">
        <div className="breadcrumb">
          <button
            onClick={() => {
              changeView('theory');
              setSelectedTheory(null);
            }}
          >
            Back to Theory
          </button>
          <span>{selectedTheory.theory_name}</span>
        </div>

        <div className="detail-header">
          <h1>{selectedTheory.theory_name}</h1>
          <span className="detail-code">{selectedTheory.theory_code}</span>
          <div className="detail-meta">
            <span>Course: {selectedTheory.course?.course_name || 'N/A'}</span>
          </div>
        </div>

        <div className="weeks-section">
          <div className="section-header">
            <h2>Weeks</h2>
            <button className="btn-primary" onClick={() => openModal('week')}>
              Add Week
            </button>
          </div>

          {!selectedTheory.weeks || selectedTheory.weeks.length === 0 ? (
            <div className="empty-state">
              <div className="empty-icon"></div>
              <h3>No weeks yet</h3>
              <p>Create weeks for this theory</p>
              <button className="btn-primary" onClick={() => openModal('week')}>
                Add Week
              </button>
            </div>
          ) : (
            <div className="weeks-list">
              {selectedTheory.weeks.map((week) => (
                <div key={week.week_id} className="week-item">
                  <div className="week-header">
                    <h4>
                      Week {week.week_number}: {week.title}
                    </h4>
                    <button className="btn-icon" onClick={() => handleDelete('week', week.week_id)}>
                      Delete
                    </button>
                  </div>
                  {week.description && <p className="week-description">{week.description}</p>}
                  <div className="modules-section">
                    <h5>Modules ({week.modules?.length || 0})</h5>
                    <button
                      className="btn-small"
                      onClick={() => {
                        setSelectedWeek(week);
                        openModal('module');
                      }}
                    >
                      + Add Module
                    </button>
                    {week.modules &&
                      week.modules.map((module) => (
                        <div key={module.module_id} className="module-item">
                          <strong>{module.title}</strong>
                          <button
                            className="btn-icon btn-small"
                            onClick={() => handleDelete('module', module.module_id)}
                          >
                            Delete
                          </button>
                        </div>
                      ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {showModal && modalType === 'week' && (
          <WeekFormModal
            onClose={closeModal}
            onSubmit={handleCreate}
            theoryId={selectedTheory.theory_id}
            loading={loading}
          />
        )}

        {showModal && modalType === 'module' && selectedWeek && (
          <ModuleFormModal
            onClose={closeModal}
            onSubmit={handleCreate}
            weekId={selectedWeek.week_id}
            loading={loading}
          />
        )}
      </div>
    );
  };

  // Render Session Detail view (Problems)
  const renderSessionDetailView = () => {
    if (!selectedSession) return null;
    return (
      <div className="session-detail-view">
        <div className="breadcrumb">
          <button
            onClick={() => {
              changeView('lab-detail');
              setSelectedSession(null);
            }}
          >
            Back to Sessions
          </button>
          <span>{selectedSession.title}</span>
        </div>

        <div className="detail-header">
          <h1>{selectedSession.title}</h1>
          <span className="session-topic">Topic: {selectedSession.topic}</span>
          <div className="detail-meta">
            <span>Start: {new Date(selectedSession.start_time).toLocaleString()}</span>
          </div>
        </div>

        <div className="problems-section">
          <div className="section-header">
            <h2>Problems</h2>
            <button className="btn-primary" onClick={() => openModal('problem')}>
              Add Problem
            </button>
          </div>

          {problems.length === 0 ? (
            <div className="empty-state">
              <div className="empty-icon"></div>
              <h3>No problems yet</h3>
              <p>Add problems to this session</p>
            </div>
          ) : (
            <div className="problems-list">
              {problems.map((problem) => (
                <div key={problem.id} className="problem-item">
                  <div className="problem-header">
                    <h4>{problem.title}</h4>
                    <span className={`badge badge-${problem.difficulty}`}>
                      {problem.difficulty}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {showModal && modalType === 'problem' && (
          <ProblemFormModal
            onClose={closeModal}
            onSubmit={handleCreate}
            sessionId={selectedSession.session_id}
            loading={loading}
          />
        )}
      </div>
    );
  };

  // Main render
  return (
    <>
      {currentView === 'courses' && renderCoursesView()}
      {currentView === 'labs' && renderLabsView()}
      {currentView === 'theory' && renderTheoryView()}
      {currentView === 'lab-detail' && selectedLab && renderLabDetailView()}
      {currentView === 'theory-detail' && selectedTheory && renderTheoryDetailView()}
      {currentView === 'session-detail' && selectedSession && renderSessionDetailView()}
    </>
  );
};

// ─── Modal Props Interfaces ──────────────────────────────────────────────────

interface CourseFormModalProps {
  onClose: () => void;
  onSubmit: (data: CourseFormData) => void;
  programs: Program[];
  branches: Branch[];
  loading: boolean;
}

interface LabFormModalProps {
  onClose: () => void;
  onSubmit: (data: LabFormData) => void;
  courses: Course[];
  loading: boolean;
}

interface TheoryFormModalProps {
  onClose: () => void;
  onSubmit: (data: TheoryFormData) => void;
  courses: Course[];
  loading: boolean;
}

interface SessionFormModalProps {
  onClose: () => void;
  onSubmit: (data: SessionFormData) => void;
  labId: number | undefined;
  courseId: number | undefined;
  loading: boolean;
}

interface WeekFormModalProps {
  onClose: () => void;
  onSubmit: (data: WeekFormData) => void;
  theoryId: number;
  loading: boolean;
}

interface ModuleFormModalProps {
  onClose: () => void;
  onSubmit: (data: ModuleFormData) => void;
  weekId: number;
  loading: boolean;
}

interface ProblemFormModalProps {
  onClose: () => void;
  onSubmit: (data: ProblemFormData) => void;
  sessionId: number;
  loading: boolean;
}

// ─── Course Form Modal ───────────────────────────────────────────────────────

const CourseFormModal = ({
  onClose,
  onSubmit,
  programs,
  branches,
  loading,
}: CourseFormModalProps) => {
  const [formData, setFormData] = useState<CourseFormData>({
    course_name: '',
    course_code: '',
    program_id: '',
    branch_ids: [],
    course_type: 'lab',
    credits: 3,
    semester_id: 1,
  });

  const filteredBranches = branches.filter(
    (b) => !formData.program_id || b.program_id === parseInt(formData.program_id)
  );

  const handleBranchToggle = (branchId: number) => {
    setFormData((prev) => {
      const newBranchIds = prev.branch_ids.includes(branchId)
        ? prev.branch_ids.filter((id) => id !== branchId)
        : [...prev.branch_ids, branchId];
      return { ...prev, branch_ids: newBranchIds };
    });
  };

  return (
    <div
      className="modal-overlay"
      onClick={(e: MouseEvent<HTMLDivElement>) => e.target === e.currentTarget && onClose()}
    >
      <div className="modal-content">
        <div className="modal-header">
          <h2>Create New Course</h2>
          <button className="modal-close" onClick={onClose}>
            Close
          </button>
        </div>
        <form
          onSubmit={(e: FormEvent) => {
            e.preventDefault();
            onSubmit(formData);
          }}
          className="admin-form"
        >
          <div className="form-group">
            <label>Course Name *</label>
            <input
              type="text"
              value={formData.course_name}
              onChange={(e: ChangeEvent<HTMLInputElement>) =>
                setFormData({ ...formData, course_name: e.target.value })
              }
              placeholder="e.g., Data Structures"
              required
            />
          </div>

          <div className="form-group">
            <label>Course Code *</label>
            <input
              type="text"
              value={formData.course_code}
              onChange={(e: ChangeEvent<HTMLInputElement>) =>
                setFormData({ ...formData, course_code: e.target.value })
              }
              placeholder="e.g., CSE201"
              required
            />
          </div>

          <div className="form-group">
            <label>Program *</label>
            <select
              value={formData.program_id}
              onChange={(e: ChangeEvent<HTMLSelectElement>) =>
                setFormData({ ...formData, program_id: e.target.value, branch_ids: [] })
              }
              required
            >
              <option value="">Select Program</option>
              {programs.map((program) => (
                <option key={program.program_id} value={program.program_id}>
                  {program.program_name}
                </option>
              ))}
            </select>
          </div>

          <div className="form-group">
            <label>Branches * (multi-select)</label>
            <div className="multi-select-container">
              {filteredBranches.map((branch) => (
                <label key={branch.branch_id} className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={formData.branch_ids.includes(branch.branch_id)}
                    onChange={() => handleBranchToggle(branch.branch_id)}
                    disabled={!formData.program_id}
                  />
                  <span>{branch.branch_name}</span>
                </label>
              ))}
            </div>
            {formData.branch_ids.length === 0 && formData.program_id && (
              <small className="error-text">Please select at least one branch</small>
            )}
          </div>

          <div className="form-row">
            <div className="form-group">
              <label>Course Type *</label>
              <select
                value={formData.course_type}
                onChange={(e: ChangeEvent<HTMLSelectElement>) =>
                  setFormData({ ...formData, course_type: e.target.value })
                }
                required
              >
                <option value="lab">Lab</option>
                <option value="theory">Theory</option>
              </select>
            </div>
            <div className="form-group">
              <label>Credits *</label>
              <input
                type="number"
                value={formData.credits}
                onChange={(e: ChangeEvent<HTMLInputElement>) =>
                  setFormData({ ...formData, credits: parseInt(e.target.value) })
                }
                min="1"
                max="6"
                required
              />
            </div>
          </div>

          <div className="form-group">
            <label>Semester *</label>
            <select
              value={formData.semester_id}
              onChange={(e: ChangeEvent<HTMLSelectElement>) =>
                setFormData({ ...formData, semester_id: parseInt(e.target.value) })
              }
              required
            >
              {[1, 2, 3, 4, 5, 6, 7, 8].map((sem) => (
                <option key={sem} value={sem}>
                  Semester {sem}
                </option>
              ))}
            </select>
          </div>

          <div className="form-actions">
            <button type="submit" className="btn-admin btn-admin-primary" disabled={loading}>
              {loading ? 'Creating...' : 'Create Course'}
            </button>
            <button type="button" className="btn-admin btn-admin-secondary" onClick={onClose}>
              Cancel
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

// ─── Lab Form Modal ──────────────────────────────────────────────────────────

const LabFormModal = ({ onClose, onSubmit, courses, loading }: LabFormModalProps) => {
  const [formData, setFormData] = useState<LabFormData>({
    lab_name: '',
    lab_code: '',
    description: '',
    course_id: '',
  });

  return (
    <div
      className="modal-overlay"
      onClick={(e: MouseEvent<HTMLDivElement>) => e.target === e.currentTarget && onClose()}
    >
      <div className="modal-content">
        <div className="modal-header">
          <h2>Create New Lab</h2>
          <button className="modal-close" onClick={onClose}>
            Close
          </button>
        </div>
        <form
          onSubmit={(e: FormEvent) => {
            e.preventDefault();
            onSubmit(formData);
          }}
          className="admin-form"
        >
          <div className="form-group">
            <label>Lab Name *</label>
            <input
              type="text"
              value={formData.lab_name}
              onChange={(e: ChangeEvent<HTMLInputElement>) =>
                setFormData({ ...formData, lab_name: e.target.value })
              }
              placeholder="e.g., Data Structures Lab"
              required
            />
          </div>

          <div className="form-group">
            <label>Lab Code *</label>
            <input
              type="text"
              value={formData.lab_code}
              onChange={(e: ChangeEvent<HTMLInputElement>) =>
                setFormData({ ...formData, lab_code: e.target.value })
              }
              placeholder="e.g., DSL201"
              required
            />
          </div>

          <div className="form-group">
            <label>Select Course *</label>
            <select
              value={formData.course_id}
              onChange={(e: ChangeEvent<HTMLSelectElement>) =>
                setFormData({ ...formData, course_id: e.target.value })
              }
              required
            >
              <option value="">Select Course</option>
              {courses.map((course) => (
                <option key={course.id} value={course.id}>
                  {course.course_code} - {course.course_name}
                </option>
              ))}
            </select>
          </div>

          <div className="form-group">
            <label>Description</label>
            <textarea
              value={formData.description}
              onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                setFormData({ ...formData, description: e.target.value })
              }
              placeholder="Lab description..."
              rows={3}
            />
          </div>

          <div className="form-actions">
            <button type="submit" className="btn-admin btn-admin-primary" disabled={loading}>
              {loading ? 'Creating...' : 'Create Lab'}
            </button>
            <button type="button" className="btn-admin btn-admin-secondary" onClick={onClose}>
              Cancel
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

// ─── Theory Form Modal ───────────────────────────────────────────────────────

const TheoryFormModal = ({ onClose, onSubmit, courses, loading }: TheoryFormModalProps) => {
  const [formData, setFormData] = useState<TheoryFormData>({
    theory_name: '',
    theory_code: '',
    course_id: '',
  });

  return (
    <div
      className="modal-overlay"
      onClick={(e: MouseEvent<HTMLDivElement>) => e.target === e.currentTarget && onClose()}
    >
      <div className="modal-content">
        <div className="modal-header">
          <h2>Create New Theory</h2>
          <button className="modal-close" onClick={onClose}>
            ×
          </button>
        </div>
        <form
          onSubmit={(e: FormEvent) => {
            e.preventDefault();
            onSubmit(formData);
          }}
          className="admin-form"
        >
          <div className="form-group">
            <label>Theory Name *</label>
            <input
              type="text"
              value={formData.theory_name}
              onChange={(e: ChangeEvent<HTMLInputElement>) =>
                setFormData({ ...formData, theory_name: e.target.value })
              }
              placeholder="e.g., Data Structures Theory"
              required
            />
          </div>

          <div className="form-group">
            <label>Theory Code *</label>
            <input
              type="text"
              value={formData.theory_code}
              onChange={(e: ChangeEvent<HTMLInputElement>) =>
                setFormData({ ...formData, theory_code: e.target.value })
              }
              placeholder="e.g., DST201"
              required
            />
          </div>

          <div className="form-group">
            <label>Select Course *</label>
            <select
              value={formData.course_id}
              onChange={(e: ChangeEvent<HTMLSelectElement>) =>
                setFormData({ ...formData, course_id: e.target.value })
              }
              required
            >
              <option value="">Select Course</option>
              {courses.map((course) => (
                <option key={course.id} value={course.id}>
                  {course.course_code} - {course.course_name}
                </option>
              ))}
            </select>
          </div>

          <div className="form-actions">
            <button type="submit" className="btn-admin btn-admin-primary" disabled={loading}>
              {loading ? 'Creating...' : 'Create Theory'}
            </button>
            <button type="button" className="btn-admin btn-admin-secondary" onClick={onClose}>
              Cancel
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

// ─── Session Form Modal ──────────────────────────────────────────────────────

const SessionFormModal = ({
  onClose,
  onSubmit,
  labId,
  courseId,
  loading,
}: SessionFormModalProps) => {
  const [formData, setFormData] = useState<SessionFormData>(() => {
    const now = new Date();
    const twoHoursLater = new Date(now.getTime() + 2 * 60 * 60 * 1000);
    return {
      title: '',
      topic: '',
      lab_id: labId,
      course_id: courseId,
      start_time: now.toISOString().slice(0, 16),
      end_time: twoHoursLater.toISOString().slice(0, 16),
      max_attempts: 0,
    };
  });

  return (
    <div
      className="modal-overlay"
      onClick={(e: MouseEvent<HTMLDivElement>) => e.target === e.currentTarget && onClose()}
    >
      <div className="modal-content">
        <div className="modal-header">
          <h2>Create New Session</h2>
          <button className="modal-close" onClick={onClose}>
            ×
          </button>
        </div>
        <form
          onSubmit={(e: FormEvent) => {
            e.preventDefault();
            onSubmit({
              ...formData,
              start_time: new Date(formData.start_time).toISOString(),
              end_time: new Date(formData.end_time).toISOString(),
            });
          }}
          className="admin-form"
        >
          <div className="form-group">
            <label>Session Title *</label>
            <input
              type="text"
              value={formData.title}
              onChange={(e: ChangeEvent<HTMLInputElement>) =>
                setFormData({ ...formData, title: e.target.value })
              }
              placeholder="e.g., Session 1: Basic Arrays"
              required
            />
          </div>

          <div className="form-group">
            <label>Topic * (MANDATORY)</label>
            <input
              type="text"
              value={formData.topic}
              onChange={(e: ChangeEvent<HTMLInputElement>) =>
                setFormData({ ...formData, topic: e.target.value })
              }
              placeholder="e.g., Arrays"
              required
            />
          </div>

          <div className="form-row">
            <div className="form-group">
              <label>Start Time *</label>
              <input
                type="datetime-local"
                value={formData.start_time}
                onChange={(e: ChangeEvent<HTMLInputElement>) =>
                  setFormData({ ...formData, start_time: e.target.value })
                }
                required
              />
            </div>
            <div className="form-group">
              <label>End Time *</label>
              <input
                type="datetime-local"
                value={formData.end_time}
                onChange={(e: ChangeEvent<HTMLInputElement>) =>
                  setFormData({ ...formData, end_time: e.target.value })
                }
                required
              />
            </div>
          </div>

          <div className="form-group">
            <label>Max Attempts (0 = unlimited)</label>
            <input
              type="number"
              value={formData.max_attempts}
              onChange={(e: ChangeEvent<HTMLInputElement>) =>
                setFormData({ ...formData, max_attempts: parseInt(e.target.value) })
              }
              min="0"
            />
          </div>

          <div className="form-actions">
            <button type="submit" className="btn-admin btn-admin-primary" disabled={loading}>
              {loading ? 'Creating...' : 'Create Session'}
            </button>
            <button type="button" className="btn-admin btn-admin-secondary" onClick={onClose}>
              Cancel
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

// ─── Week Form Modal ─────────────────────────────────────────────────────────

const WeekFormModal = ({ onClose, onSubmit, theoryId: _theoryId, loading }: WeekFormModalProps) => {
  const [formData, setFormData] = useState<WeekFormData>({
    week_number: 1,
    title: '',
    description: '',
  });

  return (
    <div
      className="modal-overlay"
      onClick={(e: MouseEvent<HTMLDivElement>) => e.target === e.currentTarget && onClose()}
    >
      <div className="modal-content">
        <div className="modal-header">
          <h2>Create New Week</h2>
          <button className="modal-close" onClick={onClose}>
            ×
          </button>
        </div>
        <form
          onSubmit={(e: FormEvent) => {
            e.preventDefault();
            onSubmit(formData);
          }}
          className="admin-form"
        >
          <div className="form-group">
            <label>Week Number *</label>
            <input
              type="number"
              value={formData.week_number}
              onChange={(e: ChangeEvent<HTMLInputElement>) =>
                setFormData({ ...formData, week_number: parseInt(e.target.value) })
              }
              min="1"
              required
            />
          </div>

          <div className="form-group">
            <label>Title *</label>
            <input
              type="text"
              value={formData.title}
              onChange={(e: ChangeEvent<HTMLInputElement>) =>
                setFormData({ ...formData, title: e.target.value })
              }
              placeholder="e.g., Introduction to Arrays"
              required
            />
          </div>

          <div className="form-group">
            <label>Description</label>
            <textarea
              value={formData.description}
              onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                setFormData({ ...formData, description: e.target.value })
              }
              placeholder="Week description..."
              rows={3}
            />
          </div>

          <div className="form-actions">
            <button type="submit" className="btn-admin btn-admin-primary" disabled={loading}>
              {loading ? 'Creating...' : 'Create Week'}
            </button>
            <button type="button" className="btn-admin btn-admin-secondary" onClick={onClose}>
              Cancel
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

// ─── Module Form Modal ───────────────────────────────────────────────────────

const ModuleFormModal = ({ onClose, onSubmit, weekId: _weekId, loading }: ModuleFormModalProps) => {
  const [formData, setFormData] = useState<ModuleFormData>({
    title: '',
    description: '',
    content: '',
    order_index: 0,
  });

  return (
    <div
      className="modal-overlay"
      onClick={(e: MouseEvent<HTMLDivElement>) => e.target === e.currentTarget && onClose()}
    >
      <div className="modal-content">
        <div className="modal-header">
          <h2>Create New Module</h2>
          <button className="modal-close" onClick={onClose}>
            ×
          </button>
        </div>
        <form
          onSubmit={(e: FormEvent) => {
            e.preventDefault();
            onSubmit(formData);
          }}
          className="admin-form"
        >
          <div className="form-group">
            <label>Module Title *</label>
            <input
              type="text"
              value={formData.title}
              onChange={(e: ChangeEvent<HTMLInputElement>) =>
                setFormData({ ...formData, title: e.target.value })
              }
              placeholder="e.g., Array Operations"
              required
            />
          </div>

          <div className="form-group">
            <label>Description</label>
            <textarea
              value={formData.description}
              onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                setFormData({ ...formData, description: e.target.value })
              }
              placeholder="Module description..."
              rows={2}
            />
          </div>

          <div className="form-group">
            <label>Content</label>
            <textarea
              value={formData.content}
              onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                setFormData({ ...formData, content: e.target.value })
              }
              placeholder="Module content..."
              rows={5}
            />
          </div>

          <div className="form-actions">
            <button type="submit" className="btn-admin btn-admin-primary" disabled={loading}>
              {loading ? 'Creating...' : 'Create Module'}
            </button>
            <button type="button" className="btn-admin btn-admin-secondary" onClick={onClose}>
              Cancel
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

// ─── Problem Form Modal ──────────────────────────────────────────────────────

const ProblemFormModal = ({
  onClose,
  onSubmit,
  sessionId: _sessionId,
  loading,
}: ProblemFormModalProps) => {
  const [formData, setFormData] = useState<ProblemFormData>({
    title: '',
    description: '',
    difficulty: 'easy',
    time_limit: 2000,
    memory_limit: 256000,
    test_cases: [{ input: '', expected_output: '', is_sample: true, points: 10 }],
  });

  const addTestCase = () => {
    setFormData({
      ...formData,
      test_cases: [
        ...formData.test_cases,
        { input: '', expected_output: '', is_sample: false, points: 10 },
      ],
    });
  };

  const removeTestCase = (index: number) => {
    if (formData.test_cases.length > 1) {
      const newTestCases = formData.test_cases.filter((_, i) => i !== index);
      setFormData({ ...formData, test_cases: newTestCases });
    }
  };

  const updateTestCase = (
    index: number,
    field: keyof TestCase,
    value: string | boolean | number
  ) => {
    const newTestCases = [...formData.test_cases];
    newTestCases[index] = { ...newTestCases[index], [field]: value };
    setFormData({ ...formData, test_cases: newTestCases });
  };

  return (
    <div
      className="modal-overlay"
      onClick={(e: MouseEvent<HTMLDivElement>) => e.target === e.currentTarget && onClose()}
    >
      <div className="modal-content modal-large">
        <div className="modal-header">
          <h2>Create New Problem</h2>
          <button className="modal-close" onClick={onClose}>
            ×
          </button>
        </div>
        <form
          onSubmit={(e: FormEvent) => {
            e.preventDefault();
            onSubmit(formData);
          }}
          className="admin-form"
        >
          <div className="form-group">
            <label>Problem Title *</label>
            <input
              type="text"
              value={formData.title}
              onChange={(e: ChangeEvent<HTMLInputElement>) =>
                setFormData({ ...formData, title: e.target.value })
              }
              placeholder="e.g., Two Sum"
              required
            />
          </div>

          <div className="form-row">
            <div className="form-group">
              <label>Difficulty *</label>
              <select
                value={formData.difficulty}
                onChange={(e: ChangeEvent<HTMLSelectElement>) =>
                  setFormData({ ...formData, difficulty: e.target.value })
                }
                required
              >
                <option value="easy">Easy</option>
                <option value="medium">Medium</option>
                <option value="hard">Hard</option>
              </select>
            </div>
            <div className="form-group">
              <label>Time Limit (ms) *</label>
              <input
                type="number"
                value={formData.time_limit}
                onChange={(e: ChangeEvent<HTMLInputElement>) =>
                  setFormData({ ...formData, time_limit: parseInt(e.target.value) })
                }
                min="100"
                required
              />
            </div>
          </div>

          <div className="form-group">
            <label>Description *</label>
            <textarea
              value={formData.description}
              onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                setFormData({ ...formData, description: e.target.value })
              }
              placeholder="Problem description..."
              rows={5}
              required
            />
          </div>

          <div className="test-cases-section">
            <div className="test-cases-header">
              <h3>Test Cases</h3>
              <button type="button" className="btn-admin btn-admin-secondary" onClick={addTestCase}>
                Add Test Case
              </button>
            </div>

            {formData.test_cases.map((tc, index) => (
              <div key={index} className="test-case-item">
                <div className="test-case-header">
                  <span className={`test-case-type ${tc.is_sample ? 'sample' : 'hidden'}`}>
                    {tc.is_sample ? '📘 Sample Test Case' : '🔒 Hidden Test Case'}
                  </span>
                  {formData.test_cases.length > 1 && (
                    <button
                      type="button"
                      className="btn-action btn-delete"
                      onClick={() => removeTestCase(index)}
                    >
                      Remove
                    </button>
                  )}
                </div>

                <div className="form-row">
                  <div className="form-group" style={{ flex: 1 }}>
                    <label>
                      <input
                        type="checkbox"
                        checked={tc.is_sample}
                        onChange={(e: ChangeEvent<HTMLInputElement>) =>
                          updateTestCase(index, 'is_sample', e.target.checked)
                        }
                        style={{ marginRight: '8px' }}
                      />
                      Sample (visible to students)
                    </label>
                  </div>
                  <div className="form-group" style={{ width: '120px' }}>
                    <label>Points</label>
                    <input
                      type="number"
                      value={tc.points}
                      onChange={(e: ChangeEvent<HTMLInputElement>) =>
                        updateTestCase(index, 'points', parseInt(e.target.value) || 0)
                      }
                      min="0"
                    />
                  </div>
                </div>

                <div className="form-group">
                  <label>Input *</label>
                  <textarea
                    value={tc.input}
                    onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                      updateTestCase(index, 'input', e.target.value)
                    }
                    placeholder="Enter input for this test case..."
                    rows={3}
                    required
                  />
                </div>

                <div className="form-group">
                  <label>Expected Output *</label>
                  <textarea
                    value={tc.expected_output}
                    onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                      updateTestCase(index, 'expected_output', e.target.value)
                    }
                    placeholder="Enter expected output..."
                    rows={2}
                    required
                  />
                </div>
              </div>
            ))}
          </div>

          <div className="form-actions">
            <button type="submit" className="btn-admin btn-admin-primary" disabled={loading}>
              {loading ? 'Creating...' : 'Create Problem'}
            </button>
            <button type="button" className="btn-admin btn-admin-secondary" onClick={onClose}>
              Cancel
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default CourseCreationDashboard;
