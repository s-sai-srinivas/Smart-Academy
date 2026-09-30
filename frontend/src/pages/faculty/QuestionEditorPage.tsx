import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate, useLocation } from 'react-router-dom';
import { quizAPI } from '../../services/api';
import { showSuccess, showError, showWarning, showInfo } from '../../utils/showAlert';
import { ChevronLeft, Plus, Save } from 'lucide-react';
import { QuestionEditor, QuestionPreview } from '../../components/faculty/QuestionEditor';
import type { QuizQuestion } from '../../components/faculty/QuestionEditor';

type PageQuestion = QuizQuestion & {
  bloom_level?: string;
  source_module_id?: number;
};

interface QuizData {
  title: string;
  description: string;
  quiz_type: string;
  time_limit_minutes: number;
  total_marks: number;
  passing_marks: number;
  max_attempts: number;
  course_offering_id?: number;
}

interface QuizResult {
  quiz?: { id: number };
  theory_source_ids?: number[];
}

interface LocationState {
  questions?: PageQuestion[];
  quizData?: QuizData;
  theorySourceIds?: number[];
}

function QuestionEditorPage() {
  const { quizId } = useParams<{ quizId: string }>();
  const numericQuizId = Number(quizId);
  const navigate = useNavigate();
  const location = useLocation();
  const state = (location.state as LocationState) || {};

  const [saving, setSaving] = useState<boolean>(false);
  const [, setQuiz] = useState<{ quiz: QuizData } | null>(null);
  const [questions, setQuestions] = useState<PageQuestion[]>(state.questions || []);
  const [editingQuestion, setEditingQuestion] = useState<number | null>(null);

  // Normalize questions to ensure consistent format (handles AI-generated questions)
  const normalizeQuestions = (rawQuestions: PageQuestion[] | undefined): PageQuestion[] => {
    return (rawQuestions || []).map((q) => {
      // Ensure every option has a unique id
      const options = (q.options || []).map((opt, idx) => ({
        ...opt,
        id: opt.id ?? Date.now() + idx + Math.random(),
      }));

      // Build correct_option_ids from options' is_correct if not already set
      let correctIds = q.correct_option_ids;
      if (!correctIds || correctIds.length === 0) {
        correctIds = options.filter((o) => o.is_correct).map((o) => o.id);
      }

      return {
        ...q,
        options,
        correct_option_ids: correctIds,
        is_ai_generated: q.is_ai_generated ?? true,
      };
    });
  };

  const loadQuiz = useCallback(async () => {
    try {
      const data = (await quizAPI.get(numericQuizId)) as {
        quiz: QuizData;
        questions?: PageQuestion[];
      };
      setQuiz(data);
      setQuestions(normalizeQuestions(data.questions));
    } catch (err) {
      console.error('Failed to load quiz:', err);
    }
  }, [numericQuizId]);

  useEffect(() => {
    if (state.questions) {
      // AI-generated or state-passed questions — normalize them
      setQuestions(normalizeQuestions(state.questions));
    } else {
      loadQuiz();
    }
  }, [state.questions, loadQuiz]);

  const handleAddQuestion = () => {
    const newQuestion: PageQuestion = {
      question_type: 'mcq',
      question_text: '',
      explanation: '',
      marks: 10,
      difficulty: 'medium',
      bloom_level: 'understand',
      options: [],
      correct_option_ids: [],
      is_ai_generated: false,
    };
    setQuestions([...questions, newQuestion]);
    setEditingQuestion(questions.length);
  };

  const handleSaveQuestion = (index: number, questionData: PageQuestion) => {
    const updated = [...questions];
    // Ensure all fields are properly copied including options
    updated[index] = {
      ...questionData,
      options: questionData.options || [],
      correct_option_ids: questionData.correct_option_ids || [],
    };
    setQuestions(updated);
    setEditingQuestion(null);
  };

  const handleDeleteQuestion = (index: number) => {
    if (!confirm('Delete this question?')) return;
    setQuestions(questions.filter((_, i) => i !== index));
  };

  const handleSaveAll = async () => {
    if (questions.length === 0) {
      showWarning('Please add at least one question');
      return;
    }

    // Validate questions
    for (let i = 0; i < questions.length; i++) {
      const q = questions[i];
      if (!q.question_text.trim()) {
        showWarning(`Question ${i + 1}: Question text is required`);
        return;
      }
      if (['mcq', 'multi_select', 'true_false'].includes(q.question_type)) {
        if (!q.options || q.options.length === 0) {
          showWarning(`Question ${i + 1}: At least one option is required`);
          return;
        }
        const correctIds = q.correct_option_ids || [];
        const hasCorrect = correctIds.length > 0 || q.options.some((o) => o.is_correct);
        if (!hasCorrect) {
          showWarning(`Question ${i + 1}: Please mark the correct option(s)`);
          return;
        }
      }
    }

    // Prepare questions for backend: set is_correct on options from correct_option_ids
    const preparedQuestions = questions.map((q, idx) => {
      const correctIds = new Set<number>((q.correct_option_ids || []).map(Number));
      return {
        question_type: q.question_type,
        question_text: q.question_text,
        explanation: q.explanation || '',
        marks: q.marks || 1,
        difficulty: q.difficulty || 'medium',
        bloom_level: q.bloom_level || 'understand',
        order_index: idx,
        is_ai_generated: q.is_ai_generated || false,
        source_module_id: q.source_module_id || undefined,
        options: (q.options || []).map((opt, j) => ({
          option_text: opt.option_text,
          is_correct: correctIds.size > 0 ? correctIds.has(opt.id) : !!opt.is_correct,
          order_index: j,
        })),
      };
    });

    setSaving(true);
    try {
      if (quizId) {
        // Existing quiz — just save questions
        await quizAPI.saveQuestions(quizId, preparedQuestions);
        showSuccess('Questions saved successfully');
        navigate(`/faculty/quizzes/${quizId}/edit`);
      } else if (state.quizData) {
        // New quiz from AI generation — create quiz first, then save questions
        const quizResult = (await quizAPI.create({
          ...state.quizData,
          theory_module_ids: state.theorySourceIds || [],
        })) as QuizResult;
        const newQuizId = quizResult.quiz?.id;
        if (!newQuizId) {
          throw new Error('Quiz was created but no ID returned');
        }
        await quizAPI.saveQuestions(newQuizId, preparedQuestions);

        // Ask faculty if they want to publish immediately
        const shouldPublish = window.confirm(
          'Quiz created with questions successfully!\n\nWould you like to publish it now so students can see it?\n(Click Cancel to keep it as draft)'
        );
        if (shouldPublish) {
          try {
            await quizAPI.publish(newQuizId);
            showSuccess('Quiz published! Students can now see it.');
          } catch (pubErr: unknown) {
            const msg = pubErr instanceof Error ? pubErr.message : String(pubErr);
            showInfo(
              'Quiz saved but failed to publish: ' +
                msg +
                '\nYou can publish it from the quiz management page.'
            );
          }
        }
        navigate('/faculty/quizzes');
      } else {
        showError('Cannot save: no quiz context available');
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      showError('Failed to save: ' + msg);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6 max-w-5xl">
      {/* Header */}
      <div className="flex items-center gap-4">
        <button
          onClick={() => navigate(quizId ? `/faculty/quizzes/${quizId}/edit` : '/faculty/quizzes')}
          className="p-2 hover:bg-background-tertiary rounded-lg transition-colors"
        >
          <ChevronLeft className="w-5 h-5 text-slate-400" />
        </button>
        <div className="flex-1">
          <h1 className="text-3xl font-bold text-white mb-2">
            {state.questions ? 'Review & Save AI Questions' : 'Question Editor'}
          </h1>
          <p className="text-slate-400">
            {questions.length} question{questions.length !== 1 ? 's' : ''} added
          </p>
        </div>
        {questions.length > 0 && (
          <button
            onClick={handleSaveAll}
            disabled={saving}
            className="flex items-center gap-2 px-6 py-3 bg-green-500 hover:bg-green-600 disabled:bg-slate-600 text-white rounded-lg font-medium transition-colors"
          >
            <Save className="w-5 h-5" />
            {saving
              ? 'Saving...'
              : state.quizData && !quizId
                ? 'Create Quiz & Save'
                : 'Save All Questions'}
          </button>
        )}
      </div>

      {/* Questions List */}
      <div className="space-y-4">
        {questions.length === 0 ? (
          <div className="text-center py-12 bg-background-tertiary rounded-lg border border-background-border">
            <p className="text-slate-400 text-lg mb-4">No questions added yet</p>
            <button
              onClick={handleAddQuestion}
              className="flex items-center gap-2 px-6 py-3 bg-blue-500 hover:bg-blue-600 text-white rounded-lg font-medium transition-colors"
            >
              <Plus className="w-5 h-5" />
              Add Your First Question
            </button>
          </div>
        ) : (
          questions.map((question, index) =>
            editingQuestion === index ? (
              <QuestionEditor
                key={index}
                index={index}
                question={question}
                onSave={handleSaveQuestion}
                onCancel={() => setEditingQuestion(null)}
              />
            ) : (
              <QuestionPreview
                key={index}
                index={index}
                question={question}
                onEdit={() => setEditingQuestion(index)}
                onDelete={() => handleDeleteQuestion(index)}
              />
            )
          )
        )}
      </div>

      {/* Add Question Button - only show when there are questions */}
      {!state.questions && questions.length > 0 && (
        <button
          onClick={handleAddQuestion}
          className="w-full py-4 border-2 border-dashed border-background-border hover:border-blue-500 rounded-xl text-slate-400 hover:text-blue-400 transition-colors flex items-center justify-center gap-2"
        >
          <Plus className="w-5 h-5" />
          Add Another Question
        </button>
      )}
    </div>
  );
}

export default QuestionEditorPage;
