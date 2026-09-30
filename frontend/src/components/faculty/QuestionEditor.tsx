import { useState, type KeyboardEvent } from 'react';
import { Save, X, Plus, Trash2, Sparkles, Edit2 } from 'lucide-react';

const questionTypes = [
  { value: 'mcq', label: 'Multiple Choice (Single Answer)' },
  { value: 'multi_select', label: 'Multiple Select' },
  { value: 'true_false', label: 'True/False' },
  { value: 'short_answer', label: 'Short Answer' },
  { value: 'fill_blank', label: 'Fill in the Blank' },
] as const;

const difficultyLevels = [
  { value: 'easy', label: 'Easy' },
  { value: 'medium', label: 'Medium' },
  { value: 'hard', label: 'Hard' },
] as const;

type QuestionType = (typeof questionTypes)[number]['value'];
type Difficulty = (typeof difficultyLevels)[number]['value'];

export interface QuizOption {
  id: number;
  option_text: string;
  is_correct: boolean;
}

export interface QuizQuestion {
  question_type: QuestionType;
  difficulty: Difficulty;
  bloom_category?: string;
  question_text: string;
  options?: QuizOption[];
  correct_option_ids?: number[];
  expected_answer?: string;
  explanation?: string;
  marks: number;
  is_ai_generated?: boolean;
}

export interface QuestionEditorProps {
  index: number;
  question: QuizQuestion;
  onSave: (index: number, question: QuizQuestion) => void;
  onCancel: () => void;
}

