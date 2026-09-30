import { useState, useEffect, useRef, ChangeEvent } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { lessonPlanAPI, contestGenerationAPI, contestsAPI } from '../../services/api';
import { getErrorMessage } from '../../utils/error';
import Swal from 'sweetalert2';

// ─── Interfaces ──────────────────────────────────────────────────────────────

interface Contest {
  id: number;
  title: string;
}

interface Problem {
  problem_id: number;
  title: string;
  difficulty: string;
  verification_status: string;
  test_cases_count?: number;
  topics_covered?: string[];
}

interface JobStatus {
  status: string;
  created_at?: string;
  error_message?: string;
  preview?: Preview;
}

interface Preview {
  total_weeks: number;
  weeks: WeekPreview[];
}

interface WeekPreview {
  week_number: number;
  week_name: string;
  module_count: number;
  modules: ModulePreview[];
}

interface ModulePreview {
  module_name: string;
  description: string;
  module_number: number;
  content_length: number;
}

interface NewContestData {
  title: string;
  description: string;
  start_time: string;
  end_time: string;
}

interface TestCaseDetail {
  category: string;
  is_sample_visible: boolean;
  input: string;
  expected_output: string;
}

interface ViewingProblem {
  title: string;
  description: string;
  input_format: string;
  output_format: string;
  constraints?: Record<string, unknown>;
  sample_input: string;
  sample_output: string;
  sample_explanation?: string;
  test_cases?: TestCaseDetail[];
  solution_approach?: string;
}

// ─── Main Component ──────────────────────────────────────────────────────────

