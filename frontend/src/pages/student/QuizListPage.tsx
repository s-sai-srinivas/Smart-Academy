import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { studentQuizAPI } from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import { showError } from '../../utils/showAlert';
import { Clock, Calendar, BookOpen, ChevronRight, Trophy, GraduationCap } from 'lucide-react';

interface QuizItem {
  id: number;
  title: string;
  course_name?: string;
  course_code?: string;
  description?: string;
  status: string;
  time_limit_minutes?: number;
  total_marks: number;
  question_count?: number;
  attempts_used?: number;
  max_attempts?: number;
  scheduled_end?: string;
}

function QuizListPage() {
  const navigate = useNavigate();
  const { user: _user } = useAuth();
  const [loading, setLoading] = useState(true);
  const [quizzes, setQuizzes] = useState<QuizItem[]>([]);

  useEffect(() => {
    loadQuizzes();
  }, []);

  const loadQuizzes = async () => {
    try {
      const data = (await studentQuizAPI.listAvailable()) as { quizzes?: QuizItem[] };
      setQuizzes(data.quizzes || []);
    } catch (err) {
      console.error('Failed to load quizzes:', err);
      showError('Failed to load quizzes');
    } finally {
      setLoading(false);
    }
  };

  const getStatusBadge = (status: string) => {
    const styles: Record<string, string> = {
      published: 'bg-green-500/10 text-green-400 border-green-500/20',
      closed: 'bg-red-500/10 text-red-400 border-red-500/20',
      archived: 'bg-background-tertiary text-text-secondary border-background-border',
    };
    return (
      <span
        className={`px-2.5 py-1 rounded-md text-xs font-medium border ${styles[status] || styles.published}`}
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
      <div>
        <h1 className="text-3xl font-bold text-text-primary mb-2">Available Quizzes</h1>
        <p className="text-text-secondary">Take assigned quizzes and track your progress</p>
      </div>

      {quizzes.length === 0 ? (
        <div className="bg-background-secondary rounded-xl border-2 border-background-border p-12 text-center">
          <BookOpen className="w-16 h-16 mx-auto mb-4 text-text-muted" />
          <h3 className="text-xl font-semibold text-text-primary mb-2">No quizzes available</h3>
          <p className="text-text-muted">
            There are no quizzes available for your courses at the moment.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {quizzes.map((quiz) => (
            <div
              key={quiz.id}
              className="bg-background-secondary rounded-xl border-2 border-background-border p-6 hover:border-blue-500/50 transition-colors cursor-pointer"
              onClick={() => navigate(`/quizzes/${quiz.id}`)}
            >
              <div className="flex items-start justify-between mb-4">
                <div className="flex-1">
                  <h3 className="text-xl font-semibold text-text-primary mb-1">{quiz.title}</h3>
                  {quiz.course_name && (
                    <div className="flex items-center gap-1.5 text-blue-400 text-sm mb-2">
                      <GraduationCap className="w-3.5 h-3.5" />
                      <span>
                        {quiz.course_code} — {quiz.course_name}
                      </span>
                    </div>
                  )}
                  <p className="text-text-secondary text-sm line-clamp-2">{quiz.description}</p>
                </div>
                {getStatusBadge(quiz.status)}
              </div>

              <div className="grid grid-cols-2 gap-3 mb-4">
                <div className="flex items-center gap-2 text-text-secondary text-sm">
                  <Clock className="w-4 h-4" />
                  <span>
                    {quiz.time_limit_minutes || 'No limit'}
                    {quiz.time_limit_minutes ? ' min' : ''}
                  </span>
                </div>
                <div className="flex items-center gap-2 text-text-secondary text-sm">
                  <Trophy className="w-4 h-4" />
                  <span>{quiz.total_marks} marks</span>
                </div>
                <div className="flex items-center gap-2 text-text-secondary text-sm">
                  <BookOpen className="w-4 h-4" />
                  <span>{quiz.question_count || 0} questions</span>
                </div>
                <div className="flex items-center gap-2 text-text-secondary text-sm">
                  <Calendar className="w-4 h-4" />
                  <span>
                    {quiz.attempts_used !== undefined
                      ? `${quiz.attempts_used}/${quiz.max_attempts || '∞'} attempts`
                      : `${quiz.max_attempts || 'Unlimited'} attempts`}
                  </span>
                </div>
              </div>

              {quiz.scheduled_end && (
                <div className="mb-4 text-xs text-text-muted">
                  Ends: {new Date(quiz.scheduled_end).toLocaleString()}
                </div>
              )}

              <div className="flex items-center justify-between mt-4 pt-4 border-t border-background-border">
                <span className="text-text-muted text-sm">
                  {quiz.status === 'published' ? 'Click to start' : 'Quiz closed'}
                </span>
                <ChevronRight className="w-5 h-5 text-text-muted" />
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default QuizListPage;
