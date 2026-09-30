import { useState, useEffect, FormEvent, ChangeEvent, DragEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { lessonPlanAPI, coursesAPI } from '../../services/api';
import { getErrorMessage } from '../../utils/error';
import Swal from 'sweetalert2';

interface Course {
  id: string;
  course_name: string;
  course_level: number;
  semester: number;
  course_type?: string;
}

interface UploadResult {
  job_id: string;
  estimated_time?: string;
}

const LessonPlanUpload = () => {
  const navigate = useNavigate();
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [courseId, setCourseId] = useState<string>('');
  const [uploading, setUploading] = useState<boolean>(false);
  const [, setProgress] = useState<string>('');
  const [courses, setCourses] = useState<Course[]>([]);
  const [loadingCourses, setLoadingCourses] = useState<boolean>(true);

  useEffect(() => {
    fetchCourses();
  }, []);

  const fetchCourses = async () => {
    try {
      const data = (await coursesAPI.getAll()) as Course[];
      const theoryCourses = data.filter((course) => course.course_type === 'theory');
      setCourses(theoryCourses);
    } catch (error) {
      console.error('Error fetching courses:', error);
      Swal.fire({
        icon: 'error',
        title: 'Error',
        text: 'Failed to load courses',
      });
    } finally {
      setLoadingCourses(false);
    }
  };

  const validateFile = (file: File): boolean => {
    const allowedTypes = ['.pdf', '.docx', '.txt'];
    const fileExt = '.' + file.name.split('.').pop()?.toLowerCase();

    if (!fileExt || !allowedTypes.includes(fileExt)) {
      Swal.fire({
        icon: 'error',
        title: 'Invalid File Type',
        text: `Please upload a PDF, DOCX, or TXT file. Received: ${fileExt || 'unknown'}`,
      });
      return false;
    }

    if (file.size > 50 * 1024 * 1024) {
      Swal.fire({
        icon: 'error',
        title: 'File Too Large',
        text: 'Maximum file size is 50MB',
      });
      return false;
    }

    return true;
  };

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file && validateFile(file)) {
      setSelectedFile(file);
    }
  };

  const handleDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    const file = e.dataTransfer.files[0];
    if (file && validateFile(file)) {
      setSelectedFile(file);
    }
  };

  const handleDragOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
  };

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();

    if (!selectedFile) {
      Swal.fire({
        icon: 'error',
        title: 'No File Selected',
        text: 'Please select a lesson plan file to upload',
      });
      return;
    }

    if (!courseId) {
      Swal.fire({
        icon: 'error',
        title: 'Course Required',
        text: 'Please select a course',
      });
      return;
    }

    setUploading(true);
    setProgress('Uploading lesson plan...');

    try {
      const result = (await lessonPlanAPI.upload(selectedFile, parseInt(courseId))) as UploadResult;

      setProgress('Upload complete! Processing started...');

      Swal.fire({
        icon: 'success',
        title: 'Upload Successful',
        html: `
                    <p>Your lesson plan is being processed.</p>
                    <p>Job ID: ${result.job_id}</p>
                    <p>Estimated time: ${result.estimated_time || '2-5 minutes'}</p>
                `,
        timer: 3000,
      });

      navigate(`/admin/lesson-plans/status/${result.job_id}`);
    } catch (error: unknown) {
      const message = getErrorMessage(error, 'Failed to upload lesson plan');
      console.error('Upload error:', error);
      Swal.fire({
        icon: 'error',
        title: 'Upload Failed',
        text: message,
      });
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="lesson-plan-upload-page">
      <div className="page-header">
        <h1>AI Lesson Plan Processor</h1>
        <p>Upload a lesson plan to automatically generate course content with practice quizzes</p>
      </div>

      <div className="upload-section">
        <div className="info-cards">
          <div className="info-card">
            <h3>📄 Supported Formats</h3>
            <p>PDF, DOCX, TXT</p>
          </div>
          <div className="info-card">
            <h3>📏 Max Size</h3>
            <p>50 MB</p>
          </div>
          <div className="info-card">
            <h3>⏱️ Processing Time</h3>
            <p>2-5 minutes</p>
          </div>
          <div className="info-card">
            <h3>✨ Output</h3>
            <p>Weeks + Modules + Practice Quizzes</p>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="upload-form">
          <div className="form-group">
            <label htmlFor="course">Select Course</label>
            <select
              id="course"
              value={courseId}
              onChange={(e: ChangeEvent<HTMLSelectElement>) => setCourseId(e.target.value)}
              required
              className="form-control"
              disabled={loadingCourses}
            >
              <option value="">-- Select a Course --</option>
              {loadingCourses ? (
                <option disabled>Loading courses...</option>
              ) : (
                courses.map((course) => (
                  <option key={course.id} value={course.id}>
                    {course.course_name} (Level {course.course_level}, Sem {course.semester})
                  </option>
                ))
              )}
            </select>
          </div>

          <div
            className="drop-zone"
            onDrop={handleDrop}
            onDragOver={handleDragOver}
            onClick={() => document.getElementById('file-input')?.click()}
          >
            <div className="drop-zone-content">
              <svg className="upload-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
                />
              </svg>
              <p>
                {selectedFile
                  ? selectedFile.name
                  : 'Drag and drop your lesson plan here, or click to browse'}
              </p>
              <p className="drop-zone-hint">Supports PDF, DOCX, TXT (max 50MB)</p>
            </div>
            <input
              type="file"
              id="file-input"
              onChange={handleFileChange}
              accept=".pdf,.docx,.txt"
              style={{ display: 'none' }}
            />
          </div>

          {selectedFile && (
            <div className="file-info">
              <span className="file-name">{selectedFile.name}</span>
              <span className="file-size">{(selectedFile.size / 1024 / 1024).toFixed(2)} MB</span>
            </div>
          )}

          <button
            type="submit"
            className="btn btn-primary"
            disabled={uploading || !selectedFile || !courseId}
          >
            {uploading ? (
              <>
                <span className="spinner"></span>
                Processing...
              </>
            ) : (
              <>
                <span>🚀</span>
                Upload and Process
              </>
            )}
          </button>
        </form>
      </div>

      <div className="how-it-works">
        <h2>How It Works</h2>
        <div className="steps">
          <div className="step">
            <div className="step-number">1</div>
            <h3>Upload</h3>
            <p>Upload your lesson plan document (PDF, DOCX, or TXT format)</p>
          </div>
          <div className="step">
            <div className="step-number">2</div>
            <h3>AI Processing</h3>
            <p>Our AI analyzes the content and extracts week-by-week structure</p>
          </div>
          <div className="step">
            <div className="step-number">3</div>
            <h3>Review</h3>
            <p>Review the generated weeks, modules, and practice quizzes</p>
          </div>
          <div className="step">
            <div className="step-number">4</div>
            <h3>Approve</h3>
            <p>Make any edits and approve to save to your course</p>
          </div>
        </div>
      </div>

      <style>{`
                .lesson-plan-upload-page {
                    max-width: 900px;
                    margin: 0 auto;
                    padding: 24px;
                }

                .page-header {
                    margin-bottom: 32px;
                }

                .page-header h1 {
                    font-size: 28px;
                    font-weight: 700;
                    margin-bottom: 8px;
                    color: var(--text-primary);
                }

                .page-header p {
                    color: var(--text-secondary);
                    font-size: 16px;
                }

                .info-cards {
                    display: grid;
                    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
                    gap: 16px;
                    margin-bottom: 32px;
                }

                .info-card {
                    background: linear-gradient(135deg, var(--sky-500) 0%, var(--sky-600) 100%);
                    color: #fff;
                    padding: 20px;
                    border-radius: 12px;
                    text-align: center;
                }

                .info-card h3 {
                    font-size: 14px;
                    font-weight: 500;
                    margin-bottom: 8px;
                    opacity: 0.9;
                }

                .info-card p {
                    font-size: 18px;
                    font-weight: 600;
                }

                .upload-form {
                    background: var(--bg-secondary);
                    padding: 32px;
                    border-radius: 16px;
                    box-shadow: var(--shadow-card);
                }

                .form-group {
                    margin-bottom: 24px;
                }

                .form-group label {
                    display: block;
                    font-weight: 600;
                    margin-bottom: 8px;
                    color: var(--text-primary);
                }

                .form-control {
                    width: 100%;
                    padding: 12px 16px;
                    border: 2px solid var(--bg-border);
                    border-radius: 8px;
                    font-size: 15px;
                    transition: border-color 0.2s;
                    background: var(--bg-primary);
                    color: var(--text-primary);
                }

                .form-control:focus {
                    outline: none;
                    border-color: var(--accent-primary);
                }

                .drop-zone {
                    border: 3px dashed var(--bg-border);
                    border-radius: 12px;
                    padding: 48px 24px;
                    text-align: center;
                    cursor: pointer;
                    transition: all 0.3s;
                    background: var(--bg-tertiary);
                }

                .drop-zone:hover {
                    border-color: var(--accent-primary);
                    background: var(--accent-primary-soft);
                }

                .upload-icon {
                    width: 64px;
                    height: 64px;
                    color: var(--accent-primary);
                    margin: 0 auto 16px;
                }

                .drop-zone-content p {
                    color: var(--text-secondary);
                    margin-bottom: 8px;
                }

                .drop-zone-hint {
                    font-size: 14px;
                    color: var(--text-muted) !important;
                }

                .file-info {
                    display: flex;
                    justify-content: space-between;
                    align-items: center;
                    padding: 12px 16px;
                    background: var(--bg-tertiary);
                    border-radius: 8px;
                    margin: 16px 0;
                }

                .file-name {
                    font-weight: 500;
                    color: var(--text-primary);
                }

                .file-size {
                    color: var(--text-secondary);
                    font-size: 14px;
                }

                .btn {
                    display: inline-flex;
                    align-items: center;
                    justify-content: center;
                    gap: 8px;
                    padding: 14px 28px;
                    font-size: 16px;
                    font-weight: 600;
                    border: none;
                    border-radius: 8px;
                    cursor: pointer;
                    transition: all 0.2s;
                }

                .btn-primary {
                    background: linear-gradient(135deg, var(--sky-500) 0%, var(--sky-600) 100%);
                    color: #fff;
                    width: 100%;
                }

                .btn-primary:hover:not(:disabled) {
                    transform: translateY(-2px);
                    box-shadow: 0 8px 20px var(--accent-primary-strong);
                }

                .btn:disabled {
                    opacity: 0.6;
                    cursor: not-allowed;
                }

                .spinner {
                    width: 20px;
                    height: 20px;
                    border: 3px solid rgba(255, 255, 255, 0.3);
                    border-top-color: #fff;
                    border-radius: 50%;
                    animation: spin 0.6s linear infinite;
                }

                @keyframes spin {
                    to { transform: rotate(360deg); }
                }

                .how-it-works {
                    margin-top: 48px;
                    padding-top: 32px;
                    border-top: 1px solid var(--bg-border);
                }

                .how-it-works h2 {
                    text-align: center;
                    font-size: 24px;
                    margin-bottom: 32px;
                    color: var(--text-primary);
                }

                .steps {
                    display: grid;
                    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
                    gap: 24px;
                }

                .step {
                    text-align: center;
                    padding: 24px;
                }

                .step-number {
                    width: 48px;
                    height: 48px;
                    background: linear-gradient(135deg, var(--sky-500) 0%, var(--sky-600) 100%);
                    color: #fff;
                    font-size: 20px;
                    font-weight: 700;
                    border-radius: 50%;
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    margin: 0 auto 16px;
                }

                .step h3 {
                    font-size: 16px;
                    margin-bottom: 8px;
                    color: var(--text-primary);
                }

                .step p {
                    font-size: 14px;
                    color: var(--text-secondary);
                }
            `}</style>
    </div>
  );
};

export default LessonPlanUpload;