export function QuestionEditor({ index, question, onSave, onCancel }: QuestionEditorProps) {
  const [edited, setEdited] = useState<QuizQuestion>({
    ...question,
    options: question.options?.map((opt) => ({ ...opt })) || [],
    correct_option_ids: question.correct_option_ids ? [...question.correct_option_ids] : [],
  });
  const [localNewOption, setLocalNewOption] = useState('');
  const [errors, _setErrors] = useState<Record<string, string>>({});

  const addOption = () => {
    if (!localNewOption.trim()) return;
    const newOptionObj: QuizOption = {
      id: Date.now(),
      option_text: localNewOption.trim(),
      is_correct: false,
    };
    setEdited({ ...edited, options: [...(edited.options || []), newOptionObj] });
    setLocalNewOption('');
  };

  const removeOption = (optIndex: number) => {
    const optionToRemove = edited.options?.[optIndex];
    if (!optionToRemove) return;
    setEdited({
      ...edited,
      options: edited.options?.filter((_, i) => i !== optIndex) || [],
      correct_option_ids: edited.correct_option_ids?.filter((id) => id !== optionToRemove.id) || [],
    });
  };

  const toggleCorrect = (optId: number) => {
    if (edited.question_type === 'mcq' || edited.question_type === 'true_false') {
      setEdited({
        ...edited,
        correct_option_ids: edited.correct_option_ids?.includes(optId) ? [] : [optId],
      });
    } else if (edited.question_type === 'multi_select') {
      setEdited({
        ...edited,
        correct_option_ids: edited.correct_option_ids?.includes(optId)
          ? edited.correct_option_ids.filter((id) => id !== optId)
          : [...(edited.correct_option_ids || []), optId],
      });
    }
  };

  const addDefaultTrueFalseOptions = () => {
    setEdited({
      ...edited,
      options: [
        { id: 1, option_text: 'True', is_correct: false },
        { id: 2, option_text: 'False', is_correct: false },
      ],
      correct_option_ids: [],
    });
  };

  return (
    <div className="bg-background-tertiary rounded-lg border-2 border-background-border p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h4 className="font-semibold text-text-primary">Question {index + 1}</h4>
        <div className="flex gap-2">
          <button
            onClick={() => onSave(index, edited)}
            className="flex items-center gap-1 px-3 py-1.5 bg-green-500 hover:bg-green-600 text-white rounded-lg text-sm font-medium transition-colors"
          >
            <Save className="w-4 h-4" />
            Save
          </button>
          <button
            onClick={onCancel}
            className="flex items-center gap-1 px-3 py-1.5 bg-slate-500 hover:bg-slate-600 text-white rounded-lg text-sm font-medium transition-colors"
          >
            <X className="w-4 h-4" />
            Cancel
          </button>
        </div>
      </div>
      <div className="grid grid-cols-3 gap-4">
        <div>
          <label className="block text-sm font-medium text-text-secondary mb-2">
            Question Type
          </label>
          <select
            value={edited.question_type}
            onChange={(e) =>
              setEdited({ ...edited, question_type: e.target.value as QuestionType })
            }
            className="w-full px-3 py-2 bg-background-secondary border border-background-border rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            {questionTypes.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-text-secondary mb-2">Difficulty</label>
          <select
            value={edited.difficulty}
            onChange={(e) => setEdited({ ...edited, difficulty: e.target.value as Difficulty })}
            className="w-full px-3 py-2 bg-background-secondary border border-background-border rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            {difficultyLevels.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-text-secondary mb-2">
            Bloom Category
          </label>
          <select
            value={edited.bloom_category || 'remember'}
            onChange={(e) => setEdited({ ...edited, bloom_category: e.target.value })}
            className="w-full px-3 py-2 bg-background-secondary border border-background-border rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="remember">Remember</option>
            <option value="understand">Understand</option>
            <option value="apply">Apply</option>
            <option value="analyze">Analyze</option>
            <option value="evaluate">Evaluate</option>
            <option value="create">Create</option>
          </select>
        </div>
      </div>
      <div>
        <label className="block text-sm font-medium text-text-secondary mb-2">Question</label>
        <textarea
          value={edited.question_text}
          onChange={(e) => setEdited({ ...edited, question_text: e.target.value })}
          className="w-full px-3 py-2.5 bg-background-secondary border border-background-border rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-blue-500"
          rows={3}
          placeholder="Enter your question..."
        />
        {errors.question_text && (
          <p className="mt-1 text-sm text-red-400">{errors.question_text}</p>
        )}
      </div>
      {['mcq', 'multi_select', 'true_false'].includes(edited.question_type) && (
        <div>
          <label className="block text-sm font-medium text-text-secondary mb-2">
            Options (select correct answer(s))
          </label>
          <div className="space-y-2">
            {edited.options?.map((opt, i) => (
              <div key={opt.id || i} className="flex items-center gap-2">
                <input
                  type={edited.question_type === 'multi_select' ? 'checkbox' : 'radio'}
                  checked={edited.correct_option_ids?.includes(opt.id)}
                  onChange={() => toggleCorrect(opt.id)}
                  className="w-4 h-4"
                />
                <span className="flex-1 text-text-primary">{opt.option_text}</span>
                <button
                  onClick={() => removeOption(i)}
                  className="p-1 hover:bg-red-500/10 rounded transition-colors"
                >
                  <Trash2 className="w-4 h-4 text-red-400" />
                </button>
              </div>
            ))}
          </div>
          <div className="mt-3 flex gap-2">
            <input
              type="text"
              value={localNewOption}
              onChange={(e) => setLocalNewOption(e.target.value)}
              onKeyPress={(e: KeyboardEvent<HTMLInputElement>) => e.key === 'Enter' && addOption()}
              className="flex-1 px-3 py-2 bg-background-secondary border border-background-border rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Enter option text..."
            />
            <button
              onClick={addOption}
              className="flex items-center gap-1 px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg text-sm font-medium transition-colors"
            >
              <Plus className="w-4 h-4" />
              Add
            </button>
          </div>
          {edited.question_type === 'true_false' && (edited.options?.length || 0) === 0 && (
            <button
              onClick={addDefaultTrueFalseOptions}
              className="mt-2 px-4 py-2 bg-slate-500 hover:bg-slate-600 text-white rounded-lg text-sm font-medium transition-colors"
            >
              Add True/False Options
            </button>
          )}
        </div>
      )}
      {edited.question_type === 'short_answer' || edited.question_type === 'fill_blank' ? (
        <div>
          <label className="block text-sm font-medium text-text-secondary mb-2">
            Expected Answer (for grading reference)
          </label>
          <input
            type="text"
            value={edited.expected_answer || ''}
            onChange={(e) => setEdited({ ...edited, expected_answer: e.target.value })}
            className="w-full px-3 py-2.5 bg-background-secondary border border-background-border rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="Enter expected answer..."
          />
        </div>
      ) : null}
      <div>
        <label className="block text-sm font-medium text-text-secondary mb-2">
          Explanation (Optional)
        </label>
        <textarea
          value={edited.explanation}
          onChange={(e) => setEdited({ ...edited, explanation: e.target.value })}
          className="w-full px-3 py-2.5 bg-background-secondary border border-background-border rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-blue-500"
          rows={2}
          placeholder="Add explanation to show students after submission..."
        />
      </div>
      <div>
        <label className="block text-sm font-medium text-text-secondary mb-2">Marks</label>
        <input
          type="number"
          value={edited.marks}
          onChange={(e) => setEdited({ ...edited, marks: parseInt(e.target.value) })}
          className="w-full px-3 py-2.5 bg-background-secondary border border-background-border rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-blue-500"
          min={1}
        />
      </div>
    </div>
  );
}

