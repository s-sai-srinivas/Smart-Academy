import { useState, useEffect, useCallback } from 'react';
import { coursesAPI } from '../../services/api';
import Breadcrumb from '../../components/common/Breadcrumb';

interface ModuleItem {
  id: number;
  module_name?: string;
  title?: string;
}

interface Week {
  id: number;
  week_name: string;
  week_order?: number;
  modules?: ModuleItem[];
  has_quiz?: boolean;
}

interface QuizOption {
  id: string | number;
  option_text: string;
  is_correct: boolean;
}

interface QuizQuestion {
  id: number;
  question_text: string;
  difficulty?: string;
  marks?: number;
  explanation?: string;
  options?: QuizOption[];
}

interface QuizData {
  title?: string;
  instructions?: string;
  questions?: QuizQuestion[];
}

interface ScoreData {
  correct: number;
  total: number;
  obtained: number;
  totalMarks: number;
  percentage: number;
}

interface StudentTheoryQuizPageProps {
  course: { course_name: string };
  courseId: string;
  week: Week;
  theoryWeeks?: Week[];
  onBack: () => void;
  onNavigateModule: (week: Week, module: ModuleItem) => void;
}

/**
 * StudentTheoryQuizPage
 * Renders the practice quiz for a theory week.
 * Features a collapsible sidebar showing all weeks/modules for easy navigation.
 */