const ReviewGeneratedContent = () => {
  const navigate = useNavigate();
  const location = useLocation();

  // Get job ID and type from URL params
  const queryParams = new URLSearchParams(location.search);
  const jobId = queryParams.get('jobId');
  const type = queryParams.get('type') || 'lesson-plan';

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [jobStatus, setJobStatus] = useState<JobStatus | null>(null);
  const [preview, setPreview] = useState<Preview | null>(null);
  // Track if user has made any edits to the preview
  const [hasEdits, setHasEdits] = useState(false);

  // Contest-specific state
  const [problems, setProblems] = useState<Problem[]>([]);
  const [selectedProblems, setSelectedProblems] = useState<number[]>([]);
  const [contests, setContests] = useState<Contest[]>([]);
  const [selectedContest, setSelectedContest] = useState<string>('');
  const [viewingProblem, setViewingProblem] = useState<ViewingProblem | null>(null);

  // Contest creation state (lifted from ContestReviewView for use in handleApproveContestProblems)
  const [createNewContest, setCreateNewContest] = useState(false);
  const [newContestData, setNewContestData] = useState<NewContestData>({
    title: '',
    description: '',
    start_time: '',
    end_time: '',
  });

  // Use a ref to track current status so the polling interval can read the
  // latest value without stale-closure issues
  const statusRef = useRef<string | null>(null);

  const fetchGeneratedProblems = async () => {
    if (!jobId) return;
    try {
      const data = (await contestGenerationAPI.getGeneratedProblems(jobId)) as {
        problems: Problem[];
      };
      setProblems(data.problems || []);
    } catch (error) {
      console.error('Error fetching problems:', error);
    }
  };

  const fetchJobStatus = async () => {
    if (!jobId) return;
    try {
      const apiClient = type === 'contest' ? contestGenerationAPI : lessonPlanAPI;
      const data = (await apiClient.getJobStatus(jobId)) as JobStatus;
      setJobStatus(data);
      statusRef.current = data.status;

      if (data.status === 'completed') {
        if (type === 'contest') {
          await fetchGeneratedProblems();
        } else if (data.preview) {
          setPreview(data.preview);
        }
      }
    } catch (error) {
      console.error('Error fetching job status:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!jobId) {
      navigate('/admin');
      return;
    }

    fetchJobStatus();

    if (type === 'contest') {
      contestsAPI
        .getAll()
        .then((d) => setContests(d as Contest[]))
        .catch(console.error);
    }

    // Poll every 5 s — use statusRef to avoid stale closure
    const interval = setInterval(() => {
      const s = statusRef.current;
      if (s === null || s === 'pending' || s === 'processing') {
        fetchJobStatus();
      }
    }, 5000);

    return () => clearInterval(interval);
  }, [jobId, type]); // eslint-disable-line react-hooks/exhaustive-deps

  const handleWeekEdit = (weekIndex: number, field: keyof WeekPreview, value: string) => {
    setHasEdits(true);

    // Update the preview with the edited value (preview is the source of truth)
    setPreview((prev) => {
      if (!prev) return prev;
      const updatedWeeks = [...prev.weeks];
      updatedWeeks[weekIndex] = {
        ...updatedWeeks[weekIndex],
        [field]: value,
      };
      return { ...prev, weeks: updatedWeeks };
    });
  };

  const handleModuleEdit = (
    weekIndex: number,
    moduleIndex: number,
    field: keyof ModulePreview,
    value: string
  ) => {
    setHasEdits(true);

    // Update the preview with the edited module value (preview is the source of truth)
    setPreview((prev) => {
      if (!prev) return prev;
      const updatedWeeks = [...prev.weeks];
      const updatedModules = [...updatedWeeks[weekIndex].modules];
      updatedModules[moduleIndex] = {
        ...updatedModules[moduleIndex],
        [field]: value,
      };
      updatedWeeks[weekIndex] = {
        ...updatedWeeks[weekIndex],
        modules: updatedModules,
      };
      return { ...prev, weeks: updatedWeeks };
    });
  };

  const handleRegenerate = async (_weekIndex: number) => {
    Swal.fire({
      icon: 'warning',
      title: 'Regenerate Week?',
      text: 'This will regenerate the week content using AI. Any edits you made will be lost.',
      showCancelButton: true,
      confirmButtonText: 'Yes, Regenerate',
      cancelButtonText: 'Cancel',
    }).then(() => {
      // In production, call API to regenerate specific week
      Swal.fire({
        icon: 'info',
        title: 'Coming Soon',
        text: 'Week regeneration will be implemented in a future update.',
      });
    });
  };

  const handleApprove = async () => {
    if (type === 'contest') {
      await handleApproveContestProblems();
    } else {
      await handleApproveLessonPlan();
    }
  };

  const handleApproveContestProblems = async () => {
    if (!jobId) return;
    if (selectedProblems.length === 0) {
      Swal.fire({
        icon: 'warning',
        title: 'No Problems Selected',
        text: 'Please select at least one problem to approve',
      });
      return;
    }

    const result = await Swal.fire({
      title: 'Approve Problems?',
      text: `You are about to approve ${selectedProblems.length} problem(s). This action cannot be undone.`,
      icon: 'question',
      showCancelButton: true,
      confirmButtonText: 'Yes, approve',
      cancelButtonText: 'Cancel',
    });

    if (!result.isConfirmed) return;

    setSaving(true);

    try {
      // Prepare contest data
      const contestData: Record<string, unknown> = {
        approved_problem_indices: selectedProblems,
      };

      if (createNewContest) {
        // Create new contest
        contestData.create_contest = true;
        contestData.contest_title =
          newContestData.title || `Contest - ${new Date().toLocaleDateString()}`;
        contestData.contest_description = newContestData.description || '';
        contestData.start_time = newContestData.start_time || '';
        contestData.end_time = newContestData.end_time || '';
      } else if (selectedContest) {
        // Add to existing contest
        contestData.contest_id = parseInt(selectedContest);
      }
      // else: save to problem bank only

      await contestGenerationAPI.approveProblems(jobId, contestData);

      Swal.fire({
        icon: 'success',
        title: 'Problems Approved',
        text: `${selectedProblems.length} problem(s) have been saved successfully`,
      });

      if (selectedContest || createNewContest) {
        // Navigate to the contest page
        const contestId = selectedContest || 'latest';
        navigate(`/admin/contests/${contestId}`);
      } else {
        navigate('/admin/contests');
      }
    } catch (error: unknown) {
      const message = getErrorMessage(error, 'Failed to save problems');
      console.error('Save error:', error);
      Swal.fire({
        icon: 'error',
        title: 'Save Failed',
        text: message,
      });
    } finally {
      setSaving(false);
    }
  };

  const handleApproveLessonPlan = async () => {
    if (!jobId) return;
    const result = await Swal.fire({
      title: 'Approve Generated Content?',
      html: `
                <p>This will create:</p>
                <ul style="text-align: left;">
                    <li>${preview?.total_weeks || 0} weeks</li>
                    <li>Practice quizzes for each week</li>
                </ul>
                ${hasEdits ? '<p style="color: var(--sky-500); font-size: 14px; margin-top: 8px;">✓ Your edits will be saved</p>' : ''}
                <p style="color: var(--text-secondary); font-size: 14px; margin-top: 16px;">
                    You can still edit the content later from the course management page.
                </p>
            `,
      icon: 'question',
      showCancelButton: true,
      confirmButtonText: 'Yes, Approve & Save',
      cancelButtonText: 'Cancel',
      confirmButtonColor: 'var(--sky-500)',
      cancelButtonColor: 'var(--text-muted)',
    });

    if (!result.isConfirmed) {
      return;
    }

    setSaving(true);

    try {
      // Send the full preview (which includes all user edits) as modifications
      // Preview is the single source of truth for edited content
      await lessonPlanAPI.approveGeneration(jobId, preview);

      Swal.fire({
        icon: 'success',
        title: 'Content Generated!',
        text: 'The course content has been saved successfully.',
        timer: 2000,
      });

      navigate('/admin/courses');
    } catch (error: unknown) {
      const message = getErrorMessage(error, 'Failed to save generated content');
      console.error('Save error:', error);
      Swal.fire({
        icon: 'error',
        title: 'Save Failed',
        text: message,
      });
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center h-64 gap-4">
        <svg className="animate-spin w-8 h-8 text-accent-primary" fill="none" viewBox="0 0 24 24">
          <circle
            className="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            strokeWidth="4"
          />
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
          />
        </svg>
        <p className="text-text-secondary">Loading generated content…</p>
      </div>
    );
  }

  if (jobStatus?.status === 'pending' || jobStatus?.status === 'processing') {
    return (
      <div className="flex flex-col items-center justify-center h-64 gap-4 text-center">
        <svg className="animate-spin w-10 h-10 text-accent-primary" fill="none" viewBox="0 0 24 24">
          <circle
            className="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            strokeWidth="4"
          />
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
          />
        </svg>
        <div>
          <h2 className="text-xl font-semibold text-text-primary mb-1">
            {type === 'contest' ? 'Generating Contest Problems…' : 'Generating Course Structure…'}
          </h2>
          <p className="text-text-secondary text-sm max-w-md">
            {type === 'contest'
              ? 'The AI is writing and verifying contest problems. This usually takes 3–5 minutes.'
              : 'The AI is analysing the document and building the course structure…'}
          </p>
        </div>
        <span className="badge badge-warning capitalize">{jobStatus?.status}</span>
        <p className="text-text-muted text-xs">This page auto-refreshes every 5 seconds</p>
      </div>
    );
  }

  if (jobStatus?.status === 'failed') {
    return (
      <div className="flex flex-col items-center justify-center h-64 gap-4 text-center">
        <div className="w-12 h-12 rounded-full bg-accent-danger/10 flex items-center justify-center">
          <svg
            className="w-6 h-6 text-accent-danger"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        </div>
        <div>
          <h2 className="text-xl font-semibold text-text-primary mb-1">Generation Failed</h2>
          <p className="text-text-secondary text-sm max-w-md">
            {jobStatus?.error_message || 'An unexpected error occurred.'}
          </p>
        </div>
        <button
          className="btn btn-primary"
          onClick={() =>
            navigate(type === 'contest' ? '/admin/contests/generate' : '/admin/lesson-plans/upload')
          }
        >
          Try Again
        </button>
      </div>
    );
  }

  // Contest-specific rendering
  if (type === 'contest') {
    return (
      <ContestReviewView
        jobId={jobId}
        jobStatus={jobStatus}
        problems={problems}
        selectedProblems={selectedProblems}
        setSelectedProblems={setSelectedProblems}
        contests={contests}
        selectedContest={selectedContest}
        setSelectedContest={setSelectedContest}
        viewingProblem={viewingProblem}
        setViewingProblem={setViewingProblem}
        saving={saving}
        handleApprove={handleApprove}
        navigate={navigate}
        createNewContest={createNewContest}
        setCreateNewContest={setCreateNewContest}
        newContestData={newContestData}
        setNewContestData={setNewContestData}
      />
    );
  }

  // Lesson plan rendering (existing)
  return (
    <div className="review-generated-content">
      <div className="page-header">
        <h1>Review Generated Content</h1>
        <p>Review and edit the AI-generated course structure before saving</p>
      </div>

      <div className="action-bar">
        <button
          className="btn btn-secondary"
          onClick={() => navigate('/admin/lesson-plans/upload')}
        >
          ← Back
        </button>
        <div className="action-buttons">
          <button
            className="btn btn-outline"
            onClick={() => {
              /* Expand all */
            }}
          >
            Expand All
          </button>
          <button className="btn btn-primary" onClick={handleApprove} disabled={saving}>
            {saving ? (
              <>
                <span className="spinner"></span>
                Saving...
              </>
            ) : (
              <>
                <span>✅</span>
                Approve & Save
              </>
            )}
          </button>
        </div>
      </div>

      <div className="content-preview">
        <div className="summary-card">
          <h3>Summary</h3>
          <div className="summary-stats">
            <div className="stat">
              <span className="stat-value">{preview?.total_weeks || 0}</span>
              <span className="stat-label">Weeks</span>
            </div>
            <div className="stat">
              <span className="stat-value">
                {preview?.weeks?.reduce((acc, week) => acc + (week.module_count || 0), 0) || 0}
              </span>
              <span className="stat-label">Modules</span>
            </div>
            <div className="stat">
              <span className="stat-value">{preview?.total_weeks || 0}</span>
              <span className="stat-label">Practice Quizzes</span>
            </div>
          </div>
        </div>

        <div className="weeks-container">
          {preview?.weeks?.map((week, weekIndex) => (
            <WeekAccordion
              key={weekIndex}
              week={week}
              weekIndex={weekIndex}
              onEdit={handleWeekEdit}
              onModuleEdit={handleModuleEdit}
              onRegenerate={handleRegenerate}
            />
          ))}
        </div>
      </div>
    </div>
  );
};

