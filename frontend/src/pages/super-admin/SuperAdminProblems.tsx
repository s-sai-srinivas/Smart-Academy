import { useState, useEffect, ChangeEvent, MouseEvent } from 'react';
import { superAdminPracticeAPI } from '../../services/api';
import { showError, showSuccess } from '../../utils/showAlert';
import { BookOpen, Plus, Edit2, Trash2, ChevronDown, ChevronUp, Check, X } from 'lucide-react';
import CreateProblemModal from '../../components/admin/CreateProblemModal';

interface TestCase {
  id?: string;
  input: string;
  expected_output: string;
  is_sample: boolean;
  points: number;
}

interface Problem {
  id: string;
  title: string;
  description?: string;
  difficulty?: string;
  time_limit?: number;
  memory_limit?: number;
  tags?: string;
  points?: number;
  test_cases?: TestCase[];
}

const SuperAdminProblems = () => {
  const [problems, setProblems] = useState<Problem[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [showCreate, setShowCreate] = useState<boolean>(false);
  const [editingProblem, setEditingProblem] = useState<Problem | null>(null);
  const [expandedProblem, setExpandedProblem] = useState<string | null>(null);
  const [showAddTestCases, setShowAddTestCases] = useState<Problem | null>(null);

  useEffect(() => {
    fetchProblems();
  }, []);

  const fetchProblems = async () => {
    try {
      setLoading(true);
      const data = (await superAdminPracticeAPI.getProblems()) as { problems?: Problem[] };
      setProblems(data.problems || []);
    } catch {
      showError('Failed to load problems');
    } finally {
      setLoading(false);
    }
  };

  const handleUpdate = async () => {
    if (!editingProblem) return;
    try {
      await superAdminPracticeAPI.update(editingProblem.id, {
        title: editingProblem.title,
        description: editingProblem.description,
        difficulty: editingProblem.difficulty,
        time_limit: editingProblem.time_limit,
        memory_limit: editingProblem.memory_limit,
        tags: editingProblem.tags,
        points: editingProblem.points,
      });
      showSuccess('Problem updated successfully!');
      setEditingProblem(null);
      fetchProblems();
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to update problem');
    }
  };

  const handleDelete = async (id: string) => {
    if (
      !confirm('Are you sure you want to delete this global problem? This action cannot be undone.')
    )
      return;
    try {
      await superAdminPracticeAPI.delete(id);
      showSuccess('Problem deleted successfully!');
      fetchProblems();
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to delete problem');
    }
  };

  const getDifficultyColor = (difficulty?: string): string => {
    switch (difficulty) {
      case 'easy':
        return 'text-green-400 bg-green-400/10';
      case 'medium':
        return 'text-yellow-400 bg-yellow-400/10';
      case 'hard':
        return 'text-red-400 bg-red-400/10';
      default:
        return 'text-gray-400 bg-gray-400/10';
    }
  };

  return (
    <div className="space-y-6 max-w-6xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary mb-1">Global Practice Problems</h1>
          <p className="text-text-secondary text-sm">
            Create and manage problems that are accessible to all students across colleges
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="btn btn-primary flex items-center gap-2"
        >
          <Plus className="w-4 h-4" />
          Add Problem
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-5">
        <div className="card-elevated flex items-center gap-4 border-l-4 border-accent-secondary">
          <BookOpen className="w-8 h-8 text-accent-secondary" />
          <div>
            <div className="text-3xl font-bold text-text-primary">
              {loading ? '...' : problems.length}
            </div>
            <div className="text-sm text-text-secondary">Total Problems</div>
          </div>
        </div>
        <div className="card-elevated flex items-center gap-4 border-l-4 border-green-500">
          <Check className="w-8 h-8 text-green-500" />
          <div>
            <div className="text-3xl font-bold text-text-primary">
              {loading ? '...' : problems.filter((p) => (p.test_cases?.length ?? 0) > 0).length}
            </div>
            <div className="text-sm text-text-secondary">With Test Cases</div>
          </div>
        </div>
        <div className="card-elevated flex items-center gap-4 border-l-4 border-yellow-500">
          <Edit2 className="w-8 h-8 text-yellow-500" />
          <div>
            <div className="text-3xl font-bold text-text-primary">
              {loading ? '...' : problems.filter((p) => p.difficulty === 'easy').length}
            </div>
            <div className="text-sm text-text-secondary">Easy Problems</div>
          </div>
        </div>
      </div>

      {/* Problems List */}
      {loading ? (
        <div className="text-center py-12 text-text-secondary">Loading problems...</div>
      ) : problems.length === 0 ? (
        <div className="text-center py-12">
          <BookOpen className="w-12 h-12 mx-auto text-text-muted mb-3" />
          <p className="text-text-secondary">No global problems yet</p>
          <button
            onClick={() => setShowCreate(true)}
            className="mt-3 text-accent-secondary hover:underline text-sm"
          >
            Create your first problem
          </button>
        </div>
      ) : (
        <div className="space-y-3">
          {problems.map((problem) => (
            <ProblemCard
              key={problem.id}
              problem={problem}
              isExpanded={expandedProblem === problem.id}
              onToggle={() =>
                setExpandedProblem(expandedProblem === problem.id ? null : problem.id)
              }
              onEdit={() => setEditingProblem({ ...problem })}
              onDelete={() => handleDelete(problem.id)}
              onAddTestCases={() => setShowAddTestCases(problem)}
              getDifficultyColor={getDifficultyColor}
            />
          ))}
        </div>
      )}

      {/* Create Problem Modal - Reuse shared component */}
      {showCreate && (
        <CreateProblemModal
          isGlobal={true}
          onClose={() => setShowCreate(false)}
          onSuccess={() => {
            setShowCreate(false);
            fetchProblems();
          }}
        />
      )}

      {/* Edit Problem Modal */}
      {editingProblem && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="w-full max-w-2xl bg-background-secondary border border-background-border rounded-xl p-6 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-5">
              <h2 className="text-xl font-bold text-text-primary">Edit Problem</h2>
              <button
                onClick={() => setEditingProblem(null)}
                className="text-text-muted hover:text-text-primary"
              >
                <X className="w-5 h-5" />
              </button>
            </div>
            <div className="space-y-4">
              <div>
                <label className="block text-xs text-text-secondary mb-1">Title *</label>
                <input
                  type="text"
                  value={editingProblem.title}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    setEditingProblem({ ...editingProblem, title: e.target.value })
                  }
                  className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary"
                />
              </div>
              <div>
                <label className="block text-xs text-text-secondary mb-1">Description *</label>
                <textarea
                  rows={6}
                  value={editingProblem.description || ''}
                  onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                    setEditingProblem({ ...editingProblem, description: e.target.value })
                  }
                  className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary resize-none"
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs text-text-secondary mb-1">Difficulty *</label>
                  <select
                    value={editingProblem.difficulty}
                    onChange={(e: ChangeEvent<HTMLSelectElement>) =>
                      setEditingProblem({ ...editingProblem, difficulty: e.target.value })
                    }
                    className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary"
                  >
                    <option value="easy">Easy</option>
                    <option value="medium">Medium</option>
                    <option value="hard">Hard</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs text-text-secondary mb-1">Points</label>
                  <input
                    type="number"
                    value={editingProblem.points || 0}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      setEditingProblem({
                        ...editingProblem,
                        points: parseInt(e.target.value) || 0,
                      })
                    }
                    className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary"
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs text-text-secondary mb-1">Time Limit (ms)</label>
                  <input
                    type="number"
                    value={editingProblem.time_limit}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      setEditingProblem({
                        ...editingProblem,
                        time_limit: parseInt(e.target.value) || 2000,
                      })
                    }
                    className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary"
                  />
                </div>
                <div>
                  <label className="block text-xs text-text-secondary mb-1">
                    Memory Limit (KB)
                  </label>
                  <input
                    type="number"
                    value={editingProblem.memory_limit}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      setEditingProblem({
                        ...editingProblem,
                        memory_limit: parseInt(e.target.value) || 256000,
                      })
                    }
                    className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary"
                  />
                </div>
              </div>
              <div>
                <label className="block text-xs text-text-secondary mb-1">
                  Tags (comma-separated)
                </label>
                <input
                  type="text"
                  value={editingProblem.tags || ''}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    setEditingProblem({ ...editingProblem, tags: e.target.value })
                  }
                  className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary"
                />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button onClick={() => setEditingProblem(null)} className="btn btn-secondary">
                  Cancel
                </button>
                <button onClick={handleUpdate} className="btn btn-primary">
                  Save Changes
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Add Test Cases Modal */}
      {showAddTestCases && (
        <AddTestCasesModal
          problem={showAddTestCases}
          onClose={() => setShowAddTestCases(null)}
          onSuccess={() => {
            setShowAddTestCases(null);
            fetchProblems();
          }}
        />
      )}
    </div>
  );
};

