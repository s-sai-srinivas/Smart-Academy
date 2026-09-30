import { useState, useRef, type ChangeEvent, type FormEvent, type DragEvent } from 'react';
import api, { adminPrincipalAPI } from '../../services/api';
import { showError } from '../../utils/showAlert';
import { secureTokenStorage } from '../../services/secureStorage';
import './AdminStudentOnboarding.css';

interface UploadResult {
  total_rows: number;
  created: number;
  updated: number;
  failed: number;
  errors?: Array<{ row: number; regdno?: string; email?: string; error: string }>;
}

const AdminOnboarding = () => {
  const [activeTab, setActiveTab] = useState<'student' | 'faculty' | 'principal'>('student');
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState<boolean>(false);
  const [result, setResult] = useState<UploadResult | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const [principalForm, setPrincipalForm] = useState<{
    regdno: string;
    name: string;
    email: string;
    password: string;
    phone: string;
  }>({ regdno: '', name: '', email: '', password: '', phone: '' });
  const [principalLoading, setPrincipalLoading] = useState<boolean>(false);
  const [principalResult, setPrincipalResult] = useState<string | null>(null);

  const handleTabChange = (tab: 'student' | 'faculty' | 'principal') => {
    setActiveTab(tab);
    setFile(null);
    setResult(null);
    setPrincipalResult(null);
    if (fileInputRef.current) fileInputRef.current.value = '';
  };

  const handleCreatePrincipal = async (e: FormEvent) => {
    e.preventDefault();
    setPrincipalLoading(true);
    setPrincipalResult(null);
    try {
      await adminPrincipalAPI.create({
        regdno: principalForm.regdno,
        name: principalForm.name,
        email: principalForm.email,
        password: principalForm.password || undefined,
        phone_number: principalForm.phone || undefined,
      });
      setPrincipalResult(`Principal "${principalForm.name}" created successfully.`);
      setPrincipalForm({ regdno: '', name: '', email: '', password: '', phone: '' });
    } catch (err) {
      showError(
        (err as { response?: { data?: { error?: string } } }).response?.data?.error ||
          (err as Error).message ||
          'Failed to create principal'
      );
    } finally {
      setPrincipalLoading(false);
    }
  };

  const handleFileSelect = (e: ChangeEvent<HTMLInputElement>) => {
    const selectedFile = e.target.files?.[0];
    if (selectedFile && selectedFile.type === 'text/csv') {
      setFile(selectedFile);
      setResult(null);
    } else if (selectedFile) {
      showError('Please upload a CSV file');
    }
  };

  const handleDragOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
  };

  const handleDragEnter = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
  };

  const handleDragLeave = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
  };

  const handleDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();

    const droppedFile = e.dataTransfer.files[0];
    if (droppedFile && droppedFile.type === 'text/csv') {
      setFile(droppedFile);
      setResult(null);
    } else if (droppedFile) {
      showError('Please upload a CSV file');
    }
  };

  const handleDownloadTemplate = async () => {
    try {
      const token = secureTokenStorage.getToken();
      const authValue = token === 'httpOnly' ? 'httpOnly' : `Bearer ${token}`;
      const endpoint =
        activeTab === 'student' ? '/api/admin/students/template' : '/api/admin/faculty/template';

      const response = await fetch(endpoint, {
        headers: { Authorization: authValue },
      });

      if (!response.ok) {
        throw new Error('Failed to download template');
      }

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${activeTab}_onboarding_template.csv`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (err) {
      showError('Failed to download template: ' + (err as Error).message);
    }
  };

  const handleUpload = async () => {
    if (!file) {
      showError('Please select a file to upload');
      return;
    }

    setUploading(true);
    setResult(null);

    try {
      const formData = new FormData();
      formData.append('csv_file', file);

      const endpoint =
        activeTab === 'student' ? '/admin/students/bulk-upload' : '/admin/faculty/bulk-upload';

      const response = (await api.post(endpoint, formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      })) as UploadResult;

      setResult(response);
      setFile(null);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    } catch (err) {
      const errorMessage =
        (err as { response?: { data?: { error?: string } } }).response?.data?.error ||
        (err as Error).message ||
        'Upload failed';
      showError(errorMessage);
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="onboarding-container">
      <div className="onboarding-header">
        <div>
          <h1>Onboarding</h1>
          <p className="subtitle">Bulk upload users via CSV</p>
        </div>

        <div className="flex bg-background-tertiary p-1 rounded-lg">
          <button
            onClick={() => handleTabChange('student')}
            className={`px-4 py-2 text-sm font-medium rounded-md transition-all ${
              activeTab === 'student'
                ? 'bg-accent-secondary text-white shadow-sm'
                : 'text-text-secondary hover:text-text-primary'
            }`}
          >
            Student Onboarding
          </button>
          <button
            onClick={() => handleTabChange('faculty')}
            className={`px-4 py-2 text-sm font-medium rounded-md transition-all ${
              activeTab === 'faculty'
                ? 'bg-accent-secondary text-white shadow-sm'
                : 'text-text-secondary hover:text-text-primary'
            }`}
          >
            Faculty Onboarding
          </button>
          <button
            onClick={() => handleTabChange('principal')}
            className={`px-4 py-2 text-sm font-medium rounded-md transition-all ${
              activeTab === 'principal'
                ? 'bg-accent-secondary text-white shadow-sm'
                : 'text-text-secondary hover:text-text-primary'
            }`}
          >
            Create Principal
          </button>
        </div>
      </div>

      <div className="onboarding-content">
        {activeTab === 'principal' ? (
          <div className="action-section">
            <div className="upload-block">
              <h3>Create Principal Account</h3>
              <p style={{ color: 'var(--text-secondary)', marginBottom: '1.5rem' }}>
                Create a principal account for your college. Only one principal can exist per
                college.
              </p>

              {principalResult && (
                <div className="success-banner" style={{ marginBottom: '1rem' }}>
                  {principalResult}
                </div>
              )}

              <form
                onSubmit={handleCreatePrincipal}
                style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}
              >
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
                  <div>
                    <label
                      style={{
                        display: 'block',
                        marginBottom: '0.25rem',
                        fontSize: '0.875rem',
                        fontWeight: 500,
                        color: 'var(--text-primary)',
                      }}
                    >
                      Registration / Employee ID{' '}
                      <span style={{ color: 'var(--accent-primary)' }}>*</span>
                    </label>
                    <input
                      type="text"
                      value={principalForm.regdno}
                      required
                      onChange={(e: ChangeEvent<HTMLInputElement>) =>
                        setPrincipalForm({ ...principalForm, regdno: e.target.value })
                      }
                      placeholder="e.g., PRIN001"
                      style={{
                        width: '100%',
                        padding: '0.5rem 0.75rem',
                        borderRadius: '6px',
                        border: '1px solid var(--border-primary)',
                        background: 'var(--bg-primary)',
                        color: 'var(--text-primary)',
                        fontSize: '0.875rem',
                      }}
                    />
                  </div>
                  <div>
                    <label
                      style={{
                        display: 'block',
                        marginBottom: '0.25rem',
                        fontSize: '0.875rem',
                        fontWeight: 500,
                        color: 'var(--text-primary)',
                      }}
                    >
                      Full Name <span style={{ color: 'var(--accent-primary)' }}>*</span>
                    </label>
                    <input
                      type="text"
                      value={principalForm.name}
                      required
                      onChange={(e: ChangeEvent<HTMLInputElement>) =>
                        setPrincipalForm({ ...principalForm, name: e.target.value })
                      }
                      placeholder="e.g., Dr. John Smith"
                      style={{
                        width: '100%',
                        padding: '0.5rem 0.75rem',
                        borderRadius: '6px',
                        border: '1px solid var(--border-primary)',
                        background: 'var(--bg-primary)',
                        color: 'var(--text-primary)',
                        fontSize: '0.875rem',
                      }}
                    />
                  </div>
                  <div>
                    <label
                      style={{
                        display: 'block',
                        marginBottom: '0.25rem',
                        fontSize: '0.875rem',
                        fontWeight: 500,
                        color: 'var(--text-primary)',
                      }}
                    >
                      Email <span style={{ color: 'var(--accent-primary)' }}>*</span>
                    </label>
                    <input
                      type="email"
                      value={principalForm.email}
                      required
                      onChange={(e: ChangeEvent<HTMLInputElement>) =>
                        setPrincipalForm({ ...principalForm, email: e.target.value })
                      }
                      placeholder="e.g., principal@college.edu"
                      style={{
                        width: '100%',
                        padding: '0.5rem 0.75rem',
                        borderRadius: '6px',
                        border: '1px solid var(--border-primary)',
                        background: 'var(--bg-primary)',
                        color: 'var(--text-primary)',
                        fontSize: '0.875rem',
                      }}
                    />
                  </div>
                  <div>
                    <label
                      style={{
                        display: 'block',
                        marginBottom: '0.25rem',
                        fontSize: '0.875rem',
                        fontWeight: 500,
                        color: 'var(--text-primary)',
                      }}
                    >
                      Password
                    </label>
                    <input
                      type="password"
                      value={principalForm.password}
                      onChange={(e: ChangeEvent<HTMLInputElement>) =>
                        setPrincipalForm({ ...principalForm, password: e.target.value })
                      }
                      placeholder="Defaults to Reg/Emp ID"
                      style={{
                        width: '100%',
                        padding: '0.5rem 0.75rem',
                        borderRadius: '6px',
                        border: '1px solid var(--border-primary)',
                        background: 'var(--bg-primary)',
                        color: 'var(--text-primary)',
                        fontSize: '0.875rem',
                      }}
                    />
                  </div>
                  <div style={{ gridColumn: '1 / -1' }}>
                    <label
                      style={{
                        display: 'block',
                        marginBottom: '0.25rem',
                        fontSize: '0.875rem',
                        fontWeight: 500,
                        color: 'var(--text-primary)',
                      }}
                    >
                      Phone Number
                    </label>
                    <input
                      type="text"
                      value={principalForm.phone}
                      onChange={(e: ChangeEvent<HTMLInputElement>) =>
                        setPrincipalForm({ ...principalForm, phone: e.target.value })
                      }
                      placeholder="Optional"
                      style={{
                        width: '50%',
                        padding: '0.5rem 0.75rem',
                        borderRadius: '6px',
                        border: '1px solid var(--border-primary)',
                        background: 'var(--bg-primary)',
                        color: 'var(--text-primary)',
                        fontSize: '0.875rem',
                      }}
                    />
                  </div>
                </div>
                <div className="upload-actions">
                  <button
                    type="submit"
                    disabled={
                      principalLoading ||
                      !principalForm.regdno ||
                      !principalForm.name ||
                      !principalForm.email
                    }
                    className="btn btn-primary"
                  >
                    {principalLoading ? 'Creating...' : 'Create Principal'}
                  </button>
                </div>
              </form>
            </div>

            <div className="divider large"></div>

            <div className="instructions-container">
              <h3>Notes</h3>
              <ul
                style={{
                  color: 'var(--text-secondary)',
                  lineHeight: '1.8',
                  paddingLeft: '1.25rem',
                }}
              >
                <li>
                  Only <strong>one principal</strong> can exist per college.
                </li>
                <li>
                  If no password is provided, the <strong>Reg/Emp ID</strong> is used as the default
                  password.
                </li>
                <li>
                  The principal can manage <strong>HOD assignments</strong> for all departments.
                </li>
                <li>
                  The principal&apos;s college is automatically set to your (admin&apos;s) college.
                </li>
              </ul>
            </div>
          </div>
        ) : (
          <>
            <div className="action-section">
              <div className="flat-row template-row">
                <div className="row-info">
                  <h3>
                    Step 1: Download {activeTab === 'student' ? 'Student' : 'Faculty'} Template
                  </h3>
                  <p>Get the sample CSV to see the required format.</p>
                </div>
                <button onClick={handleDownloadTemplate} className="btn btn-outline">
                  <span className="icon">↓</span>
                  Download Template
                </button>
              </div>

              <div className="divider"></div>

              <div className="upload-block">
                <h3>Step 2: Upload CSV</h3>
                <div
                  className={`upload-area-minimal ${file ? 'has-file' : ''}`}
                  onDragOver={handleDragOver}
                  onDragEnter={handleDragEnter}
                  onDragLeave={handleDragLeave}
                  onDrop={handleDrop}
                  onClick={() => fileInputRef.current?.click()}
                >
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept=".csv"
                    onChange={handleFileSelect}
                    style={{ display: 'none' }}
                  />
                  {file ? (
                    <div className="file-selected">
                      <span className="file-icon">📄</span>
                      <span className="file-name">{file.name}</span>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          setFile(null);
                          setResult(null);
                        }}
                        className="btn-remove"
                      >
                        Remove
                      </button>
                    </div>
                  ) : (
                    <div className="upload-placeholder">
                      <span className="upload-icon-large">☁️</span>
                      <p>
                        Drag & drop CSV file here or{' '}
                        <span className="link-text">click to browse</span>
                      </p>
                      <p className="format-hint">Supported format: .csv</p>
                    </div>
                  )}
                </div>
                <div className="upload-actions">
                  <button
                    onClick={handleUpload}
                    disabled={!file || uploading}
                    className="btn btn-primary"
                  >
                    {uploading
                      ? 'Processing...'
                      : `Upload ${activeTab === 'student' ? 'Students' : 'Faculty'}`}
                  </button>
                </div>
              </div>
            </div>

            {result && (
              <div className="results-container">
                <h3>Upload Results</h3>
                <div className="results-summary">
                  <div className="result-item">
                    <span className="label">Total Processed</span>
                    <span className="value">{result.total_rows}</span>
                  </div>
                  <div className="result-item success">
                    <span className="label">Created</span>
                    <span className="value">{result.created}</span>
                  </div>
                  <div className="result-item info">
                    <span className="label">Updated</span>
                    <span className="value">{result.updated}</span>
                  </div>
                  <div className="result-item error">
                    <span className="label">Failed</span>
                    <span className="value">{result.failed}</span>
                  </div>
                </div>
                {result.errors && result.errors.length > 0 && (
                  <div className="errors-container">
                    <h4>Failed Rows ({result.errors.length})</h4>
                    <table className="minimal-table">
                      <thead>
                        <tr>
                          <th>Row</th>
                          <th>{activeTab === 'student' ? 'Regd No' : 'Emp ID'}</th>
                          <th>Error</th>
                        </tr>
                      </thead>
                      <tbody>
                        {result.errors.map((err, idx) => (
                          <tr key={idx}>
                            <td>{err.row}</td>
                            <td>
                              {activeTab === 'student' ? err.regdno : err.regdno || err.email}
                            </td>
                            <td className="error-text">{err.error}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
                {result.failed === 0 && (
                  <div className="success-banner">
                    All {activeTab === 'student' ? 'students' : 'faculty'} processed successfully.
                  </div>
                )}
              </div>
            )}

            <div className="divider large"></div>

            <div className="instructions-container">
              <h3>Branch Code Reference</h3>
              <p>
                Use any of the following values in the <code>branch</code> column:
              </p>
              <table className="minimal-table instructions-table">
                <thead>
                  <tr>
                    <th style={{ width: '30%' }}>Branch</th>
                    <th>Accepted Values</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Computer Science & Engineering</td>
                    <td>
                      <code>CSE</code>, <code>Computer Science</code>,{' '}
                      <code>Computer Science & Eng</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Electronics & Communication</td>
                    <td>
                      <code>ECE</code>, <code>Electronics</code>, <code>Electronics & Comm</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Electrical & Electronics</td>
                    <td>
                      <code>EEE</code>, <code>Electrical</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Mechanical Engineering</td>
                    <td>
                      <code>ME</code>, <code>Mechanical</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Civil Engineering</td>
                    <td>
                      <code>CE</code>, <code>Civil</code>, <code>CIVIL</code>,{' '}
                      <code>Civil Engineering</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Information Technology</td>
                    <td>
                      <code>IT</code>, <code>Information Technology</code>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>

            <div className="divider large"></div>

            <div className="instructions-container">
              <h3>CSV Format Instructions</h3>
              {activeTab === 'student' ? (
                <table className="minimal-table instructions-table">
                  <thead>
                    <tr>
                      <th style={{ width: '20%' }}>Field</th>
                      <th style={{ width: '15%' }}>Required</th>
                      <th>Description</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr>
                      <td>
                        <code>regdno</code>
                      </td>
                      <td>
                        <span className="badge required">Yes</span>
                      </td>
                      <td>Unique registration number for the student.</td>
                    </tr>
                    <tr>
                      <td>
                        <code>name</code>
                      </td>
                      <td>
                        <span className="badge required">Yes</span>
                      </td>
                      <td>Student&apos;s full name.</td>
                    </tr>
                    <tr>
                      <td>
                        <code>email</code>
                      </td>
                      <td>
                        <span className="badge required">Yes</span>
                      </td>
                      <td>Student&apos;s email address (used for login).</td>
                    </tr>
                    <tr>
                      <td>
                        <code>phone_number</code>
                      </td>
                      <td>
                        <span className="badge optional">No</span>
                      </td>
                      <td>Contact number.</td>
                    </tr>
                    <tr>
                      <td>
                        <code>program</code>
                      </td>
                      <td>
                        <span className="badge required">Yes</span>
                      </td>
                      <td>Program name (e.g., B.Tech, M.Tech, MBA).</td>
                    </tr>
                    <tr>
                      <td>
                        <code>branch</code>
                      </td>
                      <td>
                        <span className="badge required">Yes</span>
                      </td>
                      <td>Branch name (e.g., CSE, ECE).</td>
                    </tr>
                    <tr>
                      <td>
                        <code>cohort_year</code>
                      </td>
                      <td>
                        <span className="badge required">Yes</span>
                      </td>
                      <td>Batch year (e.g., 2024). This is permanent.</td>
                    </tr>
                    <tr>
                      <td>
                        <code>admission_year</code>
                      </td>
                      <td>
                        <span className="badge optional">No</span>
                      </td>
                      <td>Admission year. Defaults to cohort_year if empty.</td>
                    </tr>
                    <tr>
                      <td>
                        <code>section</code>
                      </td>
                      <td>
                        <span className="badge optional">No</span>
                      </td>
                      <td>Section (A, B, C, etc.). Defaults to A.</td>
                    </tr>
                    <tr>
                      <td>
                        <code>role</code>
                      </td>
                      <td>
                        <span className="badge optional">No</span>
                      </td>
                      <td>
                        User role. Defaults to <code>student</code>.
                      </td>
                    </tr>
                  </tbody>
                </table>
              ) : (
                <table className="minimal-table instructions-table">
                  <thead>
                    <tr>
                      <th style={{ width: '20%' }}>Field</th>
                      <th style={{ width: '15%' }}>Required</th>
                      <th>Description</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr>
                      <td>
                        <code>regdno</code>
                      </td>
                      <td>
                        <span className="badge required">Yes</span>
                      </td>
                      <td>Employee ID (primary key for faculty).</td>
                    </tr>
                    <tr>
                      <td>
                        <code>name</code>
                      </td>
                      <td>
                        <span className="badge required">Yes</span>
                      </td>
                      <td>Faculty&apos;s full name.</td>
                    </tr>
                    <tr>
                      <td>
                        <code>email</code>
                      </td>
                      <td>
                        <span className="badge required">Yes</span>
                      </td>
                      <td>Faculty&apos;s email address (used for login).</td>
                    </tr>
                    <tr>
                      <td>
                        <code>phone_number</code>
                      </td>
                      <td>
                        <span className="badge optional">No</span>
                      </td>
                      <td>Contact number.</td>
                    </tr>
                    <tr>
                      <td>
                        <code>branch</code>
                      </td>
                      <td>
                        <span className="badge required">Yes</span>
                      </td>
                      <td>Branch name (e.g., CSE, ECE).</td>
                    </tr>
                  </tbody>
                </table>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
};

export default AdminOnboarding;
