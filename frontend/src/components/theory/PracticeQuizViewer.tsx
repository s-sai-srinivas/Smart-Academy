import { useState, useEffect, useCallback } from 'react';
import { lessonPlanAPI } from '../../services/api';
import Swal from 'sweetalert2';
import type { SweetAlertOptions } from 'sweetalert2';

export interface QuizOption {
  id: number;
  option_text: string;
  is_correct: boolean;
}

export interface QuizQuestion {
  id: number;
  question_text: string;
  options: QuizOption[];
  difficulty: string;
  marks: number;
  explanation?: string;
}

export interface QuizData {
  title: string;
  instructions?: string;
  questions: QuizQuestion[];
  total_marks?: number;
  passing_marks?: number;
}

export interface ScoreResult {
  correct: number;
  total: number;
  obtainedMarks: number;
  totalMarks: number;
  percentage: number;
}

export interface PracticeQuizViewerProps {
  weekId: number;
}

const PracticeQuizViewer = ({ weekId }: PracticeQuizViewerProps) => {
  const [loading, setLoading] = useState(true);
  const [quiz, setQuiz] = useState<QuizData | null>(null);
  const [selectedAnswers, setSelectedAnswers] = useState<Record<number, number>>({});
  const [showExplanation, setShowExplanation] = useState<Record<number, boolean>>({});
  const [score, setScore] = useState<ScoreResult | null>(null);
  const [attempted, setAttempted] = useState(false);

  const fetchPracticeQuiz = useCallback(async () => {
    try {
      const data = await lessonPlanAPI.getPracticeQuiz(weekId);
      setQuiz(data as QuizData);
    } catch (error) {
      console.error('Error fetching practice quiz:', error);
      const err = error as { status?: number };
      if (err.status === 404) {
        setQuiz(null);
      } else {
        Swal.fire({
          icon: 'error',
          title: 'Error',
          text: 'Failed to load practice quiz',
        } as SweetAlertOptions);
      }
    } finally {
      setLoading(false);
    }
  }, [weekId]);

  useEffect(() => {
    fetchPracticeQuiz();
  }, [fetchPracticeQuiz]);

  const handleOptionSelect = (questionId: number, optionId: number) => {
    if (attempted) return;
    setSelectedAnswers((prev) => ({ ...prev, [questionId]: optionId }));
  };

  const handleSubmit = () => {
    if (!quiz) return;
    const answeredCount = Object.keys(selectedAnswers).length;
    if (answeredCount < quiz.questions.length) {
      Swal.fire({
        icon: 'warning',
        title: 'Incomplete Quiz',
        text: `You have answered ${answeredCount} out of ${quiz.questions.length} questions. Please answer all questions before submitting.`,
        showCancelButton: true,
        confirmButtonText: 'Submit Anyway',
        cancelButtonText: 'Continue Answering',
      } as SweetAlertOptions).then((result) => {
        if (result.isConfirmed) calculateScore();
      });
    } else {
      calculateScore();
    }
  };

  const calculateScore = () => {
    if (!quiz) return;
    let correctCount = 0;
    let totalMarks = 0;
    let obtainedMarks = 0;
    quiz.questions.forEach((question) => {
      const selectedOptionId = selectedAnswers[question.id];
      const correctOption = question.options.find((opt) => opt.is_correct);
      if (correctOption) {
        totalMarks += question.marks;
        if (selectedOptionId === correctOption.id) {
          obtainedMarks += question.marks;
          correctCount++;
        }
      }
    });
    const percentage = Math.round((correctCount / quiz.questions.length) * 100);
    setScore({
      correct: correctCount,
      total: quiz.questions.length,
      obtainedMarks,
      totalMarks,
      percentage,
    });
    setAttempted(true);
    const allExplanations: Record<number, boolean> = {};
    quiz.questions.forEach((q) => {
      allExplanations[q.id] = true;
    });
    setShowExplanation(allExplanations);

    let icon: SweetAlertOptions['icon'] = 'info';
    let title = 'Quiz Completed!';
    if (percentage >= 80) {
      icon = 'success';
      title = 'Excellent Work!';
    } else if (percentage >= 60) {
      icon = 'success';
      title = 'Good Job!';
    } else if (percentage >= 40) {
      icon = 'warning';
      title = 'Keep Practicing!';
    } else {
      icon = 'error';
      title = 'Needs Improvement';
    }

    Swal.fire({
      icon,
      title,
      html: `<div style="text-align: left;"><p><strong>Score:</strong> ${correctCount}/${quiz.questions.length} (${percentage}%)</p><p><strong>Marks:</strong> ${obtainedMarks}/${totalMarks}</p></div>`,
      confirmButtonText: 'Review Answers',
    } as SweetAlertOptions);
  };

  const handleRetry = () => {
    setSelectedAnswers({});
    setShowExplanation({});
    setScore(null);
    setAttempted(false);
  };

  const getDifficultyColor = (difficulty: string): string => {
    switch (difficulty) {
      case 'easy':
        return '#28a745';
      case 'medium':
        return '#ffc107';
      case 'hard':
        return '#dc3545';
      default:
        return '#6c757d';
    }
  };

  if (loading) {
    return (
      <div className="practice-quiz-viewer loading">
        <div className="spinner"></div>
        <p>Loading practice quiz...</p>
      </div>
    );
  }
  if (!quiz) {
    return (
      <div className="practice-quiz-viewer no-quiz">
        <div className="no-quiz-icon">📝</div>
        <h3>No Practice Quiz Available</h3>
        <p>This week's practice quiz is being prepared. Check back later!</p>
      </div>
    );
  }

  return (
    <div className="practice-quiz-viewer">
      <div className="quiz-header">
        <div className="quiz-title-section">
          <span className="quiz-icon">📝</span>
          <h3>{quiz.title}</h3>
        </div>
        {score && (
          <div className="quiz-score-badge">
            <span className="score-value">{score.percentage}%</span>
            <span className="score-label">Score</span>
          </div>
        )}
      </div>
      {quiz.instructions && (
        <div className="quiz-instructions">
          <h4>Instructions:</h4>
          <p>{quiz.instructions}</p>
          <div className="quiz-meta">
            <span className="meta-item">
              <strong>{quiz.questions.length}</strong> Questions
            </span>
            <span className="meta-item">
              <strong>{quiz.total_marks || quiz.questions.length}</strong> Total Marks
            </span>
            <span className="meta-item">
              <strong>{quiz.passing_marks || Math.ceil(quiz.questions.length * 0.6)}</strong>{' '}
              Passing Marks
            </span>
          </div>
        </div>
      )}
      <div className="quiz-questions">
        {quiz.questions.map((question, qIndex) => {
          const userSelected = selectedAnswers[question.id];
          const isShowingExplanation = showExplanation[question.id];
          return (
            <div key={question.id} className={`question-card ${attempted ? 'attempted' : ''}`}>
              <div className="question-header">
                <span className="question-number">Question {qIndex + 1}</span>
                <span
                  className="difficulty-badge"
                  style={{ backgroundColor: getDifficultyColor(question.difficulty) }}
                >
                  {question.difficulty}
                </span>
                <span className="question-marks">
                  +{question.marks} {question.marks > 1 ? 'marks' : 'mark'}
                </span>
              </div>
              <p className="question-text">{question.question_text}</p>
              <div className="options-list">
                {question.options.map((option, oIndex) => {
                  const isSelected = userSelected === option.id;
                  const isCorrectOption = option.is_correct;
                  let optionClass = 'option-item';
                  if (attempted) {
                    if (isCorrectOption) optionClass += ' correct';
                    else if (isSelected && !isCorrectOption) optionClass += ' incorrect';
                  } else if (isSelected) optionClass += ' selected';
                  return (
                    <div
                      key={option.id}
                      className={optionClass}
                      onClick={() => handleOptionSelect(question.id, option.id)}
                    >
                      <span className="option-letter">{String.fromCharCode(65 + oIndex)}</span>
                      <span className="option-text">{option.option_text}</span>
                      {attempted && isCorrectOption && <span className="option-icon">✓</span>}
                      {attempted && isSelected && !isCorrectOption && (
                        <span className="option-icon">✗</span>
                      )}
                    </div>
                  );
                })}
              </div>
              {isShowingExplanation && (
                <div className="explanation-box">
                  <h5>💡 Explanation:</h5>
                  <p>{question.explanation}</p>
                </div>
              )}
            </div>
          );
        })}
      </div>
      <div className="quiz-actions">
        {!attempted ? (
          <button
            className="btn btn-primary btn-large"
            onClick={handleSubmit}
            disabled={Object.keys(selectedAnswers).length === 0}
          >
            <span>✅</span>Submit Quiz
          </button>
        ) : (
          <div className="result-actions">
            <button className="btn btn-secondary btn-large" onClick={handleRetry}>
              <span>🔄</span>Retry Quiz
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

export default PracticeQuizViewer;
