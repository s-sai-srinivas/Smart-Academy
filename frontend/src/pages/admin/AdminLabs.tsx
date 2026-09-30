import { useState, useEffect, useCallback, FormEvent, ChangeEvent } from 'react';
import api from '../../services/api';
import { referenceAPI } from '../../services/api';
import { showError } from '../../utils/showAlert';
import { getErrorMessage, getErrorStatus } from '../../utils/error';
import CreateProblemModal from '../../components/admin/CreateProblemModal';

interface Course {
  id: string;
  course_type: string;
  course_code: string;
  course_name: string;
}

interface Lab {
  id: string;
  lab_name: string;
  lab_code: string;
}

interface Problem {
  id: string;
  title: string;
  difficulty: string;
}

interface Session {
  id: string;
  session_order: number;
  session_name: string;
  topic_name?: string;
  problems?: Problem[];
}

interface Topic {
  id: number;
  name: string;
  description?: string;
}

interface SessionForm {
  session_name: string;
  session_order: number;
  topic_name: string;
}

const AdminLabs = () => {
  const [labCourses, setLabCourses] = useState<Course[]>([]);
  const [selectedCourse, setSelectedCourse] = useState<string>('');
  const [lab, setLab] = useState<Lab | null>(null);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [topics, setTopics] = useState<Topic[]>([]);
  const [loading, setLoading] = useState<boolean>(true);

  // Modals
  const [showSessionModal, setShowSessionModal] = useState<boolean>(false);
  const [showProblemModal, setShowProblemModal] = useState<boolean>(false);
  const [activeSessionId, setActiveSessionId] = useState<string | null>(null);

  // Forms
  const [sessionForm, setSessionForm] = useState<SessionForm>({
    session_name: '',
    session_order: 1,
    topic_name: '',
  });

  const fetchLabCoursesData = async (): Promise<Course[]> => {
    try {
      const response = (await api.get('/admin/courses')) as Course[];
      return response.filter((c) => c.course_type === 'lab' || c.course_type === 'integrated');
    } catch (error) {
      console.error('Error fetching courses:', error);
      return [];
    }
  };

  const fetchTopicsData = async (): Promise<Topic[]> => {
    try {
      const response = (await referenceAPI.getTopics()) as Topic[];
      return response || [];
    } catch (error) {
      console.error('Error fetching topics:', error);
      return [];
    }
  };

  const fetchSessionsData = useCallback(async (labId: string): Promise<Session[]> => {
    try {
      const response = (await api.get(`/admin/labs/${labId}/sessions`)) as Session[];
      return response || [];
    } catch (error) {
      console.error('Error fetching sessions:', error);
      return [];
    }
  }, []);

  const fetchLabData = useCallback(
    async (courseId: string) => {
      try {
        const response = (await api.get(`/admin/courses/${courseId}/lab`)) as Lab;
        if (response && response.id) {
          const sessions = await fetchSessionsData(response.id);
          return { lab: response, sessions };
        }
        return { lab: response, sessions: [] as Session[] };
      } catch (error: unknown) {
        if (getErrorStatus(error) === 404) {
          return { lab: null, sessions: [] as Session[] };
        }
        return { lab: null, sessions: [] as Session[] };
      }
    },
    [fetchSessionsData]
  );

  useEffect(() => {
    Promise.all([fetchLabCoursesData(), fetchTopicsData()]).then(([courses, topics]) => {
      setLabCourses(courses);
      setTopics(topics);
      setLoading(false);
    });
  }, []);

  useEffect(() => {
    if (selectedCourse) {
      fetchLabData(selectedCourse).then(({ lab, sessions }) => {
        setLab(lab);
        setSessions(sessions);
      });
    }
  }, [selectedCourse, fetchLabData]);

  const handleCreateSession = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    try {
      await api.post('/admin/labs/sessions', {
        lab_id: lab?.id,
        session_name: sessionForm.session_name,
        session_order: parseInt(String(sessionForm.session_order)),
        topic_name: sessionForm.topic_name,
      });
      setShowSessionModal(false);
      setSessionForm({ session_name: '', session_order: sessions.length + 1, topic_name: '' });
      if (lab?.id) fetchSessionsData(lab.id).then(setSessions);
    } catch (error: unknown) {
      showError(getErrorMessage(error, 'Failed to create session'));
    }
  };

  const handleCreateProblemSuccess = () => {
    setShowProblemModal(false);
    if (lab?.id) fetchSessionsData(lab.id).then(setSessions);
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
          <span className="text-text-secondary">Loading...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-text-primary mb-1">Lab Management</h1>
        <p className="text-text-secondary text-sm">
          Manage lab sessions and problems for lab courses
        </p>
      </div>

      {/* Lab Course Selector */}
      <div className="card">
        <div className="form-group">
          <label className="form-label">Select Lab Course</label>
          <select
            className="select"
            value={selectedCourse}
            onChange={(e: ChangeEvent<HTMLSelectElement>) => setSelectedCourse(e.target.value)}
          >
            <option value="">-- Select a Lab Course --</option>
            {labCourses.map((course) => (
              <option key={course.id} value={course.id}>
                {course.course_code} - {course.course_name}
              </option>
            ))}
          </select>
          {labCourses.length === 0 && (
            <p className="text-text-tertiary text-sm mt-2">
              No lab courses found. Create a lab course in the Courses page first.
            </p>
          )}
        </div>
      </div>

      {/* Lab Content */}
      {selectedCourse && lab && (
        <div className="card">
          <div className="flex items-center justify-between mb-6">
            <div>
              <h2 className="text-xl font-semibold text-text-primary">{lab.lab_name}</h2>
              <p className="text-text-secondary text-sm mt-1">Code: {lab.lab_code}</p>
            </div>
            <button
              className="btn btn-primary"
              onClick={() => {
                setSessionForm({
                  session_name: '',
                  session_order: sessions.length + 1,
                  topic_name: '',
                });
                setShowSessionModal(true);
              }}
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 4v16m8-8H4"
                />
              </svg>
              Add Session
            </button>
          </div>

          {sessions.length > 0 ? (
            <div className="space-y-4">
              {sessions.map((session) => (
                <div
                  key={session.id}
                  className="bg-background-tertiary rounded-lg border border-background-border p-4"
                >
                  <div className="flex items-center justify-between mb-4">
                    <div>
                      <span className="font-semibold text-text-primary">
                        Session {session.session_order}: {session.session_name}
                      </span>
                      {session.topic_name && (
                        <span className="text-text-secondary ml-2">({session.topic_name})</span>
                      )}
                    </div>
                    <button
                      className="btn btn-secondary text-xs px-3 py-1.5"
                      onClick={() => {
                        setActiveSessionId(session.id);
                        setShowProblemModal(true);
                      }}
                    >
                      <svg
                        className="w-3 h-3"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M12 4v16m8-8H4"
                        />
                      </svg>
                      Add Problem
                    </button>
                  </div>
                  {session.problems && session.problems.length > 0 ? (
                    <div className="pl-4 space-y-2">
                      {session.problems.map((problem) => (
                        <div
                          key={problem.id}
                          className="flex items-center justify-between py-2 border-b border-background-border last:border-b-0"
                        >
                          <span className="text-text-primary">{problem.title}</span>
                          <span
                            className={`badge ${
                              problem.difficulty === 'easy'
                                ? 'badge-success'
                                : problem.difficulty === 'medium'
                                  ? 'badge-warning'
                                  : 'badge-danger'
                            }`}
                          >
                            {problem.difficulty}
                          </span>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="text-text-muted italic text-sm">No problems added yet.</p>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-12">
              <h3 className="text-lg font-semibold text-text-primary mb-2">No Sessions Created</h3>
              <p className="text-text-secondary text-sm mb-4">
                Create your first session for this lab
              </p>
              <button className="btn btn-primary" onClick={() => setShowSessionModal(true)}>
                Create Session
              </button>
            </div>
          )}
        </div>
      )}

      {/* Session Modal */}
      {showSessionModal && (
        <div className="modal-overlay" onClick={() => setShowSessionModal(false)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-2xl font-semibold text-text-primary">Add Session</h2>
              <button
                onClick={() => setShowSessionModal(false)}
                className="text-text-muted hover:text-text-primary transition-colors p-1"
              >
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M6 18L18 6M6 6l12 12"
                  />
                </svg>
              </button>
            </div>
            <form onSubmit={handleCreateSession} className="space-y-5">
              <div className="form-group">
                <label className="form-label">Session Name</label>
                <input
                  type="text"
                  className="input"
                  value={sessionForm.session_name}
                  onChange={(e) => setSessionForm({ ...sessionForm, session_name: e.target.value })}
                  placeholder="e.g. Arrays and Strings"
                  required
                />
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label className="form-label">Order</label>
                  <input
                    type="number"
                    className="input"
                    value={sessionForm.session_order}
                    onChange={(e) =>
                      setSessionForm({ ...sessionForm, session_order: Number(e.target.value) })
                    }
                    min="1"
                    required
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Topic (Optional)</label>
                  <select
                    className="select"
                    value={sessionForm.topic_name}
                    onChange={(e) => setSessionForm({ ...sessionForm, topic_name: e.target.value })}
                  >
                    <option value="">Select a Topic</option>
                    {topics.map((topic) => (
                      <option key={topic.id} value={topic.name}>
                        {topic.name}
                      </option>
                    ))}
                  </select>
                  {topics.length === 0 && (
                    <p className="text-text-tertiary text-sm mt-2">
                      No topics available. Contact super admin to create topics.
                    </p>
                  )}
                </div>
              </div>
              <div className="flex gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowSessionModal(false)}
                  className="btn btn-secondary flex-1"
                >
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary flex-1">
                  Add Session
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Problem Modal */}
      {showProblemModal && (
        <CreateProblemModal
          labSessionId={activeSessionId ? Number(activeSessionId) : undefined}
          onClose={() => setShowProblemModal(false)}
          onSuccess={handleCreateProblemSuccess}
        />
      )}
    </div>
  );
};

export default AdminLabs;