export interface QuestionPreviewProps {
  index: number;
  question: QuizQuestion;
  onEdit: (index: number) => void;
  onDelete: (index: number) => void;
}

export function QuestionPreview({ index, question, onEdit, onDelete }: QuestionPreviewProps) {
  const typeInfo = questionTypes.find((t) => t.value === question.question_type);
  const diffInfo = difficultyLevels.find((d) => d.value === question.difficulty);

  return (
    <div className="bg-background-tertiary rounded-lg border-2 border-background-border p-4">
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-3">
          <span className="px-2.5 py-1 bg-blue-500/10 text-blue-400 rounded-md text-xs font-medium border border-blue-500/20">
            Q{index + 1}
          </span>
          <span className="px-2.5 py-1 bg-background-tertiary text-text-secondary rounded-md text-xs font-medium border border-background-border">
            {typeInfo?.label}
          </span>
          <span
            className={`px-2.5 py-1 rounded-md text-xs font-medium border ${question.difficulty === 'easy' ? 'bg-green-500/10 text-green-400 border-green-500/20' : question.difficulty === 'medium' ? 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20' : 'bg-red-500/10 text-red-400 border-red-500/20'}`}
          >
            {diffInfo?.label}
          </span>
          <span className="px-2.5 py-1 bg-purple-500/10 text-purple-400 rounded-md text-xs font-medium border border-purple-500/20">
            {question.marks} marks
          </span>
          {question.is_ai_generated && (
            <span className="px-2.5 py-1 bg-purple-500/10 text-purple-400 rounded-md text-xs font-medium border border-purple-500/20 flex items-center gap-1">
              <Sparkles className="w-3 h-3" />
              AI Generated
            </span>
          )}
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => onEdit(index)}
            className="p-1.5 hover:bg-blue-500/10 rounded-lg transition-colors"
          >
            <Edit2 className="w-4 h-4 text-blue-400" />
          </button>
          <button
            onClick={() => onDelete(index)}
            className="p-1.5 hover:bg-red-500/10 rounded-lg transition-colors"
          >
            <Trash2 className="w-4 h-4 text-red-400" />
          </button>
        </div>
      </div>
      <p className="text-text-primary mb-3">{question.question_text}</p>
      {['mcq', 'true_false', 'multi_select'].includes(question.question_type) &&
        question.options &&
        question.options.length > 0 && (
          <div className="space-y-1.5">
            {question.options.map((opt, i) => (
              <div
                key={i}
                className={`flex items-center gap-2 p-2 rounded ${question.correct_option_ids?.includes(opt.id) ? 'bg-green-500/10 border border-green-500/20' : 'bg-slate-500/10 border border-slate-500/20'}`}
              >
                <input
                  type={question.question_type === 'multi_select' ? 'checkbox' : 'radio'}
                  checked={question.correct_option_ids?.includes(opt.id)}
                  readOnly
                  className="w-4 h-4"
                />
                <span className="text-text-primary text-sm">{opt.option_text}</span>
              </div>
            ))}
          </div>
        )}
    </div>
  );
}
