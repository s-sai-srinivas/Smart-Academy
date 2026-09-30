import { useState, useEffect, FormEvent, ChangeEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '../../services/api';
import { showSuccess, showError } from '../../utils/showAlert';
import { getErrorMessage } from '../../utils/error';
import './AdminDashboard.css';

interface Course {
  id: string;
  course_code: string;
  course_name: string;
}

interface Section {
  section_id: string;
  section_name: string;
}

interface SessionItem {
  session_id: string;
  title: string;
  start_time: string;
  end_time: string;
  max_attempts: number;
  course?: Course;
  section?: Section;
}

interface FormData {
  course_id: string;
  section_id: string;
  title: string;
  start_time: string;
  end_time: string;
  max_attempts: number;
}

const AdminLabSessions = () => {
  const navigate = useNavigate();
  const [showModal, setShowModal] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(false);
  const [sessions, setSessions] = useState<SessionItem[]>([]);
  const [courses, setCourses] = useState<Course[]>([]);
  const [sections, setSections] = useState<Section[]>([]);
  const [formData, setFormData] = useState<FormData>({
    course_id: '',
    section_id: '',
    title: '',
    start_time: '',
    end_time: '',
    max_attempts: 0,
  });

  const fetchSessions = async () => {
    try {
      const response = (await api.get('/admin/lab-sessions')) as SessionItem[];
      setSessions(response);
    } catch (error) {
      console.error('Error fetching lab sessions:', error);
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

  const fetchSections = async () => {
    try {
      const response = (await api.get('/sections')) as Section[];
      setSections(response);
    } catch (error) {
      console.error('Error fetching sections:', error);
    }
  };

  useEffect(() => {
    fetchSessions();
    fetchCourses();
    fetchSections();
  }, []);

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setLoading(true);

    try {
      const payload = {
        ...formData,
        section_id: formData.section_id === '' ? null : formData.section_id,
      };
      await api.post('/admin/lab-sessions', payload);
      setShowModal(false);
      setFormData({
        course_id: '',
        section_id: '',
        title: '',
        start_time: '',
        end_time: '',
        max_attempts: 0,
      });
      fetchSessions();
      showSuccess('Lab session created successfully!');
    } catch (error: unknown) {
      const message = getErrorMessage(error, 'Failed to create lab session');
      console.error('Error creating lab session:', error);
      showError(`Failed to create lab session: ${message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleManageProblems = (sessionId: string) => {
    navigate(`/admin/lab-sessions/${sessionId}/problems`);
  };

  return (
    <div className="admin-labs">
      <div className="admin-header">
        <h1>Lab Sessions</h1>
        <button className="btn-admin btn-admin-primary" onClick={() => setShowModal(true)}>
          Schedule Lab Session
        </button>
      </div>

      {sessions.length === 0 ? (
        <div className="admin-empty">
          <div className="admin-empty-icon"></div>
          <h3>No lab sessions found</h3>
          <p>Schedule your first lab session to get started</p>
        </div>
      ) : (
        <div className="admin-card">
          <table className="admin-table">
            <thead>
              <tr>
                <th>Title</th>
                <th>Course</th>
                <th>Section</th>
                <th>Start Time</th>
                <th>End Time</th>
                <th>Max Attempts</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {sessions.map((session) => (
                <tr key={session.session_id}>
                  <td>{session.title}</td>
                  <td>{session.course?.course_name || 'N/A'}</td>
                  <td>{session.section?.section_name || 'All Sections'}</td>
                  <td>{new Date(session.start_time).toLocaleString()}</td>
                  <td>{new Date(session.end_time).toLocaleString()}</td>
                  <td>{session.max_attempts === 0 ? 'Unlimited' : session.max_attempts}</td>
                  <td>
                    <button
                      className="btn-admin btn-admin-secondary"
                      onClick={() => handleManageProblems(session.session_id)}
                      style={{ padding: '0.25rem 0.75rem', fontSize: '0.875rem' }}
                    >
                      Manage Problems
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {showModal && (
        <div
          className="modal-overlay"
          onClick={(e) => e.target === e.currentTarget && setShowModal(false)}
        >
          <div className="modal-content">
            <div className="modal-header">
              <h2>Schedule New Lab Session</h2>
              <button className="modal-close" onClick={() => setShowModal(false)}>
                Close
              </button>
            </div>
            <form onSubmit={handleSubmit} className="admin-form">
              <div className="form-group">
                <label>Title *</label>
                <input
                  type="text"
                  value={formData.title}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    setFormData({ ...formData, title: e.target.value })
                  }
                  placeholder="e.g., Arrays Lab - Week 1"
                  required
                />
              </div>

              <div className="form-row">
                <div className="form-group">
                  <label>Course *</label>
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
                  <label>Section (optional)</label>
                  <select
                    value={formData.section_id}
                    onChange={(e: ChangeEvent<HTMLSelectElement>) =>
                      setFormData({ ...formData, section_id: e.target.value })
                    }
                  >
                    <option value="">All Sections</option>
                    {sections.map((section) => (
                      <option key={section.section_id} value={section.section_id}>
                        {section.section_name}
                      </option>
                    ))}
                  </select>
                </div>
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
                    setFormData({ ...formData, max_attempts: parseInt(e.target.value) || 0 })
                  }
                  min="0"
                />
              </div>

              <div className="form-actions">
                <button type="submit" className="btn-admin btn-admin-primary" disabled={loading}>
                  {loading ? 'Creating...' : 'Schedule Lab Session'}
                </button>
                <button
                  type="button"
                  className="btn-admin btn-admin-secondary"
                  onClick={() => setShowModal(false)}
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default AdminLabSessions;