// ─── Week Accordion Props ────────────────────────────────────────────────────

interface WeekAccordionProps {
  week: WeekPreview;
  weekIndex: number;
  onEdit: (weekIndex: number, field: keyof WeekPreview, value: string) => void;
  onModuleEdit: (
    weekIndex: number,
    moduleIndex: number,
    field: keyof ModulePreview,
    value: string
  ) => void;
  onRegenerate: (weekIndex: number) => void;
}

// Week Accordion Component
const WeekAccordion = ({
  week,
  weekIndex,
  onEdit: _onEdit,
  onModuleEdit,
  onRegenerate,
}: WeekAccordionProps) => {
  const [expanded, setExpanded] = useState(true);

  return (
    <div className="week-accordion">
      <div className="week-header" onClick={() => setExpanded(!expanded)}>
        <div className="week-title">
          <span className="expand-icon">{expanded ? '▼' : '▶'}</span>
          <span className="week-number">Week {week.week_number}</span>
          <span className="week-name">{week.week_name}</span>
        </div>
        <div className="week-meta">
          <span className="module-count">{week.module_count} modules</span>
          <span className="quiz-badge">📝 Quiz Included</span>
        </div>
        <div className="week-actions">
          <button
            className="btn btn-sm btn-outline"
            onClick={(e) => {
              e.stopPropagation();
              onRegenerate(weekIndex);
            }}
          >
            🔄 Regenerate
          </button>
        </div>
      </div>

      {expanded && (
        <div className="week-content">
          <div className="modules-list">
            <h4>Modules</h4>
            {week.modules?.map((module, moduleIndex) => (
              <ModuleCard
                key={moduleIndex}
                module={module}
                weekIndex={weekIndex}
                moduleIndex={moduleIndex}
                onEdit={onModuleEdit}
              />
            ))}
          </div>

          <div className="practice-quiz-preview">
            <h4>
              <span>📝</span> Practice Quiz Preview
            </h4>
            <p className="quiz-hint">
              A practice quiz with 5-10 questions will be generated for this week. Questions will
              cover all modules and include explanations.
            </p>
          </div>
        </div>
      )}
    </div>
  );
};