// Separate component for problem card
interface ProblemCardProps {
  problem: Problem;
  isExpanded: boolean;
  onToggle: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onAddTestCases: () => void;
  getDifficultyColor: (difficulty?: string) => string;
}

function ProblemCard({
  problem,
  isExpanded,
  onToggle,
  onEdit,
  onDelete,
  onAddTestCases,
  getDifficultyColor,
}: ProblemCardProps) {
  return (
    <div className="card-elevated">
      <div className="flex items-center justify-between cursor-pointer" onClick={onToggle}>
        <div className="flex items-center gap-4 min-w-0 flex-1">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <h3 className="text-text-primary font-medium truncate">{problem.title}</h3>
              <span
                className={`text-xs font-medium px-2 py-0.5 rounded ${getDifficultyColor(problem?.difficulty)}`}
              >
                {problem?.difficulty
                  ? problem.difficulty.charAt(0).toUpperCase() + problem.difficulty.slice(1)
                  : ''}
              </span>
            </div>
            <div className="flex items-center gap-4 mt-1 text-xs text-text-muted">
              <span>{problem.test_cases?.length || 0} test cases</span>
              {problem.tags && <span>Tags: {problem.tags}</span>}
              <span>Time: {problem.time_limit}ms</span>
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={(e: MouseEvent<HTMLButtonElement>) => {
              e.stopPropagation();
              onEdit();
            }}
            className="p-2 rounded-lg text-text-secondary hover:text-text-primary hover:bg-background-tertiary transition-colors"
            title="Edit"
          >
            <Edit2 className="w-4 h-4" />
          </button>
          <button
            onClick={(e: MouseEvent<HTMLButtonElement>) => {
              e.stopPropagation();
              onDelete();
            }}
            className="p-2 rounded-lg text-text-secondary hover:text-red-400 hover:bg-red-500/10 transition-colors"
            title="Delete"
          >
            <Trash2 className="w-4 h-4" />
          </button>
          {isExpanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
        </div>
      </div>

      {/* Expanded Content */}
      {isExpanded && (
        <div className="mt-4 pt-4 border-t border-background-border">
          <div className="flex items-center justify-between mb-3">
            <span className="text-sm font-medium text-text-primary">Test Cases</span>
            <button
              onClick={onAddTestCases}
              className="btn btn-secondary text-xs flex items-center gap-1"
            >
              <Plus className="w-3 h-3" /> Add Test Cases
            </button>
          </div>
          {(problem.test_cases?.length ?? 0) > 0 ? (
            <div className="space-y-2">
              {problem.test_cases?.map((tc, i) => (
                <div key={tc.id || i} className="bg-background-tertiary rounded-lg p-3">
                  <div className="flex items-center gap-2 mb-2">
                    <span className="text-xs font-medium text-text-secondary">Test {i + 1}</span>
                    {tc.is_sample && (
                      <span className="text-xs px-1.5 py-0.5 rounded bg-blue-500/20 text-blue-400">
                        Sample
                      </span>
                    )}
                    <span className="text-xs text-text-muted">{tc.points} pts</span>
                  </div>
                  <div className="grid grid-cols-2 gap-2 text-xs">
                    <div>
                      <span className="text-text-muted">Input:</span>
                      <pre className="mt-1 p-2 bg-background-secondary rounded overflow-auto max-h-24">
                        {tc.input}
                      </pre>
                    </div>
                    <div>
                      <span className="text-text-muted">Output:</span>
                      <pre className="mt-1 p-2 bg-background-secondary rounded overflow-auto max-h-24">
                        {tc.expected_output}
                      </pre>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-text-muted text-sm">
              No test cases yet. Click "Add Test Cases" to add some.
            </p>
          )}
        </div>
      )}
    </div>
  );
}

// Separate component for adding test cases (multiple at once)
interface AddTestCasesModalProps {
  problem: Problem;
  onClose: () => void;
  onSuccess: () => void;
}

function AddTestCasesModal({ problem, onClose, onSuccess }: AddTestCasesModalProps) {
  const [testCases, setTestCases] = useState<TestCase[]>([
    { input: '', expected_output: '', is_sample: false, points: 10 },
  ]);
  const [loading, setLoading] = useState<boolean>(false);

  const addTestCase = () => {
    setTestCases([...testCases, { input: '', expected_output: '', is_sample: false, points: 10 }]);
  };

  const removeTestCase = (index: number) => {
    if (testCases.length > 1) {
      setTestCases(testCases.filter((_, i) => i !== index));
    }
  };

  const updateTestCase = (
    index: number,
    field: keyof TestCase,
    value: string | boolean | number
  ) => {
    const updated = [...testCases];
    updated[index] = { ...updated[index], [field]: value };
    setTestCases(updated);
  };

  const handleSubmit = async () => {
    setLoading(true);
    try {
      const validTestCases = testCases.filter((tc) => tc.input && tc.expected_output);
      if (validTestCases.length === 0) {
        showError('Please add at least one valid test case');
        setLoading(false);
        return;
      }
      for (const tc of validTestCases) {
        await superAdminPracticeAPI.createTestCase(problem.id, tc);
      }
      showSuccess('Test cases added successfully!');
      onSuccess();
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to add test cases');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="w-full max-w-2xl bg-background-secondary border border-background-border rounded-xl p-6 max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-xl font-bold text-text-primary">Add Test Cases</h2>
          <button onClick={onClose} className="text-text-muted hover:text-text-primary">
            <X className="w-5 h-5" />
          </button>
        </div>
        <p className="text-sm text-text-secondary mb-4">
          Adding test cases to <strong className="text-text-primary">{problem.title}</strong>
        </p>

        <div className="space-y-3">
          {testCases.map((tc, index) => (
            <div
              key={index}
              className="border border-background-border rounded-lg p-4 bg-background-tertiary"
            >
              <div className="flex items-center justify-between mb-3">
                <span className="text-sm font-medium text-text-primary">Test Case {index + 1}</span>
                {testCases.length > 1 && (
                  <button
                    type="button"
                    onClick={() => removeTestCase(index)}
                    className="text-xs text-red-400 hover:text-red-300"
                  >
                    Remove
                  </button>
                )}
              </div>
              <div className="grid grid-cols-1 gap-3">
                <div>
                  <label className="block text-xs text-text-muted mb-1">Input</label>
                  <textarea
                    rows={2}
                    value={tc.input}
                    onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                      updateTestCase(index, 'input', e.target.value)
                    }
                    className="w-full px-3 py-2 bg-background-secondary border border-background-border rounded-lg text-text-primary text-sm font-mono focus:outline-none focus:border-accent-secondary resize-none"
                  />
                </div>
                <div>
                  <label className="block text-xs text-text-muted mb-1">Expected Output</label>
                  <textarea
                    rows={2}
                    value={tc.expected_output}
                    onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                      updateTestCase(index, 'expected_output', e.target.value)
                    }
                    className="w-full px-3 py-2 bg-background-secondary border border-background-border rounded-lg text-text-primary text-sm font-mono focus:outline-none focus:border-accent-secondary resize-none"
                  />
                </div>
              </div>
              <div className="flex items-center gap-4 mt-3">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={tc.is_sample}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      updateTestCase(index, 'is_sample', e.target.checked)
                    }
                    className="rounded"
                  />
                  <span className="text-xs text-text-secondary">Sample (visible)</span>
                </label>
                <div className="flex items-center gap-2">
                  <span className="text-xs text-text-muted">Points:</span>
                  <input
                    type="number"
                    value={tc.points}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      updateTestCase(index, 'points', parseInt(e.target.value) || 10)
                    }
                    className="w-16 px-2 py-1 bg-background-secondary border border-background-border rounded text-text-primary text-xs focus:outline-none focus:border-accent-secondary"
                  />
                </div>
              </div>
            </div>
          ))}
        </div>

        <button
          type="button"
          onClick={addTestCase}
          className="w-full mt-3 py-2 border border-dashed border-background-border rounded-lg text-text-secondary hover:text-text-primary hover:border-accent-secondary transition-colors text-sm"
        >
          + Add Another Test Case
        </button>

        <div className="flex justify-end gap-3 pt-4">
          <button onClick={onClose} className="btn btn-secondary">
            Cancel
          </button>
          <button onClick={handleSubmit} disabled={loading} className="btn btn-primary">
            {loading
              ? 'Adding...'
              : `Add ${testCases.filter((tc) => tc.input && tc.expected_output).length} Test Cases`}
          </button>
        </div>
      </div>
    </div>
  );
}

export default SuperAdminProblems;
