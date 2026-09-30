import { useState, useEffect, type FormEvent, type ChangeEvent, type MouseEvent } from 'react';
import api from '../../services/api';
import { showError, showSuccess } from '../../utils/showAlert';

interface FacultyUser {
  name?: string;
  email?: string;
  role?: string;
}

interface FacultyMember {
  regdno: string;
  user?: FacultyUser;
}

interface Assignment {
  faculty_regdno: string;
  course_code: string;
  course_name: string;
  cohort_year: number | string;
  section_name: string;
}

interface Course {
  id: number | string;
  course_code: string;
  course_name: string;
  course_type?: string;
}

interface Section {
  section_id: number;
  section_name: string;
  branch_name?: string;
  cohort_year?: number | string;
}

interface Toast {
  type: 'success' | 'error';
  message: string;
}

const HODFaculty = () => {
  const [loading, setLoading] = useState<boolean>(true);
  const [faculty, setFaculty] = useState<FacultyMember[]>([]);
  const [assignments, setAssignments] = useState<Assignment[]>([]);
  const [showAssignModal, setShowAssignModal] = useState<boolean>(false);
  const [showRoleModal, setShowRoleModal] = useState<boolean>(false);
  const [toast, setToast] = useState<Toast | null>(null);

  const showToast = (type: 'success' | 'error', message: string) => {
    setToast({ type, message });
    setTimeout(() => setToast(null), 5000);
  };

  // Form states
  const [assignForm, setAssignForm] = useState<{
    faculty_id: string;
    course_id: string;
    section_ids: number[];
  }>({
    faculty_id: '',
    course_id: '',
    section_ids: [],
  });
  const [sections, setSections] = useState<Section[]>([]);

  const [roleForm, setRoleForm] = useState<{
    regdno: string;
    new_role: string;
  }>({
    regdno: '',
    new_role: '',
  });

  const [courses, setCourses] = useState<Course[]>([]);

  useEffect(() => {
    fetchFaculty();
    fetchAssignments();
    fetchCourses();
  }, []);

  const fetchSections = async (courseId: string) => {
    if (!courseId) {
      setSections([]);
      setAssignForm((prev) => ({ ...prev, section_ids: [] }));
      return;
    }
    try {
      const response = (await api.get(`/hod/courses/${courseId}/sections`)) as Section[];
      setSections(Array.isArray(response) ? response : []);
      setAssignForm((prev) => ({ ...prev, section_ids: [] }));
    } catch (error) {
      console.error('Error fetching sections:', error);
      setSections([]);
      setAssignForm((prev) => ({ ...prev, section_ids: [] }));
    }
  };

  const fetchFaculty = async () => {
    try {
      const response = (await api.get('/hod/faculty')) as FacultyMember[];
      setFaculty(Array.isArray(response) ? response : []);
    } catch (error) {
      console.error('Error fetching faculty:', error);
      setFaculty([]);
      showError('Failed to load faculty data');
    } finally {
      setLoading(false);
    }
  };

  const fetchAssignments = async () => {
    try {
      const response = (await api.get('/hod/faculty/assignments')) as Assignment[];
      setAssignments(Array.isArray(response) ? response : []);
    } catch (error) {
      console.error('Error fetching assignments:', error);
      setAssignments([]);
    }
  };

  const fetchCourses = async () => {
    try {
      const response = (await api.get('/hod/courses')) as Course[];
      setCourses(Array.isArray(response) ? response : []);
    } catch (error) {
      console.error('Error fetching courses:', error);
      setCourses([]);
    }
  };

  const handleAssignSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (assignForm.section_ids.length === 0) {
      showError('Please select at least one section');
      showToast('error', 'Please select at least one section');
      return;
    }
    try {
      const payload = {
        faculty_regdno: assignForm.faculty_id,
        course_id: parseInt(assignForm.course_id),
        section_ids: assignForm.section_ids,
      };
      const res = (await api.post('/hod/faculty/assign', payload)) as { message?: string };
      setShowAssignModal(false);
      setAssignForm({ faculty_id: '', course_id: '', section_ids: [] });
      setSections([]);
      showSuccess(res?.message || 'Faculty assigned successfully!');
      fetchAssignments();
    } catch (error) {
      const err = error as { response?: { data?: { error?: string } }; message?: string };
      showError(err.response?.data?.error || err.message || 'Failed to assign faculty');
      showToast('error', err.response?.data?.error || err.message || 'Failed to assign faculty');
    }
  };

  const handleRoleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    try {
      const payload = {
        regdno: roleForm.regdno,
        new_role: roleForm.new_role,
      };
      await api.put('/hod/faculty/role', payload);
      setShowRoleModal(false);
      setRoleForm({ regdno: '', new_role: '' });
      showToast('success', 'Role updated successfully!');
      fetchFaculty();
    } catch (error) {
      const err = error as { message?: string };
      showToast('error', err.message || 'Failed to update role');
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
          <span className="text-text-secondary">Loading...</span>
        </div>
      </div>
    );
  }

  // Group assignments by faculty
  const assignmentsByFaculty = (assignments || []).reduce<Record<string, Assignment[]>>(
    (acc, assignment) => {
      if (!acc[assignment.faculty_regdno]) {
        acc[assignment.faculty_regdno] = [];
      }
      acc[assignment.faculty_regdno].push(assignment);
      return acc;
    },
    {}
  );

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-text-primary mb-1">Faculty Management</h1>
          <p className="text-text-secondary text-sm">Manage faculty roles and course assignments</p>
        </div>
        <button onClick={() => setShowAssignModal(true)} className="btn btn-primary">
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Assign Faculty
        </button>
      </div>

      {/* Faculty List */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4">Faculty Members</h2>
        {faculty.length === 0 ? (
          <p className="text-text-muted text-center py-8">No faculty members found</p>
        ) : (
          <div className="table-container">
            <table className="table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Email</th>
                  <th>Role</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {(faculty || []).map((f) => (
                  <tr key={f.regdno}>
                    <td className="text-text-primary font-medium">{f.user?.name}</td>
                    <td className="text-text-secondary">{f.user?.email}</td>
                    <td>
                      <span className="badge badge-neutral">{f.user?.role}</span>
                    </td>
                    <td>
                      <button
                        onClick={() => {
                          setRoleForm({ regdno: f.regdno, new_role: f.user?.role || '' });
                          setShowRoleModal(true);
                        }}
                        className="btn btn-secondary text-xs px-3 py-1.5"
                      >
                        Change Role
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Course Assignments */}
      <div className="card">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold text-text-primary">Course Assignments</h2>
          <button
            onClick={() => fetchAssignments()}
            className="btn btn-secondary text-xs px-3 py-1.5"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
              />
            </svg>
            Refresh
          </button>
        </div>
        {Object.keys(assignmentsByFaculty).length === 0 ? (
          <div className="text-center py-8">
            <p className="text-text-muted mb-2">No course assignments yet</p>
            <p className="text-text-secondary text-sm">
              Use the "Assign Faculty" button above to assign faculty members to courses
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            {Object.entries(assignmentsByFaculty).map(([facultyName, facAssignments]) => (
              <div
                key={facultyName}
                className="p-4 bg-background-tertiary rounded-lg border border-background-border"
              >
                <div className="flex items-center gap-3 mb-3">
                  <div className="w-10 h-10 rounded-full bg-accent-secondary/20 flex items-center justify-center text-accent-secondary font-semibold">
                    {facultyName.charAt(0).toUpperCase()}
                  </div>
                  <span className="font-semibold text-text-primary">{facultyName}</span>
                </div>
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                  {facAssignments.map((assignment, idx) => (
                    <div
                      key={idx}
                      className="p-3 bg-background-elevated rounded-lg border border-background-border"
                    >
                      <div className="text-sm font-medium text-text-primary mb-1">
                        {assignment.course_code}
                      </div>
                      <div className="text-sm text-text-secondary mb-2">
                        {assignment.course_name}
                      </div>
                      <div className="flex flex-wrap gap-1 mt-1">
                        <span className="badge badge-info text-[10px]">
                          Batch {assignment.cohort_year}
                        </span>
                        <span className="badge badge-info text-[10px]">
                          Sec {assignment.section_name}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Assign Faculty Modal */}
      {showAssignModal && (
        <div className="modal-overlay" onClick={() => setShowAssignModal(false)}>
          <div
            className="modal-content max-w-xl"
            onClick={(e: MouseEvent<HTMLDivElement>) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-2xl font-semibold text-text-primary">Assign Faculty to Course</h2>
              <button
                onClick={() => setShowAssignModal(false)}
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
            <form onSubmit={handleAssignSubmit} className="space-y-5">
              <div className="form-group">
                <label className="form-label">Faculty Member</label>
                <select
                  className="select"
                  value={assignForm.faculty_id}
                  onChange={(e: ChangeEvent<HTMLSelectElement>) =>
                    setAssignForm({ ...assignForm, faculty_id: e.target.value })
                  }
                  required
                >
                  <option value="">Select Faculty</option>
                  {(faculty || []).map((f) => (
                    <option key={f.regdno} value={f.regdno}>
                      {f.user?.name} - {f.user?.email}
                    </option>
                  ))}
                </select>
              </div>

              <div className="form-group">
                <label className="form-label">Course</label>
                <select
                  className="select"
                  value={assignForm.course_id}
                  onChange={(e: ChangeEvent<HTMLSelectElement>) => {
                    const val = e.target.value;
                    setAssignForm({ ...assignForm, course_id: val, section_ids: [] });
                    fetchSections(val);
                  }}
                  required
                >
                  <option value="">Select Course</option>
                  {courses.length === 0 && (
                    <option disabled>No courses found — ask admin to create courses first</option>
                  )}
                  {courses.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.course_code} – {c.course_name} ({c.course_type})
                    </option>
                  ))}
                </select>
              </div>

              <div className="form-group">
                <label className="form-label">
                  Sections <span className="text-text-tertiary text-xs">(select multiple)</span>
                </label>
                <div
                  className="bg-background-tertiary border border-background-border rounded-lg p-3 max-h-48 overflow-y-auto space-y-2"
                  style={{ minHeight: '120px' }}
                >
                  {!assignForm.course_id && (
                    <p className="text-text-muted text-sm text-center py-4">
                      Select a course first
                    </p>
                  )}
                  {assignForm.course_id && sections.length === 0 && (
                    <p className="text-text-muted text-sm text-center py-4">
                      No sections found for this course
                    </p>
                  )}
                  {assignForm.course_id &&
                    sections.map((s) => {
                      const checked = assignForm.section_ids.includes(s.section_id);
                      return (
                        <label
                          key={s.section_id}
                          className={`flex items-center gap-3 p-2.5 rounded-lg cursor-pointer transition-colors ${checked ? 'bg-accent-primary/10 border border-accent-primary/30' : 'hover:bg-background-elevated border border-transparent'}`}
                        >
                          <input
                            type="checkbox"
                            className="w-4 h-4 accent-accent-primary shrink-0"
                            checked={checked}
                            onChange={() => {
                              setAssignForm((prev) => ({
                                ...prev,
                                section_ids: checked
                                  ? prev.section_ids.filter((id) => id !== s.section_id)
                                  : [...prev.section_ids, s.section_id],
                              }));
                            }}
                          />
                          <div className="flex-1 min-w-0">
                            <div className="text-sm font-medium text-text-primary">
                              {s.branch_name} – Section {s.section_name}
                            </div>
                            <div className="text-xs text-text-secondary">Batch {s.cohort_year}</div>
                          </div>
                        </label>
                      );
                    })}
                </div>
                <p className="text-text-tertiary text-xs mt-1">
                  {assignForm.section_ids.length > 0
                    ? `${assignForm.section_ids.length} section(s) selected`
                    : 'Select one or more sections to assign'}
                </p>
              </div>

              <div className="flex gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowAssignModal(false)}
                  className="btn btn-secondary flex-1"
                >
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary flex-1">
                  Assign Faculty
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Change Role Modal */}
      {showRoleModal && (
        <div className="modal-overlay" onClick={() => setShowRoleModal(false)}>
          <div
            className="modal-content"
            onClick={(e: MouseEvent<HTMLDivElement>) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-2xl font-semibold text-text-primary">Change Faculty Role</h2>
              <button
                onClick={() => setShowRoleModal(false)}
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
            <form onSubmit={handleRoleSubmit} className="space-y-5">
              <input type="hidden" value={roleForm.regdno} />

              <div className="form-group">
                <label className="form-label">New Role</label>
                <select
                  className="select"
                  value={roleForm.new_role}
                  onChange={(e: ChangeEvent<HTMLSelectElement>) =>
                    setRoleForm({ ...roleForm, new_role: e.target.value })
                  }
                  required
                >
                  <option value="">Select Role</option>
                  <option value="faculty">Faculty</option>
                  <option value="senior_faculty">Senior Faculty</option>
                  <option value="assistant_professor">Assistant Professor</option>
                  <option value="associate_professor">Associate Professor</option>
                  <option value="professor">Professor</option>
                </select>
              </div>

              <div className="flex gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowRoleModal(false)}
                  className="btn btn-secondary flex-1"
                >
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary flex-1">
                  Update Role
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Toast Notification */}
      {toast && (
        <div
          style={{
            position: 'fixed',
            top: '1.5rem',
            right: '1.5rem',
            zIndex: 9999,
            padding: '1rem 1.5rem',
            borderRadius: '0.75rem',
            backgroundColor:
              toast.type === 'error' ? 'var(--error, #ef4444)' : 'var(--success, #22c55e)',
            color: '#fff',
            fontWeight: 500,
            fontSize: '0.9rem',
            boxShadow: '0 10px 25px rgba(0,0,0,0.3)',
            display: 'flex',
            alignItems: 'center',
            gap: '0.75rem',
            animation: 'slideIn 0.3s ease-out',
            maxWidth: '400px',
          }}
        >
          <span>{toast.type === 'error' ? '✕' : '✓'}</span>
          <span>{toast.message}</span>
          <button
            onClick={() => setToast(null)}
            style={{
              marginLeft: 'auto',
              background: 'none',
              border: 'none',
              color: '#fff',
              cursor: 'pointer',
              fontSize: '1.1rem',
              padding: '0 0.25rem',
            }}
          >
            ×
          </button>
        </div>
      )}
    </div>
  );
};

export default HODFaculty;
