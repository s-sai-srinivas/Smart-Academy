import { useState, useEffect, useCallback } from 'react';
import Modal from '../common/Modal';
import { contestsAPI } from '../../services/api';
import { showError, showSuccess } from '../../utils/showAlert';

interface ContestQuizOption {
  id?: number;
  option_text: string;
  is_correct: boolean;
  order_index?: number;
}

interface ContestQuizQuestion {
  id?: number;
  question_type: 'MCQ' | 'TF';
  question_text: string;
  explanation?: string;
  marks: number;
  order_index?: number;
  options: ContestQuizOption[];
}

interface ContestQuiz {
  id?: number;
  title: string;
  instructions: string;
  total_marks: number;
  passing_marks: number;
  duration_minutes: number;
  questions: ContestQuizQuestion[];
}

interface ContestQuizModalProps {
  contestId: number;
  sectionId: number;
  sectionName: string;
  onClose: () => void;
  onSuccess: () => void;
}

function emptyQuiz(): ContestQuiz {
  return {
    title: '',
    instructions: '',
    total_marks: 0,
    passing_marks: 0,
    duration_minutes: 0,
    questions: [],
  };
}

function emptyQuestion(): ContestQuizQuestion {
  return {
    question_type: 'MCQ',
    question_text: '',
    explanation: '',
    marks: 1,
    options: [
      { option_text: '', is_correct: false },
      { option_text: '', is_correct: false },
    ],
  };
}

function toNum(v: string): number {
  const n = parseInt(v.replace(/\D/g, ''), 10);
  return isNaN(n) ? 0 : n;
}

function getErrorMessage(err: unknown): string {
  if (err instanceof Error) return err.message;
  if (typeof err === 'object' && err && 'message' in err) {
    const message = (err as { message?: unknown }).message;
    if (typeof message === 'string') return message;
  }
  return 'Unknown error';
}

function getErrorStatus(err: unknown): number | undefined {
  if (typeof err === 'object' && err && 'status' in err) {
    const status = (err as { status?: unknown }).status;
    if (typeof status === 'number') return status;
  }
  return undefined;
}

