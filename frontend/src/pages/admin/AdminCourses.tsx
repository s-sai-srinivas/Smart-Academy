import { useState, useEffect, type ChangeEvent, type FormEvent } from 'react';
import api from '../../services/api';

// ─── Types ───────────────────────────────────────────────────────────
interface ToastItem {
  id: number;
  message: string;
  type: 'success' | 'error' | 'info';
}

interface Course {
  id: number;
  course_code: string;
  course_name: string;
  course_type: 'lab' | 'theory' | 'integrated';
  credits: number;
  description?: string;
  branches?: string;
  batch?: string;
}

interface Branch {
  id: number;
  branch_name: string;
  short_name: string;
}

// ─── Toast component ─────────────────────────────────────────────────
interface ToastProps {
  toasts: ToastItem[];
  removeToast: (id: number) => void;
}

const Toast = ({ toasts, removeToast }: ToastProps) => (
  <div className="fixed top-5 right-5 z-[9999] flex flex-col gap-2 pointer-events-none">
    {toasts.map((t) => (
      <div
        key={t.id}
        className={`pointer-events-auto flex items-center gap-3 px-4 py-3 rounded-lg shadow-lg text-sm font-medium transition-all
          ${t.type === 'success' ? 'bg-green-600 text-white' : t.type === 'error' ? 'bg-red-600 text-white' : 'bg-background-elevated text-text-primary border border-background-border'}`}
        style={{ minWidth: 260 }}
      >
        {t.type === 'success' && (
          <svg className="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
          </svg>
        )}
        {t.type === 'error' && (
          <svg className="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        )}
        <span className="flex-1">{t.message}</span>
        <button
          onClick={() => removeToast(t.id)}
          className="opacity-70 hover:opacity-100 transition-opacity"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        </button>
      </div>
    ))}
  </div>
);

// ─── Confirm dialog component ────────────────────────────────────────
interface ConfirmDialogProps {
  message: string;
  onConfirm: () => void;
  onCancel: () => void;
}

const ConfirmDialog = ({ message, onConfirm, onCancel }: ConfirmDialogProps) => (
  <div className="modal-overlay" onClick={onCancel}>
    <div className="modal-content max-w-sm" onClick={(e) => e.stopPropagation()}>
      <div className="flex items-start gap-4 mb-6">
        <div className="w-10 h-10 rounded-full bg-red-500/10 flex items-center justify-center shrink-0">
          <svg
            className="w-5 h-5 text-red-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 9v2m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"
            />
          </svg>
        </div>
        <div>
          <h3 className="text-lg font-semibold text-text-primary mb-1">Are you sure?</h3>
          <p className="text-text-secondary text-sm">{message}</p>
        </div>
      </div>
      <div className="flex gap-3">
        <button onClick={onCancel} className="btn btn-secondary flex-1">
          Cancel
        </button>
        <button onClick={onConfirm} className="btn btn-danger flex-1">
          Delete
        </button>
      </div>
    </div>
  </div>
);

// ─── Multi-select pill component ─────────────────────────────────────
interface MultiSelectProps<T> {
  label: string;
  options: T[];
  selected: string[];
  onChange: (selected: string[]) => void;
  getLabel: (opt: T) => string;
  getValue: (opt: T) => string | number;
}

const MultiSelect = <T,>({
  label,
  options,
  selected,
  onChange,
  getLabel,
  getValue,
}: MultiSelectProps<T>) => {
  const [open, setOpen] = useState(false);

  const toggle = (val: string) => {
    const exists = selected.includes(val);
    onChange(exists ? selected.filter((v) => v !== val) : [...selected, val]);
  };

  const displayText =
    selected.length === 0
      ? `Select ${label}...`
      : selected.length === options.length
        ? `All ${label}`
        : selected
            .map((v) => {
              const opt = options.find((o) => String(getValue(o)) === String(v));
              return opt ? getLabel(opt) : v;
            })
            .join(', ');

  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="input w-full text-left flex items-center justify-between"
      >
        <span className={selected.length === 0 ? 'text-text-muted' : 'text-text-primary'}>
          {displayText}
        </span>
        <svg
          className={`w-4 h-4 text-text-muted transition-transform ${open ? 'rotate-180' : ''}`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {open && (
        <div className="absolute z-50 w-full mt-1 bg-background-elevated border border-background-border rounded-lg shadow-xl max-h-52 overflow-y-auto">
          {options.length === 0 && (
            <div className="px-3 py-2 text-text-muted text-sm">No options available</div>
          )}
          {options.map((opt) => {
            const val = String(getValue(opt));
            const isSelected = selected.includes(val);
            return (
              <button
                key={val}
                type="button"
                onClick={() => toggle(val)}
                className={`w-full text-left px-3 py-2 text-sm flex items-center gap-2 hover:bg-background-hover transition-colors
                  ${isSelected ? 'text-accent-primary' : 'text-text-primary'}`}
              >
                <span
                  className={`w-4 h-4 rounded border flex items-center justify-center shrink-0
                  ${isSelected ? 'bg-accent-primary border-accent-primary' : 'border-background-border'}`}
                >
                  {isSelected && (
                    <svg
                      className="w-3 h-3 text-white"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={3}
                        d="M5 13l4 4L19 7"
                      />
                    </svg>
                  )}
                </span>
                {getLabel(opt)}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
};

interface FormData {
  course_code: string;
  course_name: string;
  course_type: 'lab' | 'theory' | 'integrated';
  credits: number;
  description: string;
}

const EMPTY_FORM: FormData = {
  course_code: '',
  course_name: '',
  course_type: 'lab',
  credits: 3,
  description: '',
};

const AdminCourses = () => {
  const [courses, setCourses] = useState<Course[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [showModal, setShowModal] = useState<boolean>(false);
  const [editingCourse, setEditingCourse] = useState<Course | null>(null);
  const [formData, setFormData] = useState<FormData>(EMPTY_FORM);
  const [selectedBranches, setSelectedBranches] = useState<string[]>([]);
  const [selectedBatches, setSelectedBatches] = useState<string[]>([]);
  const [branches, setBranches] = useState<Branch[]>([]);
  const [batches, setBatches] = useState<string[]>([]);
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null);

  useEffect(() => {
    fetchCourses();
    fetchBranches();
    fetchBatches();
  }, []);

  const addToast = (message: string, type: ToastItem['type'] = 'success') => {
    const id = Date.now();
    setToasts((prev) => [...prev, { id, message, type }]);
    setTimeout(() => removeToast(id), 4000);
  };

  const removeToast = (id: number) => setToasts((prev) => prev.filter((t) => t.id !== id));

  const fetchCourses = async () => {
    try {
      const response = (await api.get('/admin/courses')) as Course[];
      setCourses(response);
    } catch (error) {
      console.error('Error fetching courses:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchBranches = async () => {
    try {
      const data = (await api.get('/branches')) as Branch[];
      setBranches(data || []);
    } catch (error) {
      console.error('Error fetching branches:', error);
    }
  };

  const fetchBatches = async () => {
    try {
      const data = (await api.get('/batches')) as string[];
      setBatches((data || []).map(String));
    } catch (error) {
      console.error('Error fetching batches:', error);
    }
  };

  const openCreateModal = () => {
    setEditingCourse(null);
    setFormData(EMPTY_FORM);
    setSelectedBranches([]);
    setSelectedBatches([]);
    setShowModal(true);
  };

  const openEditModal = (course: Course) => {
    setEditingCourse(course);
    setFormData({
      course_code: course.course_code,
      course_name: course.course_name,
      course_type: course.course_type || 'lab',
      credits: course.credits || 3,
      description: course.description || '',
    });
    // Parse stored branches/batches if available
    setSelectedBranches(
      course.branches
        ? course.branches
            .split(',')
            .map((s) => s.trim())
            .filter(Boolean)
        : []
    );
    setSelectedBatches(
      course.batch
        ? course.batch
            .split(',')
            .map((s) => s.trim())
            .filter(Boolean)
        : []
    );
    setShowModal(true);
  };

  const closeModal = () => {
    setShowModal(false);
    setEditingCourse(null);
    setFormData(EMPTY_FORM);
    setSelectedBranches([]);
    setSelectedBatches([]);
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    const payload = {
      ...formData,
      credits: parseInt(String(formData.credits)) || 3,
      // send as arrays so backend can auto-create offerings
      branches: selectedBranches,
      batches: selectedBatches.map(Number),
    };
    try {
      if (editingCourse) {
        await api.put(`/admin/courses/${editingCourse.id}`, payload);
        addToast('Course updated successfully!', 'success');
      } else {
        const res = (await api.post('/admin/courses', payload)) as { offerings_created?: number };
        const count = res?.offerings_created ?? 0;
        const detail =
          count > 0 ? ` Visible to ${count} section(s).` : ' No matching sections found yet.';
        addToast('Course created successfully!' + detail, 'success');
      }
      closeModal();
      fetchCourses();
    } catch (error) {
      const msg =
        (error as Error).message ||
        (editingCourse ? 'Failed to update course' : 'Failed to create course');
      if (
        msg.includes('duplicate key') ||
        msg.includes('unique constraint') ||
        msg.includes('already exists')
      ) {
        addToast('A course with this code already exists. Please use a different code.', 'error');
      } else {
        addToast(msg, 'error');
      }
    }
  };

  const handleDeleteConfirmed = async () => {
    const id = confirmDelete;
    setConfirmDelete(null);
    if (id === null) return;
    try {
      await api.delete(`/admin/courses/${id}`);
      addToast('Course deleted.', 'success');
      fetchCourses();
    } catch (error) {
      addToast((error as Error).message || 'Failed to delete course', 'error');
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

  return (
    <div className="space-y-6">
      <Toast toasts={toasts} removeToast={removeToast} />

      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-text-primary mb-1">Courses</h1>
          <p className="text-text-secondary text-sm">Manage theory and lab courses</p>
        </div>
        <button onClick={openCreateModal} className="btn btn-primary">
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Create Course
        </button>
      </div>

      {/* Empty State */}
      {courses.length === 0 ? (
        <div className="card flex flex-col items-center justify-center py-16">
          <h3 className="text-xl font-semibold text-text-primary mb-2">No courses found</h3>
          <p className="text-text-secondary text-sm mb-6">
            Create your first course to get started
          </p>
          <button onClick={openCreateModal} className="btn btn-primary">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 4v16m8-8H4"
              />
            </svg>
            Create Course
          </button>
        </div>
      ) : (
        <div className="table-container">
          <table className="table">
            <thead>
              <tr>
                <th>Code</th>
                <th>Name</th>
                <th>Type</th>
                <th>Credits</th>
                <th className="text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {courses.map((course) => (
                <tr key={course.id}>
                  <td>
                    <span className="font-semibold text-accent-secondary">
                      {course.course_code}
                    </span>
                  </td>
                  <td className="text-text-primary">{course.course_name}</td>
                  <td>
                    <span
                      className={`badge ${course.course_type === 'lab' ? 'badge-success' : 'badge-info'}`}
                    >
                      {course.course_type === 'lab'
                        ? 'Lab'
                        : course.course_type === 'integrated'
                          ? 'Integrated'
                          : 'Theory'}
                    </span>
                  </td>
                  <td className="text-text-secondary">{course.credits}</td>
                  <td className="text-right flex gap-2 justify-end">
                    <button
                      onClick={() => openEditModal(course)}
                      className="btn btn-secondary px-3 py-1.5 text-xs"
                    >
                      <svg
                        className="w-3.5 h-3.5"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                        />
                      </svg>
                      Edit
                    </button>
                    <button
                      onClick={() => setConfirmDelete(course.id)}
                      className="btn btn-danger px-3 py-1.5 text-xs"
                    >
                      <svg
                        className="w-3.5 h-3.5"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6M9 7h6m-7 0a1 1 0 01-1-1V5a1 1 0 011-1h8a1 1 0 011 1v1a1 1 0 01-1 1H5z"
                        />
                      </svg>
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Create / Edit Course Modal */}
      {showModal && (
        <div className="modal-overlay" onClick={closeModal}>
          <div className="modal-content max-w-xl" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-2xl font-semibold text-text-primary">
                {editingCourse ? 'Edit Course' : 'Create New Course'}
              </h2>
              <button
                onClick={closeModal}
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

            <form onSubmit={handleSubmit} className="space-y-5">
              {/* Course Code */}
              <div className="form-group">
                <label className="form-label">Course Code</label>
                <input
                  type="text"
                  className="input"
                  value={formData.course_code}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    setFormData({ ...formData, course_code: e.target.value })
                  }
                  placeholder="e.g. CS101 or CS101L"
                  required
                />
              </div>

              {/* Course Name */}
              <div className="form-group">
                <label className="form-label">Course Name</label>
                <input
                  type="text"
                  className="input"
                  value={formData.course_name}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    setFormData({ ...formData, course_name: e.target.value })
                  }
                  placeholder="e.g. Data Structures Lab"
                  required
                />
              </div>

              {/* Course Type */}
              <div className="form-group">
                <label className="form-label">Course Type</label>
                <div className="flex gap-4">
                  {(['lab', 'theory', 'integrated'] as const).map((type) => (
                    <label key={type} className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="radio"
                        name="course_type"
                        value={type}
                        checked={formData.course_type === type}
                        onChange={(e: ChangeEvent<HTMLInputElement>) =>
                          setFormData({
                            ...formData,
                            course_type: e.target.value as FormData['course_type'],
                          })
                        }
                        className="w-4 h-4"
                      />
                      <span className="text-text-primary capitalize">
                        {type === 'integrated' ? 'Integrated' : type === 'lab' ? 'Lab' : 'Theory'}
                      </span>
                    </label>
                  ))}
                </div>
              </div>

              {/* Credits */}
              <div className="form-group">
                <label className="form-label">Credits</label>
                <input
                  type="number"
                  className="input"
                  value={formData.credits}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    setFormData({ ...formData, credits: Number(e.target.value) })
                  }
                  min="1"
                  max="6"
                />
              </div>

              {/* Branches Multi-select */}
              <div className="form-group">
                <label className="form-label">
                  Branches
                  {selectedBranches.length > 0 && (
                    <span className="ml-2 text-xs text-accent-primary font-normal">
                      ({selectedBranches.length} selected)
                    </span>
                  )}
                </label>
                <MultiSelect
                  label="branches"
                  options={branches}
                  selected={selectedBranches}
                  onChange={setSelectedBranches}
                  getLabel={(b) => `${b.short_name} – ${b.branch_name}`}
                  getValue={(b) => b.short_name}
                />
                <p className="text-text-tertiary text-xs mt-1">
                  Leave empty to apply to all branches
                </p>
              </div>

              {/* Batches Multi-select */}
              <div className="form-group">
                <label className="form-label">
                  Batches (Cohort Years)
                  {selectedBatches.length > 0 && (
                    <span className="ml-2 text-xs text-accent-primary font-normal">
                      ({selectedBatches.length} selected)
                    </span>
                  )}
                </label>
                <MultiSelect
                  label="batches"
                  options={batches}
                  selected={selectedBatches}
                  onChange={setSelectedBatches}
                  getLabel={(b) => String(b)}
                  getValue={(b) => String(b)}
                />
                <p className="text-text-tertiary text-xs mt-1">
                  Leave empty to apply to all batches
                </p>
              </div>

              {/* Description */}
              <div className="form-group">
                <label className="form-label">
                  Description <span className="text-text-muted font-normal">(optional)</span>
                </label>
                <textarea
                  className="input"
                  rows={3}
                  value={formData.description}
                  onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                    setFormData({ ...formData, description: e.target.value })
                  }
                  placeholder="Brief course description..."
                />
              </div>

              {/* Form Actions */}
              <div className="flex gap-3 pt-2">
                <button type="button" onClick={closeModal} className="btn btn-secondary flex-1">
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary flex-1">
                  {editingCourse ? 'Save Changes' : 'Create Course'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Confirm Delete Dialog */}
      {confirmDelete && (
        <ConfirmDialog
          message="This will permanently delete the course and all its associated data (sessions, problems, weeks, modules)."
          onConfirm={handleDeleteConfirmed}
          onCancel={() => setConfirmDelete(null)}
        />
      )}
    </div>
  );
};

export default AdminCourses;