// ─── Module Card Props ───────────────────────────────────────────────────────

interface ModuleCardProps {
  module: ModulePreview;
  weekIndex: number;
  moduleIndex: number;
  onEdit: (
    weekIndex: number,
    moduleIndex: number,
    field: keyof ModulePreview,
    value: string
  ) => void;
}

// Module Card Component
const ModuleCard = ({ module, weekIndex, moduleIndex, onEdit }: ModuleCardProps) => {
  const [editing, setEditing] = useState(false);
  const [editedContent, setEditedContent] = useState({
    name: module.module_name,
    description: module.description,
  });

  const handleSave = () => {
    onEdit(weekIndex, moduleIndex, 'module_name', editedContent.name);
    onEdit(weekIndex, moduleIndex, 'description', editedContent.description);
    setEditing(false);
  };

  return (
    <div className="module-card">
      <div className="module-header">
        <div className="module-number">Module {module.module_number}</div>
        <div className="module-actions">
          <button className="btn btn-sm btn-outline" onClick={() => setEditing(!editing)}>
            {editing ? 'Cancel' : '✏️ Edit'}
          </button>
        </div>
      </div>

      {editing ? (
        <div className="module-edit-form">
          <div className="form-group">
            <label>Module Name</label>
            <input
              type="text"
              value={editedContent.name}
              onChange={(e: ChangeEvent<HTMLInputElement>) =>
                setEditedContent((prev) => ({ ...prev, name: e.target.value }))
              }
              className="form-control"
            />
          </div>
          <div className="form-group">
            <label>Description</label>
            <textarea
              value={editedContent.description}
              onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                setEditedContent((prev) => ({ ...prev, description: e.target.value }))
              }
              className="form-control"
              rows={3}
            />
          </div>
          <button className="btn btn-primary btn-sm" onClick={handleSave}>
            Save Changes
          </button>
        </div>
      ) : (
        <div className="module-body">
          <h5 className="module-name">{module.module_name}</h5>
          <p className="module-description">{module.description}</p>
          <div className="module-meta">
            <span className="content-length">
              📄 ~{Math.ceil(module.content_length / 500)} min read
            </span>
          </div>
        </div>
      )}
    </div>
  );
};

