import { useState, useEffect, useCallback, type ChangeEvent } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { quizAPI } from '../../services/api';
import { showSuccess, showError } from '../../utils/showAlert';
import { Plus, Edit2, Trash2, PlayCircle, BarChart3, Clock } from 'lucide-react';

interface Quiz {
  id: number;
  title: string;
  created_at: string;
  quiz_type: string;
  status: 'draft' | 'published' | 'closed' | 'archived';
  question_count?: number;
  time_limit_minutes?: number;
  max_attempts?: number | string;
}

interface Offering {
  course_offering_id: number;
  display_name: string;
}

function QuizManagementPage() {
  useParams<{ courseId: string }>();
  const navigate = useNavigate();
  const [loading, setLoading] = useState<boolean>(true);
  const [quizzes, setQuizzes] = useState<Quiz[]>([]);
  const [offerings, setOfferings] = useState<Offering[]>([]);
  const [selectedOffering, setSelectedOffering] = useState<string>('');

  const loadOfferings = useCallback(async () => {
    try {
      const data = (await quizAPI.getMyOfferings()) as { offerings?: Offering[] };
      setOfferings(data.offerings || []);
      if (data.offerings && data.offerings.length > 0) {
        setSelectedOffering(String(data.offerings[0].course_offering_id));
      }
    } catch (err) {
      console.error('Failed to load offerings:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  const loadQuizzes = useCallback(async () => {
    try {
      const data = (await quizAPI.listByOffering(selectedOffering)) as { quizzes?: Quiz[] };
      setQuizzes(data.quizzes || []);
    } catch (err) {
      console.error('Failed to load quizzes:', err);
    } finally {
      setLoading(false);
    }
  }, [selectedOffering]);

  useEffect(() => {
    loadOfferings();
  }, [loadOfferings]);

  useEffect(() => {
    if (selectedOffering) {
      loadQuizzes();
    }
  }, [selectedOffering, loadQuizzes]);

  const handleDelete = async (quizId: number) => {
    if (!confirm('Are you sure you want to delete this quiz?')) return;
    try {
      await quizAPI.delete(quizId);
      setQuizzes(quizzes.filter((q) => q.id !== quizId));
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      showError(`Failed to delete quiz: ${message}`);
    }
  };

  const handlePublish = async (quizId: number) => {
    try {
      await quizAPI.publish(quizId);
      setQuizzes(
        quizzes.map((q) => (q.id === quizId ? { ...q, status: 'published' as const } : q))
      );
      showSuccess('Quiz published successfully');
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      showError(`Failed to publish quiz: ${message}`);
    }
  };

  const handleClose = async (quizId: number) => {
    try {
      await quizAPI.close(quizId);
      setQuizzes(quizzes.map((q) => (q.id === quizId ? { ...q, status: 'closed' as const } : q)));
      showSuccess('Quiz closed successfully');
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      showError(`Failed to close quiz: ${message}`);
    }
  };

  const getStatusBadge = (status: string) => {
    const styles: Record<string, string> = {
      draft: 'bg-background-tertiary text-text-secondary border-background-border',
      published: 'bg-green-500/10 text-green-400 border-green-500/20',
      closed: 'bg-red-500/10 text-red-400 border-red-500/20',
      archived: 'bg-background-tertiary text-text-secondary border-background-border',
    };
    return (
      <span
        className={`px-2.5 py-1 rounded-md text-xs font-medium border ${styles[status] || styles.draft}`}
      >
        {status}
      </span>
    );
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <div className="w-8 h-8 border-2 border-background-border border-t-gray-500 rounded-full animate-spin mx-auto mb-3" />
          <p className="text-text-muted">Loading quizzes...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-text-primary mb-2">Quiz Management</h1>
          <p className="text-text-secondary">
            Create and manage AI-powered quizzes for your courses
          </p>
        </div>
        <button
          onClick={() => navigate(`/faculty/quizzes/create?offering=${selectedOffering}`)}
          className="flex items-center gap-2 px-4 py-2.5 bg-blue-500 hover:bg-blue-600 text-white rounded-lg font-medium transition-colors"
        >
          <Plus className="w-4 h-4" />
          Create Quiz
        </button>
      </div>

      {/* Course Offering Filter */}
      {offerings.length > 1 && (
        <div className="bg-background-secondary rounded-xl border-2 border-background-border p-4">
          <label className="block text-sm font-medium text-text-secondary mb-2">
            Select Course Offering
          </label>
          <select
            value={selectedOffering}
            onChange={(e: ChangeEvent<HTMLSelectElement>) => setSelectedOffering(e.target.value)}
            className="w-full max-w-md px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            {offerings.map((offering) => (
              <option key={offering.course_offering_id} value={offering.course_offering_id}>
                {offering.display_name}
              </option>
            ))}
          </select>
        </div>
      )}

      {/* Quizzes List */}
      <div className="bg-background-secondary rounded-xl border-2 border-background-border">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-background-border">
                <th className="text-left py-3 px-4 text-sm font-medium text-text-secondary">
                  Quiz
                </th>
                <th className="text-left py-3 px-4 text-sm font-medium text-text-secondary">
                  Type
                </th>
                <th className="text-left py-3 px-4 text-sm font-medium text-text-secondary">
                  Status
                </th>
                <th className="text-left py-3 px-4 text-sm font-medium text-text-secondary">
                  Questions
                </th>
                <th className="text-left py-3 px-4 text-sm font-medium text-text-secondary">
                  Duration
                </th>
                <th className="text-left py-3 px-4 text-sm font-medium text-text-secondary">
                  Attempts
                </th>
                <th className="text-left py-3 px-4 text-sm font-medium text-text-secondary">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody>
              {quizzes.length === 0 ? (
                <tr>
                  <td colSpan={7} className="py-12 text-center text-text-muted">
                    No quizzes created yet. Click "Create Quiz" to get started.
                  </td>
                </tr>
              ) : (
                quizzes.map((quiz) => (
                  <tr
                    key={quiz.id}
                    className="border-b border-background-border hover:bg-background-tertiary/50"
                  >
                    <td className="py-4 px-4">
                      <div>
                        <div className="font-medium text-text-primary">{quiz.title}</div>
                        <div className="text-sm text-text-muted">
                          Created {new Date(quiz.created_at).toLocaleDateString()}
                        </div>
                      </div>
                    </td>
                    <td className="py-4 px-4">
                      <span
                        className={`px-2.5 py-1 rounded-md text-xs font-medium border ${
                          quiz.quiz_type === 'ai_generated'
                            ? 'bg-purple-500/10 text-purple-400 border-purple-500/20'
                            : quiz.quiz_type === 'manual'
                              ? 'bg-blue-500/10 text-blue-400 border-blue-500/20'
                              : 'bg-indigo-500/10 text-indigo-400 border-indigo-500/20'
                        }`}
                      >
                        {quiz.quiz_type.replace('_', ' ')}
                      </span>
                    </td>
                    <td className="py-4 px-4">{getStatusBadge(quiz.status)}</td>
                    <td className="py-4 px-4 text-text-secondary">{quiz.question_count || 0}</td>
                    <td className="py-4 px-4 text-text-secondary">
                      {quiz.time_limit_minutes ? `${quiz.time_limit_minutes} min` : '-'}
                    </td>
                    <td className="py-4 px-4 text-text-secondary">
                      {quiz.max_attempts || 'Unlimited'}
                    </td>
                    <td className="py-4 px-4">
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => navigate(`/faculty/quizzes/${quiz.id}/edit`)}
                          className="p-1.5 hover:bg-blue-500/10 rounded-lg transition-colors"
                          title="Edit"
                        >
                          <Edit2 className="w-4 h-4 text-blue-400" />
                        </button>
                        <button
                          onClick={() => navigate(`/faculty/quizzes/${quiz.id}/analytics`)}
                          className="p-1.5 hover:bg-green-500/10 rounded-lg transition-colors"
                          title="Analytics"
                        >
                          <BarChart3 className="w-4 h-4 text-green-400" />
                        </button>
                        {quiz.status === 'draft' && (
                          <button
                            onClick={() => handlePublish(quiz.id)}
                            className="p-1.5 hover:bg-green-500/10 rounded-lg transition-colors"
                            title="Publish"
                          >
                            <PlayCircle className="w-4 h-4 text-green-400" />
                          </button>
                        )}
                        {quiz.status === 'published' && (
                          <button
                            onClick={() => handleClose(quiz.id)}
                            className="p-1.5 hover:bg-red-500/10 rounded-lg transition-colors"
                            title="Close"
                          >
                            <Clock className="w-4 h-4 text-red-400" />
                          </button>
                        )}
                        <button
                          onClick={() => handleDelete(quiz.id)}
                          className="p-1.5 hover:bg-red-500/10 rounded-lg transition-colors"
                          title="Delete"
                        >
                          <Trash2 className="w-4 h-4 text-red-400" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

export default QuizManagementPage;
