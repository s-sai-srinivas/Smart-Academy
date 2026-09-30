import { useState, useEffect, useCallback, ChangeEvent, FormEvent } from 'react';
import api from '../../services/api';
import { theoryAPI } from '../../services/api';
import { showError, showSuccess } from '../../utils/showAlert';
import { getErrorMessage } from '../../utils/error';
import Swal from 'sweetalert2';

// ── Types ──

interface Course {
  id: number;
  course_code: string;
  course_name: string;
  course_type: string;
}

interface Module {
  id: number;
  module_name: string;
  description?: string;
  content?: string;
}

interface Week {
  id: number;
  week_name: string;
  week_order: number;
  modules?: Module[];
}

interface Theory {
  id: number;
  theory_name: string;
  theory_code: string;
  weeks?: Week[];
}

interface TheoryPDF {
  id: number;
  theory_id: number;
  theory_module_id?: number;
  file_name: string;
  display_name?: string;
  file_size: number;
  created_at?: string;
  theory_module?: {
    id: number;
    module_name: string;
  };
}

interface JobStatus {
  status: string;
  error_message?: string;
  preview?: Preview;
}

interface PreviewWeek {
  week_number: number;
  week_name: string;
  module_count: number;
}

interface Preview {
  total_weeks: number;
  weeks: PreviewWeek[];
}

interface WeekForm {
  week_name: string;
  week_order: number;
}

interface ModuleForm {
  module_name: string;
  description: string;
  content: string;
}