export default function ContestQuizModal({
  contestId,
  sectionId,
  sectionName,
  onClose,
  onSuccess,
}: ContestQuizModalProps) {
  const [quiz, setQuiz] = useState<ContestQuiz>(emptyQuiz());
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [editingQuestionIdx, setEditingQuestionIdx] = useState<number | null>(null);

  const loadQuiz = useCallback(async () => {
    setLoading(true);
    try {
      const data = (await contestsAPI.getContestQuiz(contestId, sectionId)) as {
        quiz?: ContestQuiz;
      };
      if (data?.quiz) {
        setQuiz(data.quiz);
      }
    } catch (err: unknown) {
      if (getErrorStatus(err) !== 404) {
        console.error('Error loading quiz:', err);
      }
    } finally {
      setLoading(false);
    }
  }, [contestId, sectionId]);

  useEffect(() => {
    loadQuiz();
  }, [loadQuiz]);

  const handleSaveQuiz = async () => {
    setError('');
    if (quiz.questions.length === 0) {
      setError('At least one question is required');
      return;
    }

    // Validate questions
    for (let i = 0; i < quiz.questions.length; i++) {
      const q = quiz.questions[i];
      if (!q.question_text.trim()) {
        setError(`Question ${i + 1} text is required`);
        return;
      }
      if (q.options.length < 2) {
        setError(`Question ${i + 1} must have at least 2 options`);
        return;
      }
      const hasCorrect = q.options.some((o) => o.is_correct);
      if (!hasCorrect) {
        setError(`Question ${i + 1} must have at least one correct answer`);
        return;
      }
    }

    // Auto-calculate total marks from question count (each = 1)
    const totalMarks = quiz.questions.length;
    const payload = {
      section_id: sectionId,
      title: sectionName,
      instructions: '',
      total_marks: totalMarks,
      passing_marks: 0,
      duration_minutes: quiz.duration_minutes,
      questions: quiz.questions.map((q, i) => ({
        ...q,
        marks: 1,
        order_index: i,
        options: q.options.map((o, j) => ({ ...o, order_index: j })),
      })),
    };

    setSaving(true);
    try {
      await contestsAPI.upsertContestQuiz(contestId, payload);
      showSuccess('Quiz saved successfully');
      onSuccess();
    } catch (err: unknown) {
      const message = getErrorMessage(err) || 'Failed to save quiz';
      console.error('Error saving quiz:', err);
      setError(message);
      showError(message);
    } finally {
      setSaving(false);
    }
  };

  const addQuestion = () => {
    setQuiz((prev) => ({
      ...prev,
      questions: [...prev.questions, emptyQuestion()],
    }));
    setEditingQuestionIdx(quiz.questions.length);
  };

  const updateQuestion = (idx: number, updates: Partial<ContestQuizQuestion>) => {
    setQuiz((prev) => ({
      ...prev,
      questions: prev.questions.map((q, i) => (i === idx ? { ...q, ...updates } : q)),
    }));
  };

  const removeQuestion = (idx: number) => {
    setQuiz((prev) => ({
      ...prev,
      questions: prev.questions.filter((_, i) => i !== idx),
    }));
    if (editingQuestionIdx === idx) setEditingQuestionIdx(null);
  };

  const updateOption = (qIdx: number, oIdx: number, updates: Partial<ContestQuizOption>) => {
    setQuiz((prev) => ({
      ...prev,
      questions: prev.questions.map((q, i) => {
        if (i !== qIdx) return q;
        return {
          ...q,
          options: q.options.map((o, j) => (j === oIdx ? { ...o, ...updates } : o)),
        };
      }),
    }));
  };

  const addOption = (qIdx: number) => {
    setQuiz((prev) => ({
      ...prev,
      questions: prev.questions.map((q, i) => {
        if (i !== qIdx) return q;
        return { ...q, options: [...q.options, { option_text: '', is_correct: false }] };
      }),
    }));
  };

  const removeOption = (qIdx: number, oIdx: number) => {
    setQuiz((prev) => ({
      ...prev,
      questions: prev.questions.map((q, i) => {
        if (i !== qIdx) return q;
        return { ...q, options: q.options.filter((_, j) => j !== oIdx) };
      }),
    }));
  };

  const toggleCorrectOption = (qIdx: number, oIdx: number) => {
    setQuiz((prev) => ({
      ...prev,
      questions: prev.questions.map((q, i) => {
        if (i !== qIdx) return q;
        return {
          ...q,
          options: q.options.map((o, j) => {
            if (j !== oIdx)
              return { ...o, is_correct: q.question_type === 'MCQ' ? false : o.is_correct };
            return { ...o, is_correct: !o.is_correct };
          }),
        };
      }),
    }));
  };

  const handleDurationChange = (value: string) => {
    const num = toNum(value);
    setQuiz((prev) => ({ ...prev, duration_minutes: num }));
  };

  return (
    <Modal onClose={onClose}>
      <div className="max-h-[80vh] overflow-y-auto pr-1">
        <h2 className="text-xl font-bold text-text-primary mb-1">Questions: {sectionName}</h2>
        <p className="text-sm text-text-secondary mb-4">Create quiz questions for this section.</p>

        {loading ? (
          <p className="text-text-muted py-8 text-center">Loading...</p>
        ) : (
          <div className="space-y-4">
            {/* Optional Duration */}
            <div>
              <label className="block text-sm font-medium text-text-primary mb-1">
                Duration (minutes, optional)
              </label>
              <input
                type="text"
                inputMode="numeric"
                className="input w-32"
                value={quiz.duration_minutes || ''}
                onChange={(e) => handleDurationChange(e.target.value)}
                placeholder="e.g. 30"
              />
            </div>

            {/* Questions */}
            <div className="border-t border-background-border pt-4">
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-4">
                  <h3 className="font-semibold text-text-primary">
                    Questions ({quiz.questions.length})
                  </h3>
                  <span className="text-sm text-text-secondary">
                    Total Marks: {quiz.questions.length}
                  </span>
                </div>
                <button
                  type="button"
                  onClick={addQuestion}
                  className="btn btn-secondary text-xs px-3 py-1.5"
                >
                  + Add Question
                </button>
              </div>

              {quiz.questions.length === 0 && (
                <p className="text-text-muted text-sm text-center py-4">
                  No questions yet. Click "Add Question" to create one.
                </p>
              )}

              <div className="space-y-4">
                {quiz.questions.map((q, qIdx) => (
                  <div
                    key={qIdx}
                    className="bg-background-tertiary rounded-lg border border-background-border p-4"
                  >
                    <div className="flex items-center justify-between mb-3">
                      <span className="text-sm font-medium text-text-primary">
                        Question {qIdx + 1}
                      </span>
                      <div className="flex gap-2">
                        <button
                          type="button"
                          onClick={() =>
                            setEditingQuestionIdx(editingQuestionIdx === qIdx ? null : qIdx)
                          }
                          className="text-xs px-2 py-1 rounded border border-background-border hover:bg-background-secondary transition-colors"
                        >
                          {editingQuestionIdx === qIdx ? 'Collapse' : 'Edit'}
                        </button>
                        <button
                          type="button"
                          onClick={() => removeQuestion(qIdx)}
                          className="text-xs px-2 py-1 rounded border border-accent-danger text-accent-danger hover:bg-accent-danger/10 transition-colors"
                        >
                          Remove
                        </button>
                      </div>
                    </div>

                    {editingQuestionIdx !== qIdx ? (
                      <div>
                        <p className="text-text-primary text-sm mb-2">
                          {q.question_text || (
                            <span className="text-text-muted italic">No question text</span>
                          )}
                        </p>
                        <div className="flex flex-wrap gap-2">
                          {q.options.map((opt, oIdx) => (
                            <span
                              key={oIdx}
                              className={`text-xs px-2 py-1 rounded border ${opt.is_correct ? 'bg-accent-success/10 border-accent-success text-accent-success' : 'bg-background-secondary border-background-border text-text-secondary'}`}
                            >
                              {opt.option_text || '(empty)'}
                            </span>
                          ))}
                        </div>
                      </div>
                    ) : (
                      <div className="space-y-3">
                        <div>
                          <label className="block text-xs font-medium text-text-secondary mb-1">
                            Question Text *
                          </label>
                          <textarea
                            className="input text-sm"
                            rows={2}
                            value={q.question_text}
                            onChange={(e) =>
                              updateQuestion(qIdx, { question_text: e.target.value })
                            }
                            placeholder="Enter question..."
                          />
                        </div>
                        <div className="grid grid-cols-2 gap-3">
                          <div>
                            <label className="block text-xs font-medium text-text-secondary mb-1">
                              Type
                            </label>
                            <select
                              className="select text-sm"
                              value={q.question_type}
                              onChange={(e) =>
                                updateQuestion(qIdx, {
                                  question_type: e.target.value as 'MCQ' | 'TF',
                                })
                              }
                            >
                              <option value="MCQ">Multiple Choice</option>
                              <option value="TF">True / False</option>
                            </select>
                          </div>
                          <div className="flex items-center">
                            <span className="text-sm text-text-secondary">Marks: 1</span>
                          </div>
                        </div>
                        <div>
                          <label className="block text-xs font-medium text-text-secondary mb-1">
                            Options (check correct answer)
                          </label>
                          <div className="space-y-2">
                            {q.options.map((opt, oIdx) => (
                              <div key={oIdx} className="flex items-center gap-2">
                                <input
                                  type="checkbox"
                                  checked={opt.is_correct}
                                  onChange={() => toggleCorrectOption(qIdx, oIdx)}
                                  className="w-4 h-4 accent-accent-secondary"
                                />
                                <input
                                  type="text"
                                  className="input text-sm flex-1"
                                  value={opt.option_text}
                                  onChange={(e) =>
                                    updateOption(qIdx, oIdx, { option_text: e.target.value })
                                  }
                                  placeholder={`Option ${oIdx + 1}`}
                                />
                                <button
                                  type="button"
                                  onClick={() => removeOption(qIdx, oIdx)}
                                  className="text-accent-danger hover:text-accent-danger/80 text-xs px-2"
                                >
                                  ×
                                </button>
                              </div>
                            ))}
                          </div>
                          <button
                            type="button"
                            onClick={() => addOption(qIdx)}
                            className="mt-2 text-xs text-accent-secondary hover:text-accent-secondary/80"
                          >
                            + Add Option
                          </button>
                        </div>
                        <div>
                          <label className="block text-xs font-medium text-text-secondary mb-1">
                            Explanation (optional)
                          </label>
                          <input
                            type="text"
                            className="input text-sm"
                            value={q.explanation || ''}
                            onChange={(e) => updateQuestion(qIdx, { explanation: e.target.value })}
                            placeholder="Explanation shown after submission..."
                          />
                        </div>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>

            {error && <p className="text-accent-danger text-sm">{error}</p>}

            <div className="flex justify-end gap-3 pt-2 border-t border-background-border">
              <button
                type="button"
                onClick={onClose}
                className="px-4 py-2 border border-background-border rounded-lg text-text-primary hover:bg-background-tertiary transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleSaveQuiz}
                disabled={saving}
                className="btn btn-primary"
              >
                {saving ? 'Saving...' : 'Save Questions'}
              </button>
            </div>
          </div>
        )}
      </div>
    </Modal>
  );
}
