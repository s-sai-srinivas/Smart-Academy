import { useState, useEffect, useRef, ChangeEvent, FormEvent, MouseEvent } from 'react';
import { Link } from 'react-router-dom';
import api, { contestsAPI, referenceAPI } from '../../services/api';
import { showError, showSuccess } from '../../utils/showAlert';

// ── Types ──

interface Contest {
  contest_id: number;
  title: string;
  description?: string;
  start_time: string;
  end_time: string;
  target_cohort_years?: number[];
  target_cohort?: number;
  target_branch_ids?: number[];
  target_branch_id?: number;
  target_branch_names?: string[];
  target_branch_name?: string;
  practice_enabled?: boolean;
  practice_start_time?: string;
  practice_end_time?: string;
  is_practice_active?: boolean;
  has_quiz?: boolean;
  sections?: Section[];
}

interface Section {
  id?: number;
  section_name: string;
  section_type: 'coding' | 'quiz';
  description: string;
  duration_minutes: number;
  order_index?: number;
}

interface Branch {
  branch_id: number;
  branch_name: string;
}

interface Batch {
  year: number;
}

interface FormData {
  title: string;
  description: string;
  start_time: string;
  end_time: string;
  target_cohorts: number[];
  target_branch_ids: number[];
  has_quiz: boolean;
  quiz_duration_minutes: number;
  sections: Section[];
}

interface PracticeConfig {
  enabled: boolean;
  start_time: string;
  end_time: string;
}

interface Option {
  value: number;
  label: string;
}

interface MultiSelectDropdownProps {
  label: string;
  options: Option[];
  selectedValues: number[];
  onChange: (selected: number[]) => void;
  placeholder: string;
  disabled: boolean;
}

interface StatusInfo {
  status: string;
  label: string;
  class: string;
}

// ── Components ──