const AdminTheory = () => {
  const [theoryCourses, setTheoryCourses] = useState<Course[]>([]);
  const [selectedCourse, setSelectedCourse] = useState<string>('');
  const [theory, setTheory] = useState<Theory | null>(null);
  const [loading, setLoading] = useState<boolean>(true);

  // Tabs
  const [activeTab, setActiveTab] = useState<'curriculum' | 'pdfs'>('curriculum');

  // PDFs
  const [pdfs, setPdfs] = useState<TheoryPDF[]>([]);
  const [pdfFile, setPdfFile] = useState<File | null>(null);
  const [uploadingPdf, setUploadingPdf] = useState<boolean>(false);
  const [showPdfUpload, setShowPdfUpload] = useState<boolean>(false);

  // Modals
  const [showWeekModal, setShowWeekModal] = useState<boolean>(false);
  const [showModuleModal, setShowModuleModal] = useState<boolean>(false);
  const [showAIDialog, setShowAIDialog] = useState<boolean>(false);
  const [activeWeekId, setActiveWeekId] = useState<number | null>(null);

  // AI Upload state
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState<boolean>(false);
  const [jobId, setJobId] = useState<number | null>(null);
  const [jobStatus, setJobStatus] = useState<JobStatus | null>(null);
  const [preview, setPreview] = useState<Preview | null>(null);

  // Forms
  const [weekForm, setWeekForm] = useState<WeekForm>({ week_name: '', week_order: 1 });
  const [moduleForm, setModuleForm] = useState<ModuleForm>({
    module_name: '',
    description: '',
    content: '',
  });
  const [editingModule, setEditingModule] = useState<Module | null>(null);
  const [modulePdfFile, setModulePdfFile] = useState<File | null>(null);

  useEffect(() => {
    fetchTheoryCourses();
  }, []);

  useEffect(() => {
    if (selectedCourse) {
      fetchTheory(selectedCourse);
      fetchPDFs(selectedCourse);
    } else {
      setTheory(null);
      setPdfs([]);
    }
  }, [selectedCourse]);

  const fetchTheoryCourses = async () => {
    try {
      const response = (await api.get('/admin/courses')) as Course[];
      const theoryOnly = response.filter(
        (c) => c.course_type === 'theory' || c.course_type === 'integrated'
      );
      setTheoryCourses(theoryOnly);
      setLoading(false);
    } catch (error) {
      console.error('Error fetching courses:', error);
      setLoading(false);
    }
  };

  const fetchTheory = async (courseId: string) => {
    try {
      const response = (await api.get(`/admin/courses/${courseId}/theory`)) as Theory;
      setTheory(response);
    } catch (error) {
      console.error('Error fetching theory:', error);
      setTheory(null);
    }
  };

  const fetchPDFs = async (courseId: string) => {
    try {
      const response = (await api.get(`/admin/courses/${courseId}/theory`)) as Theory;
      if (response?.id) {
        const data = (await theoryAPI.getPDFs(response.id)) as { pdfs?: TheoryPDF[] };
        setPdfs(data.pdfs || []);
      }
    } catch (error) {
      console.error('Error fetching PDFs:', error);
      setPdfs([]);
    }
  };

  const fetchJobStatus = useCallback(async () => {
    if (jobId === null) return;
    try {
      const data = (await api.get(`/admin/lesson-plans/jobs/${jobId}`)) as JobStatus;
      setJobStatus(data);
      if (data.status === 'completed' && data.preview) {
        setPreview(data.preview);
      }
    } catch (error) {
      console.error('Error fetching job status:', error);
    }
  }, [jobId]);

  // Poll for AI job status
  useEffect(() => {
    if (jobId && (jobStatus?.status === 'pending' || jobStatus?.status === 'processing')) {
      const interval = setInterval(() => {
        fetchJobStatus();
      }, 3000);
      return () => clearInterval(interval);
    }
  }, [jobId, jobStatus, fetchJobStatus]);

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      const allowedTypes = ['.pdf', '.docx', '.txt'];
      const fileExt = '.' + file.name.split('.').pop()?.toLowerCase();
      if (!fileExt || !allowedTypes.includes(fileExt)) {
        showError(`Invalid file type. Please upload PDF, DOCX, or TXT file.`);
        return;
      }
      if (file.size > 50 * 1024 * 1024) {
        showError('File too large. Maximum size is 50MB.');
        return;
      }
      setSelectedFile(file);
    }
  };

  const handleAIUpload = async () => {
    if (!selectedFile) {
      showError('Please select a lesson plan file');
      return;
    }
    if (!selectedCourse) {
      showError('Please select a course');
      return;
    }

    setUploading(true);
    try {
      const formData = new FormData();
      formData.append('lesson_plan', selectedFile);
      formData.append('course_id', selectedCourse);

      const result = (await api.post('/admin/lesson-plans/upload', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })) as { job_id: number; status: string };

      setJobId(result.job_id);
      setJobStatus({ status: result.status });
    } catch (error: unknown) {
      showError(getErrorMessage(error, 'Upload failed'));
    } finally {
      setUploading(false);
    }
  };

  const handleApproveGeneration = async () => {
    const result = await Swal.fire({
      title: 'Approve AI Generated Content?',
      html: `<p>This will create ${preview?.total_weeks || 0} weeks with modules and practice quizzes.</p>`,
      icon: 'question',
      showCancelButton: true,
      confirmButtonText: 'Yes, Approve',
      cancelButtonText: 'Cancel',
    });

    if (!result.isConfirmed) return;

    try {
      await api.post(`/admin/lesson-plans/jobs/${jobId}/approve`, { modifications: null });
      showSuccess('Course content generated successfully!');
      setShowAIDialog(false);
      setJobId(null);
      setJobStatus(null);
      setPreview(null);
      setSelectedFile(null);
      fetchTheory(selectedCourse);
    } catch (error: unknown) {
      showError(getErrorMessage(error, 'Failed to save content'));
    }
  };

  const handleCreateWeek = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    try {
      await api.post('/admin/theories/weeks', {
        theory_id: theory?.id,
        week_name: weekForm.week_name,
        week_order: weekForm.week_order,
      });
      setShowWeekModal(false);
      setWeekForm({ week_name: '', week_order: (theory?.weeks?.length || 0) + 1 });
      showSuccess('Week created successfully');
      fetchTheory(selectedCourse);
    } catch (error: unknown) {
      showError(getErrorMessage(error, 'Failed to create week'));
    }
  };

  const handleCreateModule = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    try {
      const payload: Record<string, unknown> = {
        theory_week_id: activeWeekId,
        module_name: moduleForm.module_name,
        description: moduleForm.description,
        content: moduleForm.content,
      };
      const created = (await api.post('/admin/weeks/modules', payload)) as Module;

      // Upload PDF if a file was selected
      if (modulePdfFile && theory?.id && created?.id) {
        try {
          await theoryAPI.uploadPDF(theory.id, modulePdfFile, created.id);
        } catch (err) {
          console.error('PDF upload failed:', err);
        }
      }

      setShowModuleModal(false);
      setModuleForm({ module_name: '', description: '', content: '' });
      setModulePdfFile(null);
      fetchTheory(selectedCourse);
      fetchPDFs(selectedCourse);
    } catch (error: unknown) {
      showError(getErrorMessage(error, 'Failed to create module'));
    }
  };

  const handleEditModule = (module: Module) => {
    setEditingModule(module);
    setModuleForm({
      module_name: module.module_name,
      description: module.description || '',
      content: module.content || '',
    });
    setModulePdfFile(null);
    setShowModuleModal(true);
  };

  const handleUpdateModule = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    try {
      await api.put(`/admin/weeks/modules/${editingModule?.id}`, {
        module_name: moduleForm.module_name,
        description: moduleForm.description,
        content: moduleForm.content,
      });

      if (modulePdfFile && theory?.id && editingModule?.id) {
        try {
          await theoryAPI.uploadPDF(theory.id, modulePdfFile, editingModule.id);
        } catch (err) {
          console.error('PDF upload failed:', err);
        }
      }

      setShowModuleModal(false);
      setEditingModule(null);
      setModuleForm({ module_name: '', description: '', content: '' });
      setModulePdfFile(null);
      showSuccess('Module updated successfully');
      fetchTheory(selectedCourse);
      fetchPDFs(selectedCourse);
    } catch (error: unknown) {
      showError(getErrorMessage(error, 'Failed to update module'));
    }
  };

  const handleDeleteWeek = async (weekId: number, weekName: string) => {
    const result = await Swal.fire({
      title: 'Delete Week?',
      html: `Are you sure you want to delete "<strong>${weekName}</strong>"? All modules and practice quizzes in this week will be deleted.`,
      icon: 'warning',
      showCancelButton: true,
      confirmButtonText: 'Yes, Delete',
      cancelButtonText: 'Cancel',
      confirmButtonColor: '#dc2626',
    });

    if (!result.isConfirmed) return;

    try {
      await api.delete(`/admin/theories/weeks/${weekId}`);
      showSuccess('Week deleted successfully');
      fetchTheory(selectedCourse);
    } catch (error: unknown) {
      showError(getErrorMessage(error, 'Failed to delete week'));
    }
  };

  const handleDeleteModule = async (_weekId: number, moduleId: number, moduleName: string) => {
    const result = await Swal.fire({
      title: 'Delete Module?',
      html: `Are you sure you want to delete "<strong>${moduleName}</strong>"?`,
      icon: 'warning',
      showCancelButton: true,
      confirmButtonText: 'Yes, Delete',
      cancelButtonText: 'Cancel',
      confirmButtonColor: '#dc2626',
    });

    if (!result.isConfirmed) return;

    try {
      await api.delete(`/admin/weeks/modules/${moduleId}`);
      showSuccess('Module deleted successfully');
      fetchTheory(selectedCourse);
      fetchPDFs(selectedCourse);
    } catch (error: unknown) {
      showError(getErrorMessage(error, 'Failed to delete module'));
    }
  };

  // PDF handlers
  const handlePdfFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      if (!file.name.toLowerCase().endsWith('.pdf')) {
        showError('Only PDF files are allowed');
        return;
      }
      if (file.size > 50 * 1024 * 1024) {
        showError('File too large. Maximum size is 50MB.');
        return;
      }
      setPdfFile(file);
    }
  };

  const handleUploadPDF = async (moduleId?: number) => {
    if (!pdfFile || !theory?.id) {
      showError('Please select a PDF file');
      return;
    }
    setUploadingPdf(true);
    try {
      await theoryAPI.uploadPDF(theory.id, pdfFile, moduleId);
      showSuccess('PDF uploaded successfully');
      setShowPdfUpload(false);
      setPdfFile(null);
      fetchPDFs(selectedCourse);
    } catch (error: unknown) {
      showError(getErrorMessage(error, 'Failed to upload PDF'));
    } finally {
      setUploadingPdf(false);
    }
  };

  const handleDeletePDF = async (pdfId: number, fileName: string) => {
    const result = await Swal.fire({
      title: 'Delete PDF?',
      html: `Are you sure you want to delete "<strong>${fileName}</strong>"?`,
      icon: 'warning',
      showCancelButton: true,
      confirmButtonText: 'Yes, Delete',
      cancelButtonText: 'Cancel',
      confirmButtonColor: '#dc2626',
    });

    if (!result.isConfirmed) return;

    try {
      await theoryAPI.deletePDF(pdfId);
      showSuccess('PDF deleted successfully');
      fetchPDFs(selectedCourse);
    } catch (error: unknown) {
      showError(getErrorMessage(error, 'Failed to delete PDF'));
    }
  };

  const formatFileSize = (bytes: number): string => {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(2) + ' MB';
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
        <h1 className="text-3xl font-bold text-text-primary mb-1">Theory Management</h1>
        <p className="text-text-secondary text-sm">
          Manage theory curriculum - weeks, modules, and PDF resources for theory courses
        </p>
      </div>

      {/* Theory Course Selector */}
      <div className="card">
        <div className="form-group">
          <label className="form-label">Select Theory Course</label>
          <select
            className="select"
            value={selectedCourse}
            onChange={(e: ChangeEvent<HTMLSelectElement>) => setSelectedCourse(e.target.value)}
          >
            <option value="">-- Select a Theory Course --</option>
            {theoryCourses.map((course) => (
              <option key={course.id} value={course.id}>
                {course.course_code} - {course.course_name}
              </option>
            ))}
          </select>
          {theoryCourses.length === 0 && (
            <p className="text-text-tertiary text-sm mt-2">
              No theory courses found. Create a theory course in the Courses page first.
            </p>
          )}
        </div>
      </div>

      {/* Theory Content with Tabs */}
      {selectedCourse && theory && (
        <div className="card">
          <div className="flex items-center justify-between mb-6 pb-4 border-b border-background-border">
            <div>
              <h2 className="text-xl font-semibold text-text-primary">{theory.theory_name}</h2>
              <p className="text-text-secondary text-sm mt-1">Code: {theory.theory_code}</p>
            </div>
            <div className="flex gap-2">
              <button className="btn btn-accent" onClick={() => setShowAIDialog(true)}>
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M13 10V3L4 14h7v7l9-11h-7z"
                  />
                </svg>
                Generate with AI
              </button>
              {activeTab === 'curriculum' && (
                <button
                  className="btn btn-primary"
                  onClick={() => {
                    setWeekForm((prev) => ({
                      ...prev,
                      week_order: (theory.weeks?.length || 0) + 1,
                    }));
                    setShowWeekModal(true);
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
                  Add Week
                </button>
              )}
              {activeTab === 'pdfs' && (
                <button
                  className="btn btn-primary"
                  onClick={() => {
                    setPdfFile(null);
                    setShowPdfUpload(true);
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
                  Upload PDF
                </button>
              )}
            </div>
          </div>

          {/* Tab Buttons */}
          <div className="flex gap-1 mb-6 bg-background-tertiary rounded-lg p-1">
            <button
              onClick={() => setActiveTab('curriculum')}
              className={`flex-1 px-4 py-2 text-sm font-medium rounded-md transition-colors ${
                activeTab === 'curriculum'
                  ? 'bg-background-elevated text-text-primary shadow-sm'
                  : 'text-text-secondary hover:text-text-primary'
              }`}
            >
              Curriculum
            </button>
            <button
              onClick={() => setActiveTab('pdfs')}
              className={`flex-1 px-4 py-2 text-sm font-medium rounded-md transition-colors ${
                activeTab === 'pdfs'
                  ? 'bg-background-elevated text-text-primary shadow-sm'
                  : 'text-text-secondary hover:text-text-primary'
              }`}
            >
              PDFs{' '}
              {pdfs.length > 0 && (
                <span className="ml-1.5 px-1.5 py-0.5 text-xs rounded-full bg-accent-primary/20 text-accent-primary">
                  {pdfs.length}
                </span>
              )}
            </button>
          </div>

          {/* Curriculum Tab */}
          {activeTab === 'curriculum' && (
            <>
              {theory.weeks && theory.weeks.length > 0 ? (
                <div className="space-y-4">
                  {theory.weeks.map((week) => (
                    <div
                      key={week.id}
                      className="bg-background-tertiary rounded-lg border border-background-border p-4"
                    >
                      <div className="flex items-center justify-between mb-4">
                        <div className="flex items-center gap-2">
                          <span className="font-semibold text-text-primary">
                            Week {week.week_order}: {week.week_name}
                          </span>
                        </div>
                        <div className="flex gap-2">
                          <button
                            className="btn btn-danger text-xs px-3 py-1.5"
                            onClick={() => handleDeleteWeek(week.id, week.week_name)}
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
                                d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                              />
                            </svg>
                            Delete Week
                          </button>
                          <button
                            className="btn btn-secondary text-xs px-3 py-1.5"
                            onClick={() => {
                              setActiveWeekId(week.id);
                              setEditingModule(null);
                              setModuleForm({ module_name: '', description: '', content: '' });
                              setModulePdfFile(null);
                              setShowModuleModal(true);
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
                            Add Module
                          </button>
                        </div>
                      </div>
                      {week.modules && week.modules.length > 0 ? (
                        <div className="pl-4 space-y-2">
                          {week.modules.map((module) => (
                            <div
                              key={module.id}
                              className="bg-background-elevated rounded-lg p-3 flex items-center justify-between"
                            >
                              <div>
                                <span className="font-medium text-text-primary">
                                  {module.module_name}
                                </span>
                                {module.description && (
                                  <p className="text-text-secondary text-sm mt-1">
                                    {module.description}
                                  </p>
                                )}
                              </div>
                              <div className="flex gap-2">
                                <button
                                  className="btn btn-secondary text-xs px-3 py-1.5"
                                  onClick={() => handleEditModule(module)}
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
                                      d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                                    />
                                  </svg>
                                  Edit
                                </button>
                                <button
                                  className="btn btn-danger text-xs px-3 py-1.5"
                                  onClick={() =>
                                    handleDeleteModule(week.id, module.id, module.module_name)
                                  }
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
                                      d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                                    />
                                  </svg>
                                  Delete
                                </button>
                              </div>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <p className="text-text-muted italic text-sm">No modules added yet.</p>
                      )}
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-12">
                  <h3 className="text-lg font-semibold text-text-primary mb-2">
                    No Curriculum Weeks
                  </h3>
                  <p className="text-text-secondary text-sm mb-4">
                    Create your first week for this theory course
                  </p>
                  <button
                    className="btn btn-primary"
                    onClick={() => {
                      setWeekForm((prev) => ({ ...prev, week_order: 1 }));
                      setShowWeekModal(true);
                    }}
                  >
                    Create Week
                  </button>
                </div>
              )}
            </>
          )}

          {/* PDFs Tab */}
          {activeTab === 'pdfs' && (
            <>
              {pdfs.length === 0 ? (
                <div className="text-center py-12">
                  <svg
                    className="w-12 h-12 mx-auto text-text-muted mb-3"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
                    />
                  </svg>
                  <p className="text-text-secondary">No PDFs uploaded yet</p>
                  <button
                    onClick={() => {
                      setPdfFile(null);
                      setShowPdfUpload(true);
                    }}
                    className="mt-3 text-accent-secondary hover:underline text-sm"
                  >
                    Upload your first PDF
                  </button>
                </div>
              ) : (
                <div className="space-y-3">
                  {pdfs.map((pdf) => (
                    <div
                      key={pdf.id}
                      className="bg-background-tertiary rounded-lg border border-background-border p-4 flex items-center justify-between"
                    >
                      <div className="flex items-center gap-3 min-w-0 flex-1">
                        <div className="w-10 h-10 rounded-lg bg-red-500/10 flex items-center justify-center shrink-0">
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
                              d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
                            />
                          </svg>
                        </div>
                        <div className="min-w-0">
                          <p className="font-medium text-text-primary truncate">{pdf.file_name}</p>
                          <div className="flex items-center gap-2 mt-1">
                            <span className="text-xs text-text-muted">
                              {formatFileSize(pdf.file_size)}
                            </span>
                            {pdf.theory_module && (
                              <span className="badge badge-info text-[10px]">
                                {pdf.theory_module.module_name}
                              </span>
                            )}
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <a
                          href={`/api/theory-pdfs/${pdf.id}`}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="btn btn-secondary text-xs px-3 py-1.5"
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
                              d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                            />
                          </svg>
                          Download
                        </a>
                        <button
                          className="btn btn-danger text-xs px-3 py-1.5"
                          onClick={() => handleDeletePDF(pdf.id, pdf.file_name)}
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
                              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                            />
                          </svg>
                          Delete
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </>
          )}
        </div>
      )}

      {/* Week Modal */}
      {showWeekModal && (
        <div className="modal-overlay" onClick={() => setShowWeekModal(false)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-2xl font-semibold text-text-primary">Add Curriculum Week</h2>
              <button
                onClick={() => setShowWeekModal(false)}
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
            <form onSubmit={handleCreateWeek} className="space-y-5">
              <div className="form-group">
                <label className="form-label">Week Name</label>
                <input
                  type="text"
                  className="input"
                  value={weekForm.week_name}
                  onChange={(e) => setWeekForm({ ...weekForm, week_name: e.target.value })}
                  placeholder="e.g. Introduction to Algorithms"
                  required
                />
              </div>
              <div className="form-group">
                <label className="form-label">Week Order</label>
                <input
                  type="number"
                  className="input"
                  value={weekForm.week_order}
                  onChange={(e) =>
                    setWeekForm({ ...weekForm, week_order: parseInt(e.target.value) || 0 })
                  }
                  min={1}
                  required
                />
              </div>
              <div className="flex gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowWeekModal(false)}
                  className="btn btn-secondary flex-1"
                >
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary flex-1">
                  Add Week
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Module Modal */}
      {showModuleModal && (
        <div
          className="modal-overlay"
          onClick={() => {
            setShowModuleModal(false);
            setEditingModule(null);
            setModuleForm({ module_name: '', description: '', content: '' });
            setModulePdfFile(null);
          }}
        >
          <div className="modal-content max-w-2xl" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-2xl font-semibold text-text-primary">
                {editingModule ? 'Edit Module' : 'Add Learning Module'}
              </h2>
              <button
                onClick={() => {
                  setShowModuleModal(false);
                  setEditingModule(null);
                  setModuleForm({ module_name: '', description: '', content: '' });
                  setModulePdfFile(null);
                }}
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
            <form
              onSubmit={editingModule ? handleUpdateModule : handleCreateModule}
              className="space-y-5"
            >
              <div className="form-group">
                <label className="form-label">Module Name</label>
                <input
                  type="text"
                  className="input"
                  value={moduleForm.module_name}
                  onChange={(e) => setModuleForm({ ...moduleForm, module_name: e.target.value })}
                  placeholder="e.g. BFS Algorithm"
                  required
                />
              </div>
              <div className="form-group">
                <label className="form-label">Description / Learning Objectives</label>
                <textarea
                  rows={3}
                  className="input resize-none"
                  value={moduleForm.description}
                  onChange={(e) => setModuleForm({ ...moduleForm, description: e.target.value })}
                />
              </div>
              <div className="form-group">
                <label className="form-label">Content (Markdown supported)</label>
                <textarea
                  rows={5}
                  className="input resize-none"
                  value={moduleForm.content}
                  onChange={(e) => setModuleForm({ ...moduleForm, content: e.target.value })}
                />
              </div>
              <div className="form-group">
                <label className="form-label">Attach PDF (optional)</label>
                <div
                  className="border-2 border-dashed border-background-border rounded-lg p-4 text-center cursor-pointer hover:border-accent-primary transition-colors"
                  onClick={() => document.getElementById('module-pdf-file')?.click()}
                >
                  <input
                    id="module-pdf-file"
                    type="file"
                    accept=".pdf"
                    onChange={(e: ChangeEvent<HTMLInputElement>) => {
                      const file = e.target.files?.[0];
                      if (file) {
                        if (!file.name.toLowerCase().endsWith('.pdf')) {
                          showError('Only PDF files are allowed');
                          return;
                        }
                        if (file.size > 50 * 1024 * 1024) {
                          showError('File too large. Maximum size is 50MB.');
                          return;
                        }
                        setModulePdfFile(file);
                      }
                    }}
                    className="hidden"
                  />
                  {modulePdfFile ? (
                    <div>
                      <svg
                        className="w-8 h-8 mx-auto text-accent-primary mb-2"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                        />
                      </svg>
                      <p className="text-sm text-accent-primary font-medium">
                        {modulePdfFile.name}
                      </p>
                      <p className="text-xs text-text-muted mt-1">
                        {formatFileSize(modulePdfFile.size)}
                      </p>
                    </div>
                  ) : (
                    <div>
                      <svg
                        className="w-8 h-8 mx-auto text-text-muted mb-2"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
                        />
                      </svg>
                      <p className="text-sm text-text-secondary">Click to attach PDF</p>
                      <p className="text-xs text-text-muted mt-1">Max 50MB</p>
                    </div>
                  )}
                </div>
              </div>
              <div className="flex gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => {
                    setShowModuleModal(false);
                    setEditingModule(null);
                    setModuleForm({ module_name: '', description: '', content: '' });
                    setModulePdfFile(null);
                  }}
                  className="btn btn-secondary flex-1"
                >
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary flex-1">
                  {editingModule ? 'Save Changes' : 'Add Module'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* PDF Upload Modal */}
      {showPdfUpload && (
        <div
          className="modal-overlay"
          onClick={() => {
            setShowPdfUpload(false);
            setPdfFile(null);
          }}
        >
          <div className="modal-content max-w-md" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-2xl font-semibold text-text-primary">Upload PDF</h2>
              <button
                onClick={() => {
                  setShowPdfUpload(false);
                  setPdfFile(null);
                }}
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
            <div className="space-y-5">
              <div className="form-group">
                <label className="form-label">Select PDF File</label>
                <div
                  className="border-2 border-dashed border-background-border rounded-lg p-8 text-center cursor-pointer hover:border-accent-primary transition-colors"
                  onClick={() => document.getElementById('pdf-upload-file')?.click()}
                >
                  <input
                    id="pdf-upload-file"
                    type="file"
                    accept=".pdf"
                    onChange={handlePdfFileChange}
                    className="hidden"
                  />
                  {pdfFile ? (
                    <div>
                      <svg
                        className="w-10 h-10 mx-auto text-accent-primary mb-3"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                        />
                      </svg>
                      <p className="font-semibold text-accent-primary">{pdfFile.name}</p>
                      <p className="text-sm text-text-muted mt-1">{formatFileSize(pdfFile.size)}</p>
                    </div>
                  ) : (
                    <div>
                      <svg
                        className="w-10 h-10 mx-auto text-text-muted mb-3"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
                        />
                      </svg>
                      <p className="text-text-secondary">Click to select PDF file</p>
                      <p className="text-sm text-text-muted mt-1">Max 50MB</p>
                    </div>
                  )}
                </div>
              </div>

              <div className="flex gap-3">
                <button
                  type="button"
                  onClick={() => {
                    setShowPdfUpload(false);
                    setPdfFile(null);
                  }}
                  className="btn btn-secondary flex-1"
                >
                  Cancel
                </button>
                <button
                  className="btn btn-primary flex-1"
                  onClick={() => handleUploadPDF()}
                  disabled={uploadingPdf || !pdfFile}
                >
                  {uploadingPdf ? 'Uploading...' : 'Upload PDF'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* AI Generation Dialog */}
      {showAIDialog && (
        <div className="modal-overlay" onClick={() => setShowAIDialog(false)}>
          <div className="modal-content max-w-2xl" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-2xl font-semibold text-text-primary">
                <span className="text-accent-primary">✨</span> Generate Course Content with AI
              </h2>
              <button
                onClick={() => {
                  setShowAIDialog(false);
                  if (preview) {
                    setJobId(null);
                    setJobStatus(null);
                    setPreview(null);
                    setSelectedFile(null);
                  }
                }}
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

            {!jobId ? (
              <>
                <div className="bg-background-tertiary rounded-lg p-4 mb-6">
                  <h3 className="font-semibold text-text-primary mb-2">How it works:</h3>
                  <ul className="space-y-2 text-sm text-text-secondary">
                    <li>📄 Upload your lesson plan (PDF, DOCX, or TXT)</li>
                    <li>🤖 AI analyzes and extracts week-by-week structure</li>
                    <li>📚 Auto-generates modules with content for each week</li>
                    <li>📝 Creates practice quizzes for each week</li>
                    <li>✅ You review and approve before saving</li>
                  </ul>
                </div>

                <div className="form-group">
                  <label className="form-label">Upload Lesson Plan</label>
                  <div
                    className="border-2 border-dashed border-background-border rounded-lg p-8 text-center cursor-pointer hover:border-accent-primary transition-colors"
                    onClick={() => document.getElementById('lesson-plan-file')?.click()}
                  >
                    <input
                      id="lesson-plan-file"
                      type="file"
                      accept=".pdf,.docx,.txt"
                      onChange={handleFileChange}
                      className="hidden"
                      disabled={uploading}
                    />
                    {selectedFile ? (
                      <div>
                        <p className="font-semibold text-accent-primary">{selectedFile.name}</p>
                        <p className="text-sm text-text-muted mt-1">
                          {(selectedFile.size / 1024 / 1024).toFixed(2)} MB
                        </p>
                      </div>
                    ) : (
                      <div>
                        <svg
                          className="w-12 h-12 mx-auto text-text-muted mb-3"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
                          />
                        </svg>
                        <p className="text-text-secondary">Click to browse or drag file here</p>
                        <p className="text-sm text-text-muted mt-1">PDF, DOCX, TXT (max 50MB)</p>
                      </div>
                    )}
                  </div>
                </div>

                <div className="flex gap-3 pt-4">
                  <button
                    type="button"
                    onClick={() => setShowAIDialog(false)}
                    className="btn btn-secondary flex-1"
                    disabled={uploading}
                  >
                    Cancel
                  </button>
                  <button
                    className="btn btn-primary flex-1"
                    onClick={handleAIUpload}
                    disabled={uploading || !selectedFile}
                  >
                    {uploading ? (
                      <>
                        <svg className="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
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
                        Processing...
                      </>
                    ) : (
                      <>
                        <svg
                          className="w-4 h-4"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M13 10V3L4 14h7v7l9-11h-7z"
                          />
                        </svg>
                        Generate Content
                      </>
                    )}
                  </button>
                </div>
              </>
            ) : jobStatus?.status === 'pending' || jobStatus?.status === 'processing' ? (
              <div className="text-center py-12">
                <svg
                  className="animate-spin h-16 w-16 mx-auto text-accent-primary mb-4"
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
                <h3 className="text-lg font-semibold text-text-primary mb-2">
                  Processing Your Lesson Plan
                </h3>
                <p className="text-text-secondary text-sm">Status: {jobStatus?.status}</p>
                <p className="text-text-muted text-sm mt-2">This usually takes 2-5 minutes...</p>
              </div>
            ) : jobStatus?.status === 'failed' ? (
              <div className="text-center py-8">
                <p className="text-error-primary mb-4">{jobStatus?.error_message}</p>
                <button className="btn btn-primary" onClick={() => setJobId(null)}>
                  Try Again
                </button>
              </div>
            ) : preview ? (
              <div>
                <div className="bg-accent-subtle rounded-lg p-4 mb-6">
                  <h3 className="font-semibold text-text-primary mb-2">Preview</h3>
                  <div className="grid grid-cols-3 gap-4">
                    <div className="text-center">
                      <p className="text-2xl font-bold text-accent-primary">
                        {preview.total_weeks || 0}
                      </p>
                      <p className="text-sm text-text-muted">Weeks</p>
                    </div>
                    <div className="text-center">
                      <p className="text-2xl font-bold text-accent-primary">
                        {preview.weeks?.reduce((acc, w) => acc + (w.module_count || 0), 0) || 0}
                      </p>
                      <p className="text-sm text-text-muted">Modules</p>
                    </div>
                    <div className="text-center">
                      <p className="text-2xl font-bold text-accent-primary">
                        {preview.total_weeks || 0}
                      </p>
                      <p className="text-sm text-text-muted">Practice Quizzes</p>
                    </div>
                  </div>
                </div>

                <div className="max-h-64 overflow-y-auto mb-6 space-y-2">
                  {preview.weeks?.slice(0, 3).map((week, i) => (
                    <div key={i} className="bg-background-tertiary rounded-lg p-3">
                      <p className="font-semibold text-text-primary">
                        Week {week.week_number}: {week.week_name}
                      </p>
                      <p className="text-sm text-text-muted mt-1">{week.module_count} modules</p>
                    </div>
                  ))}
                  {preview.weeks?.length > 3 && (
                    <p className="text-sm text-text-muted text-center">
                      + {preview.weeks.length - 3} more weeks
                    </p>
                  )}
                </div>

                <div className="flex gap-3">
                  <button
                    type="button"
                    onClick={() => {
                      setJobId(null);
                      setJobStatus(null);
                      setPreview(null);
                      setSelectedFile(null);
                    }}
                    className="btn btn-secondary flex-1"
                  >
                    Cancel
                  </button>
                  <button className="btn btn-primary flex-1" onClick={handleApproveGeneration}>
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M5 13l4 4L19 7"
                      />
                    </svg>
                    Approve & Save
                  </button>
                </div>
              </div>
            ) : null}
          </div>
        </div>
      )}
    </div>
  );
};

export default AdminTheory;
