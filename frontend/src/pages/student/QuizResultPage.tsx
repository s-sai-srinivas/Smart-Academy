import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { studentQuizAPI } from '../../services/api';
import { showError } from '../../utils/showAlert';
import {
  ChevronLeft,
  CheckCircle,
  XCircle,
  Award,
  Clock,
  TrendingUp,
  TrendingDown,
} from 'lucide-react';

interface OptionResult {
  id: string | number;
  option_text: string;
  is_correct: boolean;
}

interface QuestionResult {
  question_id: number;
  is_correct: boolean;
  marks_awarded: number;
  question_text: string;
  question_type: string;
  options?: OptionResult[];
  user_selected_option_ids?: (string | number)[];
  user_text_answer?: string;
  expected_answer?: string;
  explanation?: string;
}

interface ResultData {
  attempt: {
    percentage: number;
    status: string;
    marks_obtained: number;
    attempt_number: number;
    submitted_at: string;
    time_taken_minutes?: number;
    total_marks?: number;
  };
  quiz: {
    title: string;
    total_marks: number;
    passing_marks: number;
    max_attempts?: number;
  };
  question_results?: QuestionResult[];
}

function QuizResultPage() {
  const { id: quizId } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [result, setResult] = useState<ResultData | null>(null);

  const loadResult = useCallback(async () => {
    if (!quizId) return;
    try {
      const data = (await studentQuizAPI.getResult(quizId)) as ResultData;
      setResult(data);
    } catch (err) {
      console.error('Failed to load result:', err);
      showError('Failed to load result');
    } finally {
      setLoading(false);
    }
  }, [quizId]);

  useEffect(() => {
    loadResult();
  }, [loadResult]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <div className="w-8 h-8 border-2 border-background-border border-t-gray-500 rounded-full animate-spin mx-auto mb-3" />
          <p className="text-text-muted">Loading results...</p>
        </div>
      </div>
    );
  }

  if (!result) {
    return (
      <div className="text-center py-12">
        <p className="text-text-muted">Failed to load results</p>
      </div>
    );
  }

  const { attempt, quiz, question_results } = result;
  const percentage = attempt.percentage;
  const isPassed = attempt.status === 'passed';
  const scoreColor =
    percentage >= 80
      ? 'text-green-400'
      : percentage >= 60
        ? 'text-blue-400'
        : percentage >= 40
          ? 'text-yellow-400'
          : 'text-red-400';

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <button
          onClick={() => navigate('/quizzes')}
          className="p-2 hover:bg-background-tertiary rounded-lg transition-colors"
        >
          <ChevronLeft className="w-5 h-5 text-text-secondary" />
        </button>
        <div>
          <h1 className="text-3xl font-bold text-text-primary mb-2">Quiz Results</h1>
          <p className="text-text-secondary">{quiz.title}</p>
        </div>
      </div>

      {/* Score Card */}
      <div
        className={`bg-background-secondary rounded-xl border-2 ${
          isPassed ? 'border-green-500/30' : 'border-red-500/30'
        } p-8 text-center`}
      >
        <div className="flex items-center justify-center gap-2 mb-4">
          {isPassed ? (
            <CheckCircle className="w-12 h-12 text-green-400" />
          ) : (
            <XCircle className="w-12 h-12 text-red-400" />
          )}
        </div>
        <h2 className="text-2xl font-bold text-text-primary mb-2">
          {isPassed ? 'Quiz Passed!' : 'Quiz Not Passed'}
        </h2>
        <p className="text-text-secondary mb-6">
          {isPassed
            ? 'Great job! You have successfully completed this quiz.'
            : 'You need to score at least 40% to pass.'}
        </p>

        <div className="flex items-center justify-center gap-8 mb-6">
          <div className="text-center">
            <div className={`text-5xl font-bold ${scoreColor} mb-2`}>
              {attempt.marks_obtained}/{quiz.total_marks}
            </div>
            <div className="text-text-secondary text-sm">Marks Obtained</div>
          </div>
          <div className="text-center">
            <div className={`text-5xl font-bold ${scoreColor} mb-2`}>{percentage}%</div>
            <div className="text-text-secondary text-sm">Percentage</div>
          </div>
          <div className="text-center">
            <div className="text-4xl font-bold text-text-primary mb-2">
              {Math.round(attempt.time_taken_minutes || 0)}
            </div>
            <div className="text-text-secondary text-sm">Minutes Taken</div>
          </div>
        </div>

        {/* Performance Indicator */}
        <div className="max-w-md mx-auto">
          <div className="flex items-center justify-between text-sm mb-2">
            <span className="text-text-secondary">Your Score</span>
            <span className="text-text-secondary">Passing: {quiz.passing_marks}</span>
          </div>
          <div className="h-4 bg-slate-700 rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full ${
                percentage >= 80
                  ? 'bg-green-500'
                  : percentage >= 60
                    ? 'bg-blue-500'
                    : percentage >= 40
                      ? 'bg-yellow-500'
                      : 'bg-red-500'
              }`}
              style={{ width: `${percentage}%` }}
            />
          </div>
          <div className="relative h-4 mt-1">
            <div
              className="absolute top-0 w-1 h-6 bg-white -mt-1"
              style={{ left: `${(quiz.passing_marks / quiz.total_marks) * 100}%` }}
            />
            <span
              className="absolute text-xs text-text-muted -mt-5"
              style={{ left: `${(quiz.passing_marks / quiz.total_marks) * 100}%` }}
            >
              Passing
            </span>
          </div>
        </div>
      </div>

      {/* Attempt Info */}
      <div className="grid grid-cols-3 gap-4">
        <div className="bg-background-secondary rounded-xl border-2 border-background-border p-4">
          <div className="flex items-center gap-2 text-text-secondary mb-2">
            <Award className="w-4 h-4" />
            <span className="text-sm">Attempt #</span>
          </div>
          <div className="text-2xl font-bold text-text-primary">{attempt.attempt_number}</div>
        </div>
        <div className="bg-background-secondary rounded-xl border-2 border-background-border p-4">
          <div className="flex items-center gap-2 text-text-secondary mb-2">
            <Clock className="w-4 h-4" />
            <span className="text-sm">Submitted At</span>
          </div>
          <div className="text-lg font-semibold text-text-primary">
            {new Date(attempt.submitted_at).toLocaleString()}
          </div>
        </div>
        <div className="bg-background-secondary rounded-xl border-2 border-background-border p-4">
          <div className="flex items-center gap-2 text-text-secondary mb-2">
            {attempt.marks_obtained >= (attempt.total_marks || quiz.total_marks) * 0.8 ? (
              <TrendingUp className="w-4 h-4 text-green-400" />
            ) : (
              <TrendingDown className="w-4 h-4 text-red-400" />
            )}
            <span className="text-sm">Correct Answers</span>
          </div>
          <div className="text-2xl font-bold text-text-primary">
            {(question_results || []).filter((q) => q.is_correct).length}/
            {(question_results || []).length}
          </div>
        </div>
      </div>

      {/* Question Results */}
      <div className="bg-background-secondary rounded-xl border-2 border-background-border p-6">
        <h3 className="text-xl font-semibold text-text-primary mb-4">Question Breakdown</h3>
        <div className="space-y-3">
          {question_results &&
            question_results.map((q, index) => (
              <div
                key={q.question_id}
                className={`p-4 rounded-lg border-2 ${
                  q.is_correct
                    ? 'border-green-500/30 bg-green-500/10'
                    : 'border-red-500/30 bg-red-500/10'
                }`}
              >
                <div className="flex items-start justify-between mb-2">
                  <div className="flex items-center gap-3">
                    <span className="text-sm text-text-secondary">Question {index + 1}</span>
                    {q.is_correct ? (
                      <CheckCircle className="w-4 h-4 text-green-400" />
                    ) : (
                      <XCircle className="w-4 h-4 text-red-400" />
                    )}
                  </div>
                  <span
                    className={`text-sm font-medium ${
                      q.is_correct ? 'text-green-400' : 'text-red-400'
                    }`}
                  >
                    {q.is_correct ? `+${q.marks_awarded} marks` : '0 marks'}
                  </span>
                </div>
                <p className="text-text-primary mb-3">{q.question_text}</p>

                {q.question_type === 'mcq' ||
                q.question_type === 'true_false' ||
                q.question_type === 'multi_select' ? (
                  <div className="space-y-2">
                    {q.options?.map((opt) => {
                      const isUserSelected = q.user_selected_option_ids?.includes(opt.id);
                      const isCorrect = opt.is_correct;
                      return (
                        <div
                          key={opt.id}
                          className={`flex items-center justify-between p-2 rounded ${
                            isCorrect && isUserSelected
                              ? 'bg-green-500/20 border border-green-500/50'
                              : isCorrect && !isUserSelected
                                ? 'bg-green-500/10 border border-green-500/30'
                                : isUserSelected && !isCorrect
                                  ? 'bg-red-500/20 border border-red-500/50'
                                  : 'bg-background-tertiary'
                          }`}
                        >
                          <span
                            className={`text-sm ${
                              isCorrect
                                ? 'text-green-400'
                                : isUserSelected
                                  ? 'text-red-400'
                                  : 'text-text-secondary'
                            }`}
                          >
                            {opt.option_text}
                          </span>
                          <div className="flex gap-2">
                            {isUserSelected && (
                              <span className="text-xs text-text-secondary">Your answer</span>
                            )}
                            {isCorrect && <CheckCircle className="w-4 h-4 text-green-400" />}
                          </div>
                        </div>
                      );
                    })}
                  </div>
                ) : (
                  <div className="space-y-2">
                    {q.user_text_answer && (
                      <div className="p-3 bg-background-tertiary rounded-lg">
                        <span className="text-xs text-text-secondary block mb-1">Your Answer:</span>
                        <p className="text-text-primary text-sm">{q.user_text_answer}</p>
                      </div>
                    )}
                    {q.expected_answer && (
                      <div className="p-3 bg-green-500/10 rounded-lg border border-green-500/20">
                        <span className="text-xs text-green-400 block mb-1">Expected Answer:</span>
                        <p className="text-green-400 text-sm">{q.expected_answer}</p>
                      </div>
                    )}
                  </div>
                )}

                {q.explanation && (
                  <div className="mt-3 pt-3 border-t border-slate-700">
                    <p className="text-sm text-text-secondary">
                      <span className="font-medium text-text-secondary">Explanation:</span>{' '}
                      {q.explanation}
                    </p>
                  </div>
                )}
              </div>
            ))}
        </div>
      </div>

      {/* Actions */}
      <div className="flex justify-center gap-4">
        <button
          onClick={() => navigate('/quizzes')}
          className="px-6 py-3 bg-slate-600 hover:bg-slate-500 text-white rounded-lg font-medium transition-colors"
        >
          Back to Quizzes
        </button>
        {!isPassed && quiz.max_attempts && attempt.attempt_number < quiz.max_attempts && (
          <button
            onClick={() => navigate(`/quizzes/${quizId}`)}
            className="px-6 py-3 bg-blue-500 hover:bg-blue-600 text-white rounded-lg font-medium transition-colors"
          >
            Retry Quiz ({quiz.max_attempts - attempt.attempt_number} attempts left)
          </button>
        )}
      </div>
    </div>
  );
}

export default QuizResultPage;