// ─── Contest Review View Props ───────────────────────────────────────────────

interface ContestReviewViewProps {
  jobId: string | null;
  jobStatus: JobStatus | null;
  problems: Problem[];
  selectedProblems: number[];
  setSelectedProblems: React.Dispatch<React.SetStateAction<number[]>>;
  contests: Contest[];
  selectedContest: string;
  setSelectedContest: React.Dispatch<React.SetStateAction<string>>;
  viewingProblem: ViewingProblem | null;
  setViewingProblem: React.Dispatch<React.SetStateAction<ViewingProblem | null>>;
  saving: boolean;
  handleApprove: () => void;
  navigate: ReturnType<typeof useNavigate>;
  createNewContest: boolean;
  setCreateNewContest: React.Dispatch<React.SetStateAction<boolean>>;
  newContestData: NewContestData;
  setNewContestData: React.Dispatch<React.SetStateAction<NewContestData>>;
}

// Contest Review View Component
const ContestReviewView = ({
  jobId,
  jobStatus,
  problems,
  selectedProblems,
  setSelectedProblems,
  contests,
  selectedContest,
  setSelectedContest,
  viewingProblem,
  setViewingProblem,
  saving,
  handleApprove,
  navigate,
  createNewContest,
  setCreateNewContest,
  newContestData,
  setNewContestData,
}: ContestReviewViewProps) => {
  const handleProblemSelect = (problemId: number) => {
    setSelectedProblems((prev) =>
      prev.includes(problemId) ? prev.filter((id) => id !== problemId) : [...prev, problemId]
    );
  };

  const handleSelectAll = () => {
    if (selectedProblems.length === problems.length) {
      setSelectedProblems([]);
    } else {
      setSelectedProblems(problems.map((p) => p.problem_id));
    }
  };

  const handleViewProblem = async (problemIndex: number) => {
    if (!jobId) return;
    try {
      const problem = (await contestGenerationAPI.getProblemDetail(
        jobId,
        problemIndex
      )) as ViewingProblem;
      setViewingProblem(problem);
    } catch {
      Swal.fire({ icon: 'error', title: 'Error', text: 'Failed to load problem details' });
    }
  };

  const handleRegenerate = async (problemIndex: number) => {
    if (!jobId) return;
    const result = await Swal.fire({
      title: 'Regenerate Problem?',
      text: 'Request the AI to generate a new version of this problem.',
      icon: 'question',
      showCancelButton: true,
      confirmButtonText: 'Yes, regenerate',
      cancelButtonText: 'Cancel',
    });
    if (!result.isConfirmed) return;
    try {
      await contestGenerationAPI.regenerateProblem(jobId, problemIndex, 'Quality improvement');
      Swal.fire({
        icon: 'success',
        title: 'Regeneration Requested',
        text: 'Problem regeneration has been queued.',
      });
    } catch (error: unknown) {
      Swal.fire({
        icon: 'error',
        title: 'Failed',
        text: getErrorMessage(error, 'Failed to regenerate problem'),
      });
    }
  };

  const difficultyBadge = (d: string) => {
    const map: Record<string, string> = {
      easy: 'badge badge-success',
      medium: 'badge badge-warning',
      hard: 'badge badge-danger',
    };
    return map[d] || 'badge badge-neutral';
  };

  const verificationBadge = (s: string) => {
    const map: Record<string, string> = {
      passed: 'badge badge-success',
      failed: 'badge badge-danger',
      pending: 'badge badge-neutral',
    };
    return map[s] || 'badge badge-neutral';
  };

  const allSelected = problems.length > 0 && selectedProblems.length === problems.length;

  return (
    <div className="space-y-6 pb-24">
      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold text-text-primary mb-1">Review Generated Problems</h1>
          <p className="text-text-secondary text-sm">
            Select the problems you want to keep. Approved problems are saved to the problem bank.
          </p>
        </div>
        <button
          onClick={() => navigate('/admin/contests/generate')}
          className="btn btn-secondary shrink-0"
        >
          ← Generate More
        </button>
      </div>

      {/* Job info bar */}
      <div className="card flex flex-wrap items-center gap-4 text-sm py-3">
        <div className="flex items-center gap-2 text-text-secondary">
          <span className="text-text-muted">Job ID:</span>
          <span className="font-mono text-text-primary">{jobId}</span>
        </div>
        <div className="flex items-center gap-2 text-text-secondary">
          <span className="text-text-muted">Status:</span>
          <span className="badge badge-success capitalize">{jobStatus?.status}</span>
        </div>
        <div className="flex items-center gap-2 text-text-secondary">
          <span className="text-text-muted">Generated:</span>
          <span className="text-text-primary">
            {jobStatus?.created_at ? new Date(jobStatus.created_at).toLocaleString() : '—'}
          </span>
        </div>
        <div className="ml-auto flex items-center gap-2 font-medium text-text-primary">
          <span className="text-accent-primary">{problems.length}</span>
          <span className="text-text-muted">problems ready</span>
        </div>
      </div>

      {/* Associate with contest (optional) */}
      <div className="card space-y-2">
        <div className="flex items-center gap-2 mb-2">
          <input
            type="checkbox"
            id="create-contest-toggle"
            checked={createNewContest}
            onChange={(e: ChangeEvent<HTMLInputElement>) => setCreateNewContest(e.target.checked)}
            className="w-4 h-4 rounded border-background-border"
          />
          <label htmlFor="create-contest-toggle" className="form-label mb-0 cursor-pointer">
            Create New Contest
          </label>
        </div>

        {createNewContest ? (
          <div className="space-y-3 p-3 bg-background-tertiary rounded-lg">
            <div>
              <label className="form-label">Contest Title</label>
              <input
                type="text"
                value={newContestData.title}
                onChange={(e: ChangeEvent<HTMLInputElement>) =>
                  setNewContestData({ ...newContestData, title: e.target.value })
                }
                className="input"
                placeholder="e.g., Weekly Contest 1"
              />
            </div>
            <div>
              <label className="form-label">Description</label>
              <textarea
                value={newContestData.description}
                onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                  setNewContestData({ ...newContestData, description: e.target.value })
                }
                className="textarea"
                placeholder="Contest description..."
                rows={2}
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="form-label">Start Time</label>
                <input
                  type="datetime-local"
                  value={newContestData.start_time}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    setNewContestData({ ...newContestData, start_time: e.target.value })
                  }
                  className="input"
                />
              </div>
              <div>
                <label className="form-label">End Time</label>
                <input
                  type="datetime-local"
                  value={newContestData.end_time}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    setNewContestData({ ...newContestData, end_time: e.target.value })
                  }
                  className="input"
                />
              </div>
            </div>
          </div>
        ) : (
          <>
            <label className="form-label">Add to Existing Contest</label>
            <select
              value={selectedContest}
              onChange={(e: ChangeEvent<HTMLSelectElement>) => setSelectedContest(e.target.value)}
              className="select"
            >
              <option value="">Save to problem bank only (no contest)</option>
              {contests.map((contest) => (
                <option key={contest.id} value={contest.id}>
                  {contest.title}
                </option>
              ))}
            </select>
          </>
        )}
        <p className="text-text-muted text-xs">
          You can always add saved problems to a contest later from the Contests page.
        </p>
      </div>

      {/* Select-all row */}
      <div className="flex items-center justify-between">
        <label className="flex items-center gap-3 cursor-pointer select-none">
          <span
            className={`w-5 h-5 rounded border flex items-center justify-center shrink-0 cursor-pointer transition-colors
                            ${allSelected ? 'bg-accent-primary border-accent-primary' : 'border-background-border bg-background-tertiary'}`}
            onClick={handleSelectAll}
          >
            {allSelected && (
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
          <span className="text-sm text-text-secondary">
            <span className="text-text-primary font-medium">{selectedProblems.length}</span> of{' '}
            <span className="text-text-primary font-medium">{problems.length}</span> selected
          </span>
        </label>
        {selectedProblems.length > 0 && (
          <button
            onClick={() => setSelectedProblems([])}
            className="text-xs text-text-muted hover:text-text-secondary transition-colors"
          >
            Clear selection
          </button>
        )}
      </div>

      {/* Problem cards */}
      {problems.length === 0 ? (
        <div className="card flex flex-col items-center justify-center py-16">
          <p className="text-text-secondary">No problems generated yet.</p>
        </div>
      ) : (
        <div className="grid gap-4">
          {problems.map((problem, index) => {
            const isSelected = selectedProblems.includes(problem.problem_id);
            return (
              <div
                key={index}
                className={`card transition-all border-2 ${isSelected ? 'border-accent-primary' : 'border-background-border'}`}
              >
                <div className="flex items-start gap-4">
                  {/* Checkbox */}
                  <span
                    className={`w-5 h-5 rounded border flex items-center justify-center shrink-0 cursor-pointer mt-1 transition-colors
                                            ${isSelected ? 'bg-accent-primary border-accent-primary' : 'border-background-border bg-background-tertiary'}`}
                    onClick={() => handleProblemSelect(problem.problem_id)}
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

                  {/* Content */}
                  <div className="flex-1 min-w-0">
                    <div className="flex flex-wrap items-start justify-between gap-2 mb-3">
                      <h3 className="text-base font-semibold text-text-primary">{problem.title}</h3>
                      <div className="flex items-center gap-2 shrink-0">
                        <span className={difficultyBadge(problem.difficulty)}>
                          {problem.difficulty}
                        </span>
                        <span className={verificationBadge(problem.verification_status)}>
                          {problem.verification_status || 'pending'}
                        </span>
                      </div>
                    </div>

                    <div className="grid grid-cols-2 sm:grid-cols-3 gap-3 mb-4">
                      <div>
                        <p className="text-xs text-text-muted mb-1">Test Cases</p>
                        <p className="text-sm font-medium text-text-primary">
                          {problem.test_cases_count ?? 0}
                        </p>
                      </div>
                      <div className="col-span-2 sm:col-span-2">
                        <p className="text-xs text-text-muted mb-1">Topics</p>
                        <p className="text-sm text-text-primary">
                          {problem.topics_covered?.join(', ') || '—'}
                        </p>
                      </div>
                    </div>

                    <div className="flex gap-2">
                      <button
                        onClick={() => handleViewProblem(index)}
                        className="btn btn-secondary text-xs px-3 py-1.5"
                      >
                        View Details
                      </button>
                      <button
                        onClick={() => handleRegenerate(index)}
                        className="btn btn-ghost text-xs px-3 py-1.5"
                      >
                        🔄 Regenerate
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Sticky approve bar */}
      <div className="fixed bottom-0 left-0 right-0 bg-background-secondary border-t border-background-border shadow-elevated z-40">
        <div className="max-w-7xl mx-auto px-4 py-3 flex items-center justify-between gap-4">
          <p className="text-sm text-text-secondary hidden sm:block">
            {selectedProblems.length === 0
              ? 'Select problems to approve'
              : `${selectedProblems.length} problem${selectedProblems.length !== 1 ? 's' : ''} selected`}
          </p>
          <div className="flex items-center gap-3 ml-auto">
            <button onClick={() => navigate('/admin/contests')} className="btn btn-secondary">
              Cancel
            </button>
            <button
              onClick={handleApprove}
              disabled={selectedProblems.length === 0 || saving}
              className="btn btn-primary"
            >
              {saving ? (
                <>
                  <svg className="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
                    <circle
                      className="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      strokeWidth="4"
                    />
                    <path
                      className="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
                    />
                  </svg>
                  Saving…
                </>
              ) : (
                `Approve ${selectedProblems.length || ''} Problem${selectedProblems.length !== 1 ? 's' : ''}`
              )}
            </button>
          </div>
        </div>
      </div>

      {/* Problem Detail Modal */}
      {viewingProblem && (
        <div className="modal-overlay" onClick={() => setViewingProblem(null)}>
          <div className="modal-content max-w-4xl w-full" onClick={(e) => e.stopPropagation()}>
            {/* Modal Header */}
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-xl font-bold text-text-primary">{viewingProblem.title}</h2>
              <button
                onClick={() => setViewingProblem(null)}
                className="btn btn-ghost w-8 h-8 p-0 flex items-center justify-center"
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

            <div className="space-y-5">
              {/* Description */}
              <div className="form-group">
                <p className="form-label">Problem Description</p>
                <p className="text-text-primary text-sm whitespace-pre-line leading-relaxed">
                  {viewingProblem.description}
                </p>
              </div>

              {/* Input / Output */}
              <div className="form-row">
                <div className="form-group">
                  <p className="form-label">Input Format</p>
                  <p className="text-text-primary text-sm">{viewingProblem.input_format}</p>
                </div>
                <div className="form-group">
                  <p className="form-label">Output Format</p>
                  <p className="text-text-primary text-sm">{viewingProblem.output_format}</p>
                </div>
              </div>

              {/* Constraints */}
              {viewingProblem.constraints && Object.keys(viewingProblem.constraints).length > 0 && (
                <div className="form-group">
                  <p className="form-label">Constraints</p>
                  <div className="bg-background-tertiary rounded-lg p-3 border border-background-border space-y-1">
                    {Object.entries(viewingProblem.constraints).map(([k, v]) => (
                      <p key={k} className="text-sm font-mono text-text-primary">
                        {k}: {String(v)}
                      </p>
                    ))}
                  </div>
                </div>
              )}

              {/* Sample I/O */}
              <div className="form-row">
                <div className="form-group">
                  <p className="form-label">Sample Input</p>
                  <pre className="bg-background-tertiary rounded-lg p-3 text-xs font-mono text-text-primary border border-background-border overflow-x-auto">
                    {viewingProblem.sample_input}
                  </pre>
                </div>
                <div className="form-group">
                  <p className="form-label">Sample Output</p>
                  <pre className="bg-background-tertiary rounded-lg p-3 text-xs font-mono text-text-primary border border-background-border overflow-x-auto">
                    {viewingProblem.sample_output}
                  </pre>
                </div>
              </div>

              {viewingProblem.sample_explanation && (
                <div className="form-group">
                  <p className="form-label">Explanation</p>
                  <p className="text-text-secondary text-sm">{viewingProblem.sample_explanation}</p>
                </div>
              )}

              {/* Test Cases */}
              <div className="form-group">
                <p className="form-label">Test Cases ({viewingProblem.test_cases?.length ?? 0})</p>
                <div className="space-y-2 max-h-64 overflow-y-auto">
                  {viewingProblem.test_cases?.map((tc, idx) => (
                    <div
                      key={idx}
                      className="bg-background-tertiary rounded-lg p-3 border border-background-border"
                    >
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-xs font-medium text-text-secondary capitalize">
                          {tc.category}
                        </span>
                        {tc.is_sample_visible && (
                          <span className="badge badge-info text-xs">Visible to Students</span>
                        )}
                      </div>
                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <p className="text-xs text-text-muted mb-1">Input</p>
                          <pre className="bg-background-elevated rounded p-2 text-xs font-mono text-text-primary overflow-x-auto">
                            {tc.input}
                          </pre>
                        </div>
                        <div>
                          <p className="text-xs text-text-muted mb-1">Expected Output</p>
                          <pre className="bg-background-elevated rounded p-2 text-xs font-mono text-text-primary overflow-x-auto">
                            {tc.expected_output}
                          </pre>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Solution Approach */}
              {viewingProblem.solution_approach && (
                <div className="form-group">
                  <p className="form-label">
                    Solution Approach{' '}
                    <span className="text-text-muted font-normal">(Instructor Only)</span>
                  </p>
                  <p className="text-text-secondary text-sm">{viewingProblem.solution_approach}</p>
                </div>
              )}
            </div>

            <div className="flex justify-end mt-6">
              <button onClick={() => setViewingProblem(null)} className="btn btn-secondary">
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default ReviewGeneratedContent;