function StudentTheoryQuizPage({
  course,
  courseId: _courseId,
  week,
  theoryWeeks,
  onBack,
  onNavigateModule,
}: StudentTheoryQuizPageProps) {
  const [loading, setLoading] = useState(true);
  const [quiz, setQuiz] = useState<QuizData | null>(null);
  const [selectedAnswers, setSelectedAnswers] = useState<Record<number, string | number>>({});
  const [attempted, setAttempted] = useState(false);
  const [score, setScore] = useState<ScoreData | null>(null);
  const [showResult, setShowResult] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  const fetchQuiz = useCallback(async () => {
    setLoading(true);
    try {
      const data = (await coursesAPI.getPracticeQuiz(week.id)) as QuizData;
      setQuiz(data);
    } catch (err) {
      if ((err as { status?: number }).status === 404) {
        setQuiz(null);
      } else {
        console.error('Failed to load quiz:', err);
        setQuiz(null);
      }
    } finally {
      setLoading(false);
    }
  }, [week.id]);

  useEffect(() => {
    fetchQuiz();
  }, [fetchQuiz]);

  const handleSelect = (questionId: number, optionId: string | number) => {
    if (attempted) return;
    setSelectedAnswers((prev) => ({ ...prev, [questionId]: optionId }));
  };

  const handleSubmit = () => {
    if (!quiz || !quiz.questions) return;
    let correct = 0;
    let totalMarks = 0;
    let obtained = 0;

    quiz.questions.forEach((q) => {
      const correct_opt = q.options?.find((o) => o.is_correct);
      if (correct_opt) {
        totalMarks += q.marks || 1;
        if (selectedAnswers[q.id] === correct_opt.id) {
          correct++;
          obtained += q.marks || 1;
        }
      }
    });

    setScore({
      correct,
      total: quiz.questions.length,
      obtained,
      totalMarks,
      percentage: Math.round((correct / quiz.questions.length) * 100),
    });
    setAttempted(true);
    setShowResult(true);
  };

  const handleRetry = () => {
    setSelectedAnswers({});
    setAttempted(false);
    setScore(null);
    setShowResult(false);
  };

  const getDifficultyClass = (d?: string) => {
    if (d === 'easy') return 'badge-success';
    if (d === 'medium') return 'badge-warning';
    if (d === 'hard') return 'badge-danger';
    return 'badge-neutral';
  };

  return (
    <div className="flex min-h-screen">
      {/* Collapsible Sidebar */}
      <div
        className={`${sidebarCollapsed ? 'w-12' : 'w-72'} flex-shrink-0 border-r border-border-light bg-bg-primary transition-all duration-300`}
      >
        {/* Sidebar Header */}
        <div className="flex items-center justify-between p-3 border-b border-border-light">
          {!sidebarCollapsed && (
            <span className="text-sm font-semibold text-text-primary truncate">
              {course.course_name}
            </span>
          )}
          <button
            onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
            className="p-1.5 rounded hover:bg-bg-secondary text-text-muted hover:text-text-primary transition-colors"
            title={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            {sidebarCollapsed ? (
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 5l7 7-7 7"
                />
              </svg>
            ) : (
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M15 19l-7-7 7-7"
                />
              </svg>
            )}
          </button>
        </div>

        {/* Weeks/Modules List */}
        {!sidebarCollapsed && (
          <div className="overflow-y-auto max-h-[calc(100vh-60px)]">
            {theoryWeeks?.map((w, wIndex) => (
              <div key={w.id} className="border-b border-border-light/50">
                {/* Week Header */}
                <div
                  className={`px-3 py-2 text-xs font-semibold uppercase tracking-wide
                  ${w.id === week.id ? 'text-accent-secondary bg-accent-secondary/5' : 'text-text-muted bg-bg-secondary'}`}
                >
                  Week {w.week_order || wIndex + 1}: {w.week_name}
                </div>
                {/* Modules */}
                {w.modules?.map((m) => (
                  <button
                    key={m.id}
                    onClick={() => onNavigateModule(w, m)}
                    className="w-full text-left px-3 py-2 text-sm text-text-secondary hover:bg-bg-secondary hover:text-text-primary transition-colors flex items-center gap-2 border-l-2 border-transparent"
                  >
                    <span className="text-xs">📄</span>
                    <span className="truncate">{m.module_name || m.title}</span>
                  </button>
                ))}
                {/* Quiz Link */}
                <button
                  onClick={() => {}}
                  disabled={w.id === week.id}
                  className={`w-full text-left px-3 py-2 text-sm transition-colors flex items-center gap-2 border-l-2
                    ${
                      w.id === week.id
                        ? 'bg-accent-secondary/10 text-accent-secondary font-medium border-accent-secondary'
                        : 'text-text-muted hover:text-accent-secondary hover:bg-bg-secondary border-transparent'
                    }`}
                >
                  <span className="text-xs">📝</span>
                  <span className="truncate">Practice Quiz</span>
                </button>
              </div>
            ))}
          </div>
        )}

        {/* Collapsed view - just icons */}
        {sidebarCollapsed && (
          <div className="flex flex-col items-center py-2 gap-1">
            {theoryWeeks?.map((w, wIndex) => (
              <div key={w.id} className="relative group">
                <div
                  className={`w-8 h-8 rounded flex items-center justify-center text-xs font-medium
                  ${w.id === week.id ? 'bg-accent-secondary text-white' : 'bg-bg-secondary text-text-muted hover:text-text-primary'}`}
                >
                  {w.week_order || wIndex + 1}
                </div>
                {/* Tooltip */}
                <div className="absolute left-full ml-2 px-2 py-1 bg-bg-elevated border border-border-light rounded text-xs text-text-primary whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity z-10 pointer-events-none">
                  Week {w.week_order || wIndex + 1}: {w.week_name}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Main Content */}
      <div className="flex-1 overflow-y-auto">
        <div className="p-6 max-w-4xl mx-auto">
          {/* Breadcrumb */}
          <Breadcrumb
            items={[
              { label: 'Dashboard', to: '/dashboard' },
              { label: 'Courses', to: '/courses' },
              { label: course.course_name, onClick: onBack },
              { label: week.week_name, onClick: onBack },
              { label: 'Practice Quiz' },
            ]}
          />

          {/* Back button */}
          <button
            onClick={onBack}
            className="mb-6 flex items-center gap-2 text-text-secondary hover:text-text-primary transition-colors text-sm"
          >
            ← Back to {course.course_name}
          </button>

          {loading ? (
            <div className="card flex items-center justify-center py-20">
              <div className="text-center">
                <div className="w-8 h-8 border-2 border-accent-secondary/30 border-t-accent-secondary rounded-full animate-spin mx-auto mb-3" />
                <p className="text-text-muted">Loading quiz...</p>
              </div>
            </div>
          ) : !quiz ? (
            <div className="card text-center py-16">
              <div className="text-5xl mb-4">📝</div>
              <h3 className="text-lg font-semibold text-text-primary mb-2">No Quiz Available</h3>
              <p className="text-text-secondary text-sm">
                The practice quiz for this week is being prepared. Check back later!
              </p>
            </div>
          ) : (
            <>
              {/* Quiz Header */}
              <div className="card mb-6">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <div className="text-xs font-medium text-accent-secondary uppercase tracking-wide mb-1">
                      Week {week.week_order}: {week.week_name}
                    </div>
                    <h1 className="text-2xl font-bold text-text-primary mb-1">
                      {quiz.title || 'Practice Quiz'}
                    </h1>
                    {quiz.instructions && (
                      <p className="text-text-secondary text-sm mt-2">{quiz.instructions}</p>
                    )}
                  </div>
                  <div className="flex gap-4 text-sm text-text-secondary flex-shrink-0">
                    <div className="text-center">
                      <div className="text-2xl font-bold text-text-primary">
                        {quiz.questions?.length ?? 0}
                      </div>
                      <div className="text-xs">Questions</div>
                    </div>
                    <div className="text-center">
                      <div className="text-2xl font-bold text-text-primary">
                        {quiz.questions?.reduce((s, q) => s + (q.marks || 1), 0) ?? 0}
                      </div>
                      <div className="text-xs">Total Marks</div>
                    </div>
                  </div>
                </div>

                {/* Score result banner */}
                {showResult && score && (
                  <div
                    className={`mt-4 p-4 rounded-lg border flex items-center justify-between
                    ${
                      score.percentage >= 60
                        ? 'bg-accent-success/10 border-accent-success/30'
                        : 'bg-accent-warning/10 border-accent-warning/30'
                    }`}
                  >
                    <div>
                      <div className="font-semibold text-text-primary">
                        {score.percentage >= 80
                          ? '🎉 Excellent!'
                          : score.percentage >= 60
                            ? '👍 Good Job!'
                            : '📖 Keep Practicing!'}
                      </div>
                      <div className="text-sm text-text-secondary">
                        {score.correct}/{score.total} correct · {score.obtained}/{score.totalMarks}{' '}
                        marks
                      </div>
                    </div>
                    <div
                      className={`text-3xl font-bold ${score.percentage >= 60 ? 'text-accent-success' : 'text-accent-warning'}`}
                    >
                      {score.percentage}%
                    </div>
                  </div>
                )}
              </div>

              {/* Questions */}
              <div className="space-y-5 mb-6">
                {quiz.questions?.map((question, qIndex) => {
                  const userSelected = selectedAnswers[question.id];
                  return (
                    <div key={question.id} className="card">
                      <div className="flex items-center gap-3 mb-3">
                        <span className="text-xs font-semibold text-accent-secondary">
                          Q{qIndex + 1}
                        </span>
                        {question.difficulty && (
                          <span
                            className={`badge ${getDifficultyClass(question.difficulty)} text-xs`}
                          >
                            {question.difficulty}
                          </span>
                        )}
                        <span className="ml-auto text-xs text-text-tertiary">
                          {question.marks || 1} {(question.marks || 1) === 1 ? 'mark' : 'marks'}
                        </span>
                      </div>

                      <p className="text-text-primary text-sm leading-relaxed mb-4 font-medium">
                        {question.question_text}
                      </p>

                      <div className="space-y-2">
                        {question.options?.map((option, oIdx) => {
                          const isSelected = userSelected === option.id;
                          const isCorrect = option.is_correct;
                          let cls =
                            'flex items-center gap-3 p-3 rounded-lg border text-sm cursor-pointer transition-all ';

                          if (!attempted) {
                            cls += isSelected
                              ? 'border-accent-secondary bg-accent-secondary/10 text-text-primary'
                              : 'border-background-border bg-background-tertiary text-text-secondary hover:border-accent-secondary/40 hover:bg-background-elevated';
                          } else {
                            if (isCorrect) {
                              cls +=
                                'border-accent-success bg-accent-success/10 text-accent-success';
                            } else if (isSelected && !isCorrect) {
                              cls += 'border-accent-danger bg-accent-danger/10 text-accent-danger';
                            } else {
                              cls +=
                                'border-background-border bg-background-tertiary text-text-tertiary opacity-60';
                            }
                          }

                          return (
                            <div
                              key={option.id}
                              className={cls}
                              onClick={() => handleSelect(question.id, option.id)}
                            >
                              <div
                                className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold flex-shrink-0
                                ${!attempted && isSelected ? 'bg-accent-secondary text-white' : ''}
                                ${attempted && isCorrect ? 'bg-accent-success text-white' : ''}
                                ${attempted && isSelected && !isCorrect ? 'bg-accent-danger text-white' : ''}
                                ${!((!attempted && isSelected) || (attempted && isCorrect) || (attempted && isSelected && !isCorrect)) ? 'bg-background-border text-text-tertiary' : ''}
                              `}
                              >
                                {String.fromCharCode(65 + oIdx)}
                              </div>
                              <span className="flex-1">{option.option_text}</span>
                              {attempted && isCorrect && <span className="flex-shrink-0">✓</span>}
                              {attempted && isSelected && !isCorrect && (
                                <span className="flex-shrink-0">✗</span>
                              )}
                            </div>
                          );
                        })}
                      </div>

                      {/* Explanation (shown after attempt) */}
                      {attempted && question.explanation && (
                        <div className="mt-3 p-3 rounded-lg bg-accent-secondary/5 border border-accent-secondary/20 text-sm">
                          <span className="font-semibold text-accent-secondary">
                            💡 Explanation:{' '}
                          </span>
                          <span className="text-text-secondary">{question.explanation}</span>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>

              {/* Actions */}
              <div className="flex items-center justify-center gap-4">
                {!attempted ? (
                  <button
                    onClick={handleSubmit}
                    disabled={Object.keys(selectedAnswers).length === 0}
                    className="btn btn-primary px-8 py-2.5 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Submit Quiz
                  </button>
                ) : (
                  <button onClick={handleRetry} className="btn btn-secondary px-8 py-2.5">
                    🔄 Retry Quiz
                  </button>
                )}
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

export default StudentTheoryQuizPage;
