import { useState, useEffect, useRef, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { studentQuizAPI } from '../../services/api';
import { useNotification } from '../../context/NotificationContext';
import { Clock, ChevronLeft, AlertTriangle, CheckCircle } from 'lucide-react';

interface OptionData {
  id: string | number;
  option_text: string;
}

interface QuestionData {
  id: number;
  question_type: string;
  difficulty?: string;
  marks: number;
  question_text: string;
  options?: OptionData[];
}

interface QuizData {
  id: number;
  title: string;
  questions: QuestionData[];
}

interface AttemptData {
  id: number;
  time_remaining_seconds: number;
}

interface AnswerData {
  selected_option_ids?: (string | number)[];
  text_answer?: string;
}

function QuizTakingPage() {
  const { id: quizId } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const notification = useNotification();
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [quiz, setQuiz] = useState<QuizData | null>(null);
  const [attempt, setAttempt] = useState<AttemptData | null>(null);
  const [currentQuestionIndex, setCurrentQuestionIndex] = useState(0);
  const [answers, setAnswers] = useState<Record<number, AnswerData>>({});
  const [timeRemaining, setTimeRemaining] = useState<number | null>(null);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const handleAutoSubmit = useCallback(async () => {
    if (!quiz || !attempt) return;
    try {
      const formattedAnswers = quiz.questions.map((q) => {
        const answer = answers[q.id] || {};
        return {
          question_id: q.id,
          selected_option_ids: answer.selected_option_ids || [],
          text_answer: answer.text_answer || '',
        };
      });

      if (!quizId) return;
      await studentQuizAPI.submitAttempt(quizId, attempt.id, formattedAnswers);
      navigate(`/quizzes/${quizId}/result`);
    } catch (err) {
      console.error('Auto-submit failed:', err);
    }
  }, [quiz, attempt, answers, quizId, navigate]);

  const startTimer = useCallback(
    (_seconds: number) => {
      if (timerRef.current) clearInterval(timerRef.current);

      timerRef.current = setInterval(() => {
        setTimeRemaining((prev) => {
          if (prev === null || prev <= 1) {
            handleAutoSubmit();
            return 0;
          }
          return prev - 1;
        });
      }, 1000);
    },
    [handleAutoSubmit]
  );

  const loadQuiz = useCallback(async () => {
    if (!quizId) return;
    try {
      const data = (await studentQuizAPI.get(quizId)) as { quiz: QuizData };
      setQuiz(data.quiz);

      // Start attempt
      const attemptData = (await studentQuizAPI.startAttempt(quizId)) as { attempt: AttemptData };
      setAttempt(attemptData.attempt);
      setTimeRemaining(attemptData.attempt.time_remaining_seconds);
    } catch (err) {
      console.error('Failed to load quiz:', err);
      notification.error('Failed to load quiz: ' + (err as Error).message);
      navigate('/quizzes');
    } finally {
      setLoading(false);
    }
  }, [quizId, navigate, notification]);

  const stopTimer = useCallback(() => {
    if (timerRef.current) clearInterval(timerRef.current);
  }, []);

  useEffect(() => {
    loadQuiz();
    return () => stopTimer();
  }, [loadQuiz, stopTimer]);

  useEffect(() => {
    if (attempt?.time_remaining_seconds) {
      setTimeRemaining(attempt.time_remaining_seconds);
      startTimer(attempt.time_remaining_seconds);
    }
    return () => stopTimer();
  }, [attempt, startTimer, stopTimer]);

  const formatTime = (seconds: number | null) => {
    if (seconds === null) return '0:00';
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;

    if (hrs > 0) {
      return `${hrs}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    }
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const handleAnswerSelect = (questionId: number, answerData: AnswerData) => {
    setAnswers((prev) => ({
      ...prev,
      [questionId]: answerData,
    }));
  };

  const handleSubmit = async () => {
    if (!confirm('Are you sure you want to submit?')) return;
    if (!quiz || !attempt) return;

    setSubmitting(true);
    try {
      // Format answers for submission
      const formattedAnswers = quiz.questions.map((q) => {
        const answer = answers[q.id] || {};
        return {
          question_id: q.id,
          selected_option_ids: answer.selected_option_ids || [],
          text_answer: answer.text_answer || '',
        };
      });

      if (!quizId) return;
      await studentQuizAPI.submitAttempt(quizId, attempt.id, formattedAnswers);
      notification.success('Quiz submitted successfully!');
      navigate(`/quizzes/${quizId}/result`);
    } catch (err) {
      notification.error('Failed to submit quiz: ' + (err as Error).message);
    } finally {
      setSubmitting(false);
    }
  };

  if (loading || !quiz || !attempt) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <div className="w-8 h-8 border-2 border-background-border border-t-gray-500 rounded-full animate-spin mx-auto mb-3" />
          <p className="text-text-muted">Loading quiz...</p>
        </div>
      </div>
    );
  }

  const currentQuestion = quiz.questions[currentQuestionIndex];
  const progress = ((currentQuestionIndex + 1) / quiz.questions.length) * 100;
  const answeredCount = Object.keys(answers).length;
  const isTimeLow = timeRemaining !== null && timeRemaining < 60;

  return (
    <div className="max-w-4xl mx-auto space-y-4">
      {/* Header with Timer */}
      <div
        className="bg-background-secondary rounded-xl border-2 border-background-border p-4 sticky top-20 z-10"
        role="region"
        aria-label="Quiz progress"
      >
        <div className="flex items-center justify-between mb-3">
          <div>
            <h2 className="text-lg font-semibold text-text-primary">{quiz.title}</h2>
            <p className="text-sm text-text-secondary" aria-live="polite">
              Question {currentQuestionIndex + 1} of {quiz.questions.length}
            </p>
          </div>
          <div
            className={`flex items-center gap-2 px-4 py-2 rounded-lg ${
              isTimeLow ? 'bg-red-500/20 text-red-400' : 'bg-blue-500/10 text-blue-400'
            }`}
            role="timer"
            aria-label="Time remaining"
          >
            <Clock className="w-5 h-5" aria-hidden="true" />
            <span className="text-xl font-bold">{formatTime(timeRemaining)}</span>
          </div>
        </div>

        {/* Progress Bar */}
        <div className="flex items-center gap-3">
          <div
            className="flex-1 h-2 bg-slate-700 rounded-full overflow-hidden"
            role="progressbar"
            aria-valuenow={progress}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-label="Quiz progress"
          >
            <div
              className="h-full bg-blue-500 rounded-full transition-all"
              style={{ width: `${progress}%` }}
            />
          </div>
          <span className="text-sm text-text-secondary" aria-live="polite">
            {answeredCount}/{quiz.questions.length} answered
          </span>
        </div>

        {/* Navigation Buttons */}
        <div className="flex items-center justify-between mt-3">
          <button
            onClick={() => setCurrentQuestionIndex(Math.max(0, currentQuestionIndex - 1))}
            disabled={currentQuestionIndex === 0}
            className="px-4 py-2 bg-slate-600 hover:bg-slate-500 disabled:bg-slate-700 disabled:text-text-muted text-white rounded-lg transition-colors"
            aria-label="Go to previous question"
          >
            Previous
          </button>
          <div className="flex gap-2" role="group" aria-label="Question navigation">
            {quiz.questions.map((_, index) => (
              <button
                key={index}
                onClick={() => setCurrentQuestionIndex(index)}
                className={`w-8 h-8 rounded-lg font-medium text-sm transition-colors ${
                  index === currentQuestionIndex
                    ? 'bg-blue-500 text-white'
                    : answers[quiz.questions[index].id]
                      ? 'bg-green-500/20 text-green-400 border border-green-500/50'
                      : 'bg-slate-700 text-text-secondary hover:bg-slate-600'
                }`}
                aria-label={`Question ${index + 1}`}
                aria-current={index === currentQuestionIndex ? 'step' : undefined}
              >
                {index + 1}
              </button>
            ))}
          </div>
          {currentQuestionIndex < quiz.questions.length - 1 ? (
            <button
              onClick={() =>
                setCurrentQuestionIndex(
                  Math.min(quiz.questions.length - 1, currentQuestionIndex + 1)
                )
              }
              className="px-4 py-2 bg-slate-600 hover:bg-slate-500 text-white rounded-lg transition-colors"
              aria-label="Go to next question"
            >
              Next
            </button>
          ) : (
            <button
              onClick={handleSubmit}
              disabled={submitting}
              className="flex items-center gap-2 px-6 py-2 bg-green-500 hover:bg-green-600 disabled:bg-slate-600 text-white rounded-lg font-medium transition-colors"
              aria-label={submitting ? 'Submitting quiz' : 'Submit quiz'}
            >
              <CheckCircle className="w-4 h-4" />
              {submitting ? 'Submitting...' : 'Submit Quiz'}
            </button>
          )}
        </div>
      </div>

      {/* Time Warning */}
      {isTimeLow && (
        <div
          className="bg-red-500/10 border-2 border-red-500/20 rounded-xl p-4 flex items-center gap-3"
          role="alert"
          aria-live="assertive"
        >
          <AlertTriangle className="w-5 h-5 text-red-400 flex-shrink-0" aria-hidden="true" />
          <div>
            <p className="text-red-400 font-medium">Time is running low!</p>
            <p className="text-red-400/70 text-sm">You have less than 1 minute remaining</p>
          </div>
        </div>
      )}

      {/* Question */}
      <div className="bg-background-secondary rounded-xl border-2 border-background-border p-6">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-3">
            <span className="px-3 py-1 bg-blue-500/10 text-blue-400 rounded-md text-sm font-medium border border-blue-500/20">
              Question {currentQuestionIndex + 1}
            </span>
            <span className="px-2.5 py-1 bg-background-tertiary text-text-secondary rounded-md text-xs font-medium border border-background-border">
              {currentQuestion.question_type.replace('_', ' ')}
            </span>
            <span
              className={`px-2.5 py-1 rounded-md text-xs font-medium border ${
                currentQuestion.difficulty === 'easy'
                  ? 'bg-green-500/10 text-green-400 border-green-500/20'
                  : currentQuestion.difficulty === 'medium'
                    ? 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20'
                    : 'bg-red-500/10 text-red-400 border-red-500/20'
              }`}
            >
              {currentQuestion.difficulty}
            </span>
            <span className="px-2.5 py-1 bg-purple-500/10 text-purple-400 rounded-md text-xs font-medium border border-purple-500/20">
              {currentQuestion.marks} marks
            </span>
          </div>
          <p className="text-xl text-text-primary">{currentQuestion.question_text}</p>
        </div>

        {/* Answer Options */}
        {['mcq', 'true_false', 'multi_select'].includes(currentQuestion.question_type) && (
          <div className="space-y-3" role="group" aria-label="Answer options">
            {currentQuestion.options?.map((option) => {
              const isSelected = (answers[currentQuestion.id]?.selected_option_ids || []).includes(
                option.id
              );
              return (
                <label
                  key={option.id}
                  className={`flex items-center gap-4 p-4 rounded-xl border-2 cursor-pointer transition-all ${
                    isSelected
                      ? 'border-blue-500 bg-blue-500/10'
                      : 'border-background-border bg-background-tertiary hover:border-background-elevated'
                  }`}
                >
                  <input
                    type={
                      currentQuestion.question_type === 'mcq' ||
                      currentQuestion.question_type === 'true_false'
                        ? 'radio'
                        : 'checkbox'
                    }
                    name={`question-${currentQuestion.id}`}
                    checked={isSelected}
                    onChange={() => {
                      if (
                        currentQuestion.question_type === 'mcq' ||
                        currentQuestion.question_type === 'true_false'
                      ) {
                        handleAnswerSelect(currentQuestion.id, {
                          selected_option_ids: [option.id],
                        });
                      } else {
                        const current = answers[currentQuestion.id]?.selected_option_ids || [];
                        const updated = current.includes(option.id)
                          ? current.filter((id) => id !== option.id)
                          : [...current, option.id];
                        handleAnswerSelect(currentQuestion.id, { selected_option_ids: updated });
                      }
                    }}
                    className="w-5 h-5 text-blue-500 rounded focus:ring-blue-500"
                    aria-checked={isSelected}
                  />
                  <span className="text-text-primary flex-1">{option.option_text}</span>
                </label>
              );
            })}
          </div>
        )}

        {(currentQuestion.question_type === 'short_answer' ||
          currentQuestion.question_type === 'fill_blank') && (
          <div>
            <label
              className="block text-sm font-medium text-text-secondary mb-2"
              htmlFor="text-answer"
            >
              Your Answer
            </label>
            <textarea
              id="text-answer"
              value={answers[currentQuestion.id]?.text_answer || ''}
              onChange={(e) =>
                handleAnswerSelect(currentQuestion.id, { text_answer: e.target.value })
              }
              className="w-full px-4 py-3 bg-background-tertiary border border-background-border rounded-xl text-text-primary focus:outline-none focus:ring-2 focus:ring-blue-500"
              rows={6}
              placeholder="Type your answer here..."
              aria-label="Your text answer"
            />
          </div>
        )}
      </div>

      {/* Flag for Review (optional feature placeholder) */}
      <div className="flex items-center justify-between px-4">
        <button
          onClick={() => navigate('/quizzes')}
          className="flex items-center gap-2 text-text-secondary hover:text-text-primary transition-colors"
        >
          <ChevronLeft className="w-4 h-4" />
          Quit Quiz
        </button>
        <p className="text-text-muted text-sm">
          Make sure to review your answers before submitting
        </p>
      </div>
    </div>
  );
}

export default QuizTakingPage;