function MultiSelectDropdown({
  label,
  options,
  selectedValues,
  onChange,
  placeholder,
  disabled,
}: MultiSelectDropdownProps) {
  const [isOpen, setIsOpen] = useState<boolean>(false);
  const [openDirection, setOpenDirection] = useState<string>('down');
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside as unknown as EventListener);
    return () =>
      document.removeEventListener('mousedown', handleClickOutside as unknown as EventListener);
  }, []);

  useEffect(() => {
    if (!isOpen) return;

    const updateDropdownDirection = () => {
      if (!dropdownRef.current) return;

      const rect = dropdownRef.current.getBoundingClientRect();
      const menuHeight = 260;
      const viewportPadding = 16;
      const spaceBelow = window.innerHeight - rect.bottom - viewportPadding;
      const spaceAbove = rect.top - viewportPadding;

      setOpenDirection(spaceBelow < menuHeight && spaceAbove > spaceBelow ? 'up' : 'down');
    };

    updateDropdownDirection();
    window.addEventListener('resize', updateDropdownDirection);
    window.addEventListener('scroll', updateDropdownDirection, true);

    return () => {
      window.removeEventListener('resize', updateDropdownDirection);
      window.removeEventListener('scroll', updateDropdownDirection, true);
    };
  }, [isOpen]);

  const handleToggle = (value: number) => {
    const newSelected = selectedValues.includes(value)
      ? selectedValues.filter((v) => v !== value)
      : [...selectedValues, value];
    onChange(newSelected);
  };

  const handleRemove = (value: number) => {
    onChange(selectedValues.filter((v) => v !== value));
  };

  const getSelectedItemLabel = (value: number) => {
    const item = options.find((opt) => opt.value === value);
    return item ? item.label : String(value);
  };

  return (
    <div className="relative" ref={dropdownRef}>
      <label className="block text-sm font-medium text-text-primary mb-1">{label}</label>

      {/* Selected items display */}
      <div
        className={`input min-h-[42px] cursor-pointer flex items-center gap-2 flex-wrap ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
        onClick={() => !disabled && setIsOpen(!isOpen)}
      >
        {selectedValues.length === 0 ? (
          <span className="text-text-muted">{placeholder}</span>
        ) : (
          selectedValues.map((value) => (
            <span
              key={value}
              className="inline-flex items-center gap-1 px-2 py-1 bg-accent-secondary/20 text-accent-secondary rounded text-sm"
            >
              {getSelectedItemLabel(value)}
              {!disabled && (
                <button
                  type="button"
                  onClick={(e: MouseEvent<HTMLButtonElement>) => {
                    e.stopPropagation();
                    handleRemove(value);
                  }}
                  className="hover:text-accent-danger"
                >
                  ×
                </button>
              )}
            </span>
          ))
        )}
        <span className="ml-auto text-text-muted">▼</span>
      </div>

      {/* Dropdown menu */}
      {isOpen && !disabled && (
        <div
          className={`absolute left-0 z-[70] w-full rounded-lg border border-background-border bg-background-secondary shadow-elevated ${
            openDirection === 'up' ? 'bottom-full mb-1' : 'top-full mt-1'
          }`}
        >
          <div
            className="max-h-60 overflow-y-auto overscroll-contain"
            onWheel={(e) => e.stopPropagation()}
          >
            {options.length === 0 ? (
              <div className="p-3 text-text-muted text-sm">No options available</div>
            ) : (
              options.map((option) => (
                <div
                  key={option.value}
                  className={`px-3 py-2 cursor-pointer hover:bg-background-tertiary flex items-center gap-2 ${
                    selectedValues.includes(option.value) ? 'bg-accent-secondary/10' : ''
                  }`}
                  onClick={() => handleToggle(option.value)}
                >
                  <input
                    type="checkbox"
                    checked={selectedValues.includes(option.value)}
                    onChange={() => {}}
                    className="w-4 h-4 accent-accent-secondary"
                    onClick={(e: MouseEvent<HTMLInputElement>) => e.stopPropagation()}
                  />
                  <span className="text-text-primary">{option.label}</span>
                </div>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function AdminContests() {
  const [contests, setContests] = useState<Contest[]>([]);
  const [showCreateForm, setShowCreateForm] = useState<boolean>(false);
  const [editingContest, setEditingContest] = useState<Contest | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [saving, setSaving] = useState<boolean>(false);
  const [error, setError] = useState<string>('');
  const [success, setSuccess] = useState<string>('');

  // Practice mode state
  const [showPracticeModal, setShowPracticeModal] = useState<boolean>(false);
  const [practiceContest, setPracticeContest] = useState<Contest | null>(null);
  const [practiceConfig, setPracticeConfig] = useState<PracticeConfig>({
    enabled: false,
    start_time: '',
    end_time: '',
  });
  const [savingPractice, setSavingPractice] = useState<boolean>(false);

  // Reference data
  const [branches, setBranches] = useState<Branch[]>([]);
  const [batches, setBatches] = useState<Batch[]>([]);

  // Form state
  const [formData, setFormData] = useState<FormData>({
    title: '',
    description: '',
    start_time: '',
    end_time: '',
    target_cohorts: [],
    target_branch_ids: [],
    has_quiz: false,
    quiz_duration_minutes: 30,
    sections: [],
  });

  useEffect(() => {
    loadContests();
    loadReferenceData();
  }, []);

  const loadContests = async () => {
    try {
      setLoading(true);
      const data = (await contestsAPI.getAll()) as Contest[] | { contests: Contest[] };
      setContests(Array.isArray(data) ? data : data?.contests || []);
    } catch (err) {
      console.error('Failed to load contests:', err);
      showError('Failed to load contests');
    } finally {
      setLoading(false);
    }
  };

  const loadReferenceData = async () => {
    try {
      const [branchesData, batchesData] = await Promise.all([
        referenceAPI.getBranches() as Promise<Branch[]>,
        referenceAPI.getBatches() as Promise<Batch[]>,
      ]);
      setBranches(branchesData || []);
      setBatches(batchesData || []);
    } catch (err) {
      console.error('Failed to load reference data:', err);
    }
  };

  // Convert batches to options format
  const batchOptions: Option[] = batches.map((year) => ({
    value: year.year,
    label: `Batch ${year.year}`,
  }));

  // Convert branches to options format
  const branchOptions: Option[] = branches.map((branch) => ({
    value: branch.branch_id,
    label: branch.branch_name,
  }));

  const handleInputChange = (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const handleCohortChange = (selected: number[]) => {
    setFormData((prev) => ({
      ...prev,
      target_cohorts: selected,
    }));
  };

  const handleBranchChange = (selected: number[]) => {
    setFormData((prev) => ({
      ...prev,
      target_branch_ids: selected,
    }));
  };

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setSaving(true);
    setError('');
    setSuccess('');

    try {
      const toRFC3339 = (dt: string) => (dt ? new Date(dt).toISOString() : '');

      if (formData.target_cohorts.length === 0) {
        showError('At least one target batch is required');
        setSaving(false);
        return;
      }

      const submitData = {
        title: formData.title,
        description: formData.description,
        start_time: toRFC3339(formData.start_time),
        end_time: toRFC3339(formData.end_time),
        target_cohorts: formData.target_cohorts,
        target_branch_ids: formData.target_branch_ids,
        has_quiz: formData.sections.some((s) => s.section_type === 'quiz'),
        quiz_duration_minutes: formData.sections
          .filter((s) => s.section_type === 'quiz')
          .reduce((sum, s) => sum + s.duration_minutes, 0),
      };

      // Save sections if contest was created
      let createdContestId: number | undefined;
      if (!editingContest) {
        const created = (await contestsAPI.create(submitData)) as { contest_id?: number };
        createdContestId = created?.contest_id;
        showSuccess('Contest created successfully!');
      } else {
        await contestsAPI.update(editingContest.contest_id, submitData);
        createdContestId = editingContest.contest_id;
        showSuccess('Contest updated successfully!');
      }

      // Save sections
      if (createdContestId && formData.sections.length > 0) {
        await api.put(`/admin/contests/${createdContestId}/sections`, {
          sections: formData.sections,
        });
      }

      setFormData({
        title: '',
        description: '',
        start_time: '',
        end_time: '',
        target_cohorts: [],
        target_branch_ids: [],
        has_quiz: false,
        quiz_duration_minutes: 30,
        sections: [],
      });

      setShowCreateForm(false);
      setEditingContest(null);
      loadContests();
    } catch (err) {
      console.error('Failed to save contest:', err);
      showError(err instanceof Error ? err.message : 'Failed to save contest');
    } finally {
      setSaving(false);
    }
  };

  const handleEdit = (contest: Contest) => {
    setEditingContest(contest);
    setFormData({
      title: contest.title,
      description: contest.description || '',
      start_time: contest.start_time ? new Date(contest.start_time).toISOString().slice(0, 16) : '',
      end_time: contest.end_time ? new Date(contest.end_time).toISOString().slice(0, 16) : '',
      target_cohorts:
        contest.target_cohort_years || (contest.target_cohort ? [contest.target_cohort] : []),
      target_branch_ids:
        contest.target_branch_ids || (contest.target_branch_id ? [contest.target_branch_id] : []),
      has_quiz: contest.has_quiz || false,
      quiz_duration_minutes: 30,
      sections: contest.sections || [],
    });
    setShowCreateForm(true);
  };

  const handleDelete = async (contestId: number) => {
    if (!confirm('Are you sure you want to delete this contest?')) return;

    try {
      await contestsAPI.delete(contestId);
      loadContests();
    } catch (err) {
      console.error('Failed to delete contest:', err);
      showError(err instanceof Error ? err.message : 'Failed to delete contest');
    }
  };

  const handleCancel = () => {
    setShowCreateForm(false);
    setEditingContest(null);
    setError('');
    setSuccess('');
  };

  // Practice mode functions
  const handleOpenPracticeModal = (contest: Contest) => {
    setPracticeContest(contest);
    setPracticeConfig({
      enabled: contest.practice_enabled || false,
      start_time: contest.practice_start_time
        ? new Date(contest.practice_start_time).toISOString().slice(0, 16)
        : '',
      end_time: contest.practice_end_time
        ? new Date(contest.practice_end_time).toISOString().slice(0, 16)
        : '',
    });
    setShowPracticeModal(true);
  };

  const handlePracticeConfigChange = (e: ChangeEvent<HTMLInputElement>) => {
    const { name, value, type, checked } = e.target;
    setPracticeConfig((prev) => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value,
    }));
  };

  const handleSavePracticeMode = async () => {
    if (!practiceContest) return;
    setSavingPractice(true);

    try {
      const toRFC3339 = (dt: string) => (dt ? new Date(dt).toISOString() : null);

      await contestsAPI.enablePracticeMode(practiceContest.contest_id, {
        enabled: practiceConfig.enabled,
        start_time: practiceConfig.enabled ? toRFC3339(practiceConfig.start_time) : null,
        end_time: practiceConfig.enabled ? toRFC3339(practiceConfig.end_time) : null,
      });

      showSuccess(`Practice mode ${practiceConfig.enabled ? 'enabled' : 'disabled'} successfully!`);
      setShowPracticeModal(false);
      setPracticeContest(null);
      loadContests();
    } catch (err) {
      console.error('Failed to update practice mode:', err);
      showError(err instanceof Error ? err.message : 'Failed to update practice mode');
    } finally {
      setSavingPractice(false);
    }
  };

  const getContestStatus = (contest: Contest): StatusInfo => {
    const now = new Date();
    const start = new Date(contest.start_time);
    const end = new Date(contest.end_time);

    if (now < start) return { status: 'upcoming', label: 'Upcoming', class: 'badge-info' };
    if (now > end) return { status: 'ended', label: 'Ended', class: 'badge-neutral' };
    return { status: 'active', label: 'Active', class: 'badge-success' };
  };

  const formatDateTime = (dateStr: string) => {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      hour12: true,
    });
  };

  return (
    <div className="p-6 max-w-6xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-3xl font-bold text-text-primary">Contest Management</h1>
        <div className="flex gap-3">
          <Link
            to="/admin/contests/generate"
            className="px-4 py-2 bg-gradient-to-r from-purple-600 to-indigo-600 text-white rounded-lg hover:from-purple-700 hover:to-indigo-700 transition-colors font-medium"
          >
            AI Generate Problems
          </Link>
          {!showCreateForm && (
            <button onClick={() => setShowCreateForm(true)} className="btn btn-primary">
              + Create Contest
            </button>
          )}
        </div>
      </div>

      {error && (
        <div className="card bg-accent-danger/10 border-accent-danger text-accent-danger p-4 mb-6">
          {error}
        </div>
      )}

      {success && (
        <div className="card bg-accent-success/10 border-accent-success text-accent-success p-4 mb-6">
          {success}
        </div>
      )}

      {showCreateForm ? (
        <div className="card mb-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-bold text-text-primary">
              {editingContest ? 'Edit Contest' : 'Create New Contest'}
            </h2>
            <button onClick={handleCancel} className="text-text-muted hover:text-text-primary">
              Cancel
            </button>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-text-primary mb-1">
                Contest Title *
              </label>
              <input
                type="text"
                name="title"
                value={formData.title}
                onChange={handleInputChange}
                required
                className="input"
                placeholder="e.g., Weekly Coding Challenge #5"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-text-primary mb-1">
                Description
              </label>
              <textarea
                name="description"
                value={formData.description}
                onChange={handleInputChange}
                rows={3}
                className="input"
                placeholder="Contest description and rules..."
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-text-primary mb-1">
                  Start Time *
                </label>
                <input
                  type="datetime-local"
                  name="start_time"
                  value={formData.start_time}
                  onChange={handleInputChange}
                  required
                  className="input"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-text-primary mb-1">
                  End Time *
                </label>
                <input
                  type="datetime-local"
                  name="end_time"
                  value={formData.end_time}
                  onChange={handleInputChange}
                  required
                  className="input"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <MultiSelectDropdown
                label="Target Batches *"
                options={batchOptions}
                selectedValues={formData.target_cohorts}
                onChange={handleCohortChange}
                placeholder="Select batches..."
                disabled={false}
              />
              <MultiSelectDropdown
                label="Target Branches (Optional)"
                options={branchOptions}
                selectedValues={formData.target_branch_ids}
                onChange={handleBranchChange}
                placeholder="All branches"
                disabled={formData.target_cohorts.length === 0}
              />
            </div>

            {/* Sections Builder */}
            <div className="p-4 bg-background-tertiary rounded-lg border border-background-border">
              <div className="flex items-center justify-between mb-3">
                <div>
                  <span className="text-sm font-medium text-text-primary">Contest Sections</span>
                  <p className="text-xs text-text-secondary mt-0.5">
                    Add sections to your contest. Each section can be Coding (problems) or Quiz (MCQ
                    questions).
                  </p>
                </div>
                <button
                  type="button"
                  onClick={() =>
                    setFormData((prev) => ({
                      ...prev,
                      sections: [
                        ...prev.sections,
                        {
                          section_name: '',
                          section_type: 'coding',
                          description: '',
                          duration_minutes: 30,
                        },
                      ],
                    }))
                  }
                  className="btn btn-secondary text-xs px-3 py-1.5"
                >
                  + Add Section
                </button>
              </div>

              {formData.sections.length === 0 ? (
                <p className="text-text-muted text-sm text-center py-4">
                  No sections yet. Add coding or quiz sections to structure your contest.
                </p>
              ) : (
                <div className="space-y-3">
                  {formData.sections.map((section, idx) => (
                    <div
                      key={idx}
                      className="bg-background-elevated rounded-lg border border-background-border p-3"
                    >
                      <div className="flex items-center gap-3">
                        <span className="text-xs font-mono text-text-muted w-6">#{idx + 1}</span>
                        <input
                          type="text"
                          value={section.section_name}
                          onChange={(e) => {
                            const updated = [...formData.sections];
                            updated[idx] = { ...updated[idx], section_name: e.target.value };
                            setFormData((prev) => ({ ...prev, sections: updated }));
                          }}
                          placeholder="Section name (e.g. Round 1)"
                          className="input flex-1 text-sm"
                        />
                        <select
                          value={section.section_type}
                          onChange={(e) => {
                            const updated = [...formData.sections];
                            updated[idx] = {
                              ...updated[idx],
                              section_type: e.target.value as 'coding' | 'quiz',
                            };
                            setFormData((prev) => ({ ...prev, sections: updated }));
                          }}
                          className="select text-sm w-28"
                        >
                          <option value="coding">Coding</option>
                          <option value="quiz">Quiz</option>
                        </select>
                        {section.section_type === 'quiz' && (
                          <input
                            type="number"
                            value={section.duration_minutes}
                            onChange={(e) => {
                              const updated = [...formData.sections];
                              updated[idx] = {
                                ...updated[idx],
                                duration_minutes: parseInt(e.target.value) || 0,
                              };
                              setFormData((prev) => ({ ...prev, sections: updated }));
                            }}
                            min={5}
                            max={180}
                            className="input w-20 text-sm"
                            placeholder="Min"
                          />
                        )}
                        <button
                          type="button"
                          onClick={() => {
                            const updated = formData.sections.filter((_, i) => i !== idx);
                            setFormData((prev) => ({ ...prev, sections: updated }));
                          }}
                          className="text-text-muted hover:text-accent-danger transition-colors p-1"
                        >
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
                              d="M6 18L18 6M6 6l12 12"
                            />
                          </svg>
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div className="flex justify-end gap-3 pt-4">
              <button
                type="button"
                onClick={handleCancel}
                className="px-4 py-2 border border-background-border rounded-lg text-text-primary hover:bg-background-tertiary transition-colors"
              >
                Cancel
              </button>
              <button type="submit" disabled={saving} className="btn btn-primary">
                {saving ? 'Saving...' : editingContest ? 'Update Contest' : 'Create Contest'}
              </button>
            </div>
          </form>
        </div>
      ) : (
        <>
          {loading ? (
            <div className="flex items-center justify-center h-64">
              <p className="text-text-muted">Loading contests...</p>
            </div>
          ) : contests.length === 0 ? (
            <div className="card text-center py-16">
              <div className="text-6xl mb-4">🏆</div>
              <h3 className="text-xl font-semibold text-text-primary mb-2">No Contests Yet</h3>
              <p className="text-text-secondary mb-6">Create your first contest to get started!</p>
              <button onClick={() => setShowCreateForm(true)} className="btn btn-primary">
                + Create Contest
              </button>
            </div>
          ) : (
            <div className="space-y-4">
              {contests.map((contest) => {
                const status = getContestStatus(contest);
                return (
                  <div key={contest.contest_id} className="card">
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-3 mb-2">
                          <h3 className="text-xl font-semibold text-text-primary">
                            {contest.title}
                          </h3>
                          <span className={`badge ${status.class}`}>{status.label}</span>
                        </div>
                        {contest.description && (
                          <p className="text-text-secondary text-sm mb-3">{contest.description}</p>
                        )}
                        <div className="flex flex-wrap items-center gap-3 text-sm text-text-muted">
                          <span>Start: {formatDateTime(contest.start_time)}</span>
                          <span>End: {formatDateTime(contest.end_time)}</span>
                          {/* Display multiple batches */}
                          {contest.target_cohort_years && contest.target_cohort_years.length > 0 ? (
                            <span className="inline-flex items-center gap-1">
                              Batches:
                              {contest.target_cohort_years.map((y) => (
                                <span
                                  key={y}
                                  className="px-2 py-0.5 bg-accent-secondary/20 rounded text-xs"
                                >
                                  {y}
                                </span>
                              ))}
                            </span>
                          ) : (
                            contest.target_cohort && (
                              <span className="px-2 py-0.5 bg-accent-secondary/20 rounded text-xs">
                                Batch {contest.target_cohort}
                              </span>
                            )
                          )}
                          {/* Display multiple branches */}
                          {contest.target_branch_names && contest.target_branch_names.length > 0 ? (
                            <span className="inline-flex items-center gap-1">
                              Branches:
                              {contest.target_branch_names.map((name, i) => (
                                <span
                                  key={i}
                                  className="px-2 py-0.5 bg-accent-primary/20 rounded text-xs"
                                >
                                  {name}
                                </span>
                              ))}
                            </span>
                          ) : (
                            contest.target_branch_name && (
                              <span className="px-2 py-0.5 bg-accent-primary/20 rounded text-xs">
                                {contest.target_branch_name}
                              </span>
                            )
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Link
                          to={`/admin/contests/${contest.contest_id}/problems`}
                          className="px-3 py-1 text-sm border border-background-border rounded hover:bg-background-tertiary transition-colors"
                        >
                          Problems
                        </Link>
                        {/* Practice Mode button for ended contests */}
                        {status.status === 'ended' && (
                          <button
                            onClick={() => handleOpenPracticeModal(contest)}
                            className={`px-3 py-1 text-sm border rounded transition-colors ${
                              contest.practice_enabled
                                ? 'border-accent-success text-accent-success hover:bg-accent-success/10'
                                : 'border-accent-primary text-accent-primary hover:bg-accent-primary/10'
                            }`}
                          >
                            {contest.practice_enabled ? 'Practice ON' : 'Practice'}
                          </button>
                        )}
                        {/* Show practice mode status badge */}
                        {contest.is_practice_active && (
                          <span className="px-2 py-1 text-xs bg-accent-success/20 text-accent-success rounded">
                            Practice Active
                          </span>
                        )}
                        <button
                          onClick={() => handleEdit(contest)}
                          className="px-3 py-1 text-sm border border-background-border rounded hover:bg-background-tertiary transition-colors"
                        >
                          Edit
                        </button>
                        <button
                          onClick={() => handleDelete(contest.contest_id)}
                          className="px-3 py-1 text-sm border border-accent-danger text-accent-danger rounded hover:bg-accent-danger/10 transition-colors"
                        >
                          Delete
                        </button>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </>
      )}

      {/* Practice Mode Configuration Modal */}
      {showPracticeModal && practiceContest && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-background-primary rounded-lg p-6 w-full max-w-md border border-background-border">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-xl font-semibold text-text-primary">
                Practice Mode: {practiceContest.title}
              </h3>
              <button
                onClick={() => setShowPracticeModal(false)}
                className="text-text-muted hover:text-text-primary"
              >
                ×
              </button>
            </div>

            <p className="text-text-secondary text-sm mb-4">
              Practice mode allows students to access and solve contest problems after the contest
              has ended. Practice submissions do not count toward the leaderboard.
            </p>

            <div className="space-y-4">
              <div className="flex items-center gap-3">
                <input
                  type="checkbox"
                  name="enabled"
                  checked={practiceConfig.enabled}
                  onChange={handlePracticeConfigChange}
                  className="w-5 h-5 accent-accent-primary"
                />
                <label className="text-text-primary">Enable Practice Mode</label>
              </div>

              {practiceConfig.enabled && (
                <div className="space-y-3 border-l-2 border-accent-primary/30 pl-4">
                  <div>
                    <label className="block text-sm font-medium text-text-primary mb-1">
                      Practice Start Time (Optional)
                    </label>
                    <input
                      type="datetime-local"
                      name="start_time"
                      value={practiceConfig.start_time}
                      onChange={handlePracticeConfigChange}
                      className="input"
                    />
                    <p className="text-text-muted text-xs mt-1">Leave empty to start immediately</p>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-text-primary mb-1">
                      Practice End Time (Optional)
                    </label>
                    <input
                      type="datetime-local"
                      name="end_time"
                      value={practiceConfig.end_time}
                      onChange={handlePracticeConfigChange}
                      className="input"
                    />
                    <p className="text-text-muted text-xs mt-1">
                      Leave empty for unlimited practice
                    </p>
                  </div>
                </div>
              )}
            </div>

            <div className="flex justify-end gap-3 mt-6">
              <button
                onClick={() => setShowPracticeModal(false)}
                className="px-4 py-2 border border-background-border rounded-lg text-text-primary hover:bg-background-tertiary transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleSavePracticeMode}
                disabled={savingPractice}
                className="btn btn-primary"
              >
                {savingPractice ? 'Saving...' : 'Save'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default AdminContests;
