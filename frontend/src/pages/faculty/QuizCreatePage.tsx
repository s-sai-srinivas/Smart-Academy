import { useState, useEffect, useCallback, type ChangeEvent, type FormEvent } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { quizAPI } from '../../services/api';
import { showError, showWarning } from '../../utils/showAlert';
import { Sparkles, ChevronLeft, Loader2 } from 'lucide-react';

interface Offering {
  course_offering_id: number;
  display_name: string;
}

interface TheoryModule {
  id: number;
  module_name: string;
  week_name?: string;
  description?: string;
}

interface FormData {
  title: string;
  description: string;
  quiz_type: string;
  time_limit_minutes: number;
  total_marks: number;
  passing_marks: number;
  max_attempts: number;
}

interface AISettings {
  question_count: number;
  difficulty_easy: number;
  difficulty_medium: number;
  difficulty_hard: number;
  include_mcq: boolean;
  include_true_false: boolean;
  include_multi_select: boolean;
  custom_prompt: string;
}

function QuizCreatePage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const defaultOffering = searchParams.get('offering');

  const [loading, setLoading] = useState<boolean>(false);
  const [generating, setGenerating] = useState<boolean>(false);
  const [offerings, setOfferings] = useState<Offering[]>([]);
  const [selectedOffering, setSelectedOffering] = useState<string>(defaultOffering || '');
  const [theoryModules, setTheoryModules] = useState<TheoryModule[]>([]);
  const [selectedModules, setSelectedModules] = useState<number[]>([]);

  const [formData, setFormData] = useState<FormData>({
    title: '',
    description: '',
    quiz_type: 'ai_generated',
    time_limit_minutes: 30,
    total_marks: 100,
    passing_marks: 40,
    max_attempts: 1,
  });

  const [aiSettings, setAiSettings] = useState<AISettings>({
    question_count: 10,
    difficulty_easy: 4,
    difficulty_medium: 4,
    difficulty_hard: 2,
    include_mcq: true,
    include_true_false: true,
    include_multi_select: false,
    custom_prompt: '',
  });

  const loadOfferings = useCallback(async () => {
    try {
      const data = (await quizAPI.getMyOfferings()) as { offerings?: Offering[] };
      setOfferings(data.offerings || []);
    } catch (err) {
      console.error('Failed to load offerings:', err);
    }
  }, []);

  const loadTheoryModules = useCallback(async () => {
    try {
      const data = (await quizAPI.getTheoryModules(selectedOffering)) as {
        weeks?: Array<{ week_name?: string; week_order?: number; modules?: TheoryModule[] }>;
      };
      // Backend returns { weeks: [{ id, week_name, modules: [...] }] }
      // Flatten into a flat list of modules with week info
      const weeks = data.weeks || [];
      const flatModules: TheoryModule[] = weeks.flatMap((week) =>
        (week.modules || []).map((mod: TheoryModule) => ({
          ...mod,
          week_name: week.week_name,
        }))
      );
      setTheoryModules(flatModules);
    } catch (err) {
      console.error('Failed to load theory modules:', err);
    }
  }, [selectedOffering]);

  useEffect(() => {
    loadOfferings();
  }, [loadOfferings]);

  useEffect(() => {
    if (selectedOffering) {
      loadTheoryModules();
    }
  }, [selectedOffering, loadTheoryModules]);

  const toggleModule = (moduleId: number) => {
    setSelectedModules((prev) =>
      prev.includes(moduleId) ? prev.filter((id) => id !== moduleId) : [...prev, moduleId]
    );
  };

  const handleGenerateAI = async () => {
    if (!selectedOffering) {
      showWarning('Please select a course offering');
      return;
    }
    if (selectedModules.length === 0) {
      showWarning('Please select at least one theory module');
      return;
    }

    setGenerating(true);
    try {
      const generationData = {
        course_offering_id: parseInt(selectedOffering),
        theory_module_ids: selectedModules,
        question_count: aiSettings.question_count,
        difficulty_mix: {
          easy: aiSettings.difficulty_easy,
          medium: aiSettings.difficulty_medium,
          hard: aiSettings.difficulty_hard,
        },
        question_types: [
          ...(aiSettings.include_mcq ? ['mcq'] : []),
          ...(aiSettings.include_true_false ? ['true_false'] : []),
          ...(aiSettings.include_multi_select ? ['multi_select'] : []),
        ],
        custom_prompt: aiSettings.custom_prompt || undefined,
      };

      const result = (await quizAPI.generateAI(generationData)) as {
        questions: unknown[];
        theory_source_ids?: number[];
      };

      // Navigate to question editor with generated questions
      navigate(`/faculty/quizzes/create/manual?offering=${selectedOffering}&generated=true`, {
        state: {
          quizData: { ...formData, course_offering_id: parseInt(selectedOffering) },
          questions: result.questions,
          theorySourceIds: result.theory_source_ids,
        },
      });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      showError(`Failed to generate questions: ${message}`);
    } finally {
      setGenerating(false);
    }
  };

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!selectedOffering) {
      showWarning('Please select a course offering');
      return;
    }

    setLoading(true);
    try {
      const quizData = {
        ...formData,
        course_offering_id: parseInt(selectedOffering),
      };

      const result = (await quizAPI.create(quizData)) as {
        quiz: { id: number };
        theory_source_ids?: number[];
      };

      // Navigate to question editor
      navigate(`/faculty/quizzes/${result.quiz.id}/questions`, {
        state: { theorySourceIds: result.theory_source_ids },
      });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      showError(`Failed to create quiz: ${message}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6 max-w-4xl">
      {/* Header */}
      <div className="flex items-center gap-4">
        <button
          onClick={() => navigate('/faculty/quizzes')}
          className="p-2 hover:bg-background-tertiary rounded-lg transition-colors"
        >
          <ChevronLeft className="w-5 h-5 text-text-secondary" />
        </button>
        <div>
          <h1 className="text-3xl font-bold text-text-primary mb-2">Create New Quiz</h1>
          <p className="text-text-secondary">
            Configure quiz settings and generate questions with AI
          </p>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Course Offering Selection */}
        <div className="bg-background-secondary rounded-xl border-2 border-background-border p-6">
          <h3 className="text-lg font-semibold text-text-primary mb-4">Course Offering</h3>
          <select
            value={selectedOffering}
            onChange={(e: ChangeEvent<HTMLSelectElement>) => setSelectedOffering(e.target.value)}
            className="w-full px-3 py-2.5 bg-background-tertiary border border-background-border rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-blue-500"
            required
          >
            <option value="">Select a course offering</option>
            {offerings.map((offering) => (
              <option key={offering.course_offering_id} value={offering.course_offering_id}>
                {offering.display_name}
              </option>
            ))}
          </select>
        </div>

        {/* Quiz Settings */}
        <div className="bg-background-secondary rounded-xl border-2 border-background-border p-6">
          <h3 className="text-lg font-semibold text-text-primary mb-4">Quiz Settings</h3>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-2">
                Quiz Title
              </label>
              <input
                type="text"
                value={formData.title}
                onChange={(e: ChangeEvent<HTMLInputElement>) =>
                  setFormData({ ...formData, title: e.target.value })
                }
                className="w-full px-3 py-2.5 bg-background-tertiary border border-background-border rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="Enter quiz title"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-2">
                Time Limit (minutes)
              </label>
              <input
                type="number"
                value={formData.time_limit_minutes}
                onChange={(e: ChangeEvent<HTMLInputElement>) =>
                  setFormData({ ...formData, time_limit_minutes: parseInt(e.target.value) })
                }
                className="w-full px-3 py-2.5 bg-background-tertiary border border-background-border rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-blue-500"
                min={1}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-2">
                Total Marks
              </label>
              <input
                type="number"
                value={formData.total_marks}
                onChange={(e: ChangeEvent<HTMLInputElement>) =>
                  setFormData({ ...formData, total_marks: parseInt(e.target.value) })
                }
                className="w-full px-3 py-2.5 bg-background-tertiary border border-background-border rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-blue-500"
                min={1}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-2">
                Passing Marks
              </label>
              <input
                type="number"
                value={formData.passing_marks}
                onChange={(e: ChangeEvent<HTMLInputElement>) =>
                  setFormData({ ...formData, passing_marks: parseInt(e.target.value) })
                }
                className="w-full px-3 py-2.5 bg-background-tertiary border border-background-border rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-blue-500"
                min={0}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-2">
                Max Attempts
              </label>
              <input
                type="number"
                value={formData.max_attempts}
                onChange={(e: ChangeEvent<HTMLInputElement>) =>
                  setFormData({ ...formData, max_attempts: parseInt(e.target.value) })
                }
                className="w-full px-3 py-2.5 bg-background-tertiary border border-background-border rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-blue-500"
                min={1}
              />
            </div>
          </div>
        </div>

        {/* Theory Module Selection */}
        {selectedOffering && (
          <div className="bg-background-secondary rounded-xl border-2 border-background-border p-6">
            <h3 className="text-lg font-semibold text-text-primary mb-4">Select Theory Modules</h3>
            <p className="text-sm text-text-secondary mb-4">
              Choose the theory modules that AI should use to generate questions
            </p>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3 max-h-64 overflow-y-auto">
              {theoryModules.length === 0 ? (
                <p className="text-text-muted col-span-2">
                  No theory modules available for this course offering
                </p>
              ) : (
                theoryModules.map((module) => (
                  <label
                    key={module.id}
                    className={`flex items-start gap-3 p-4 rounded-lg border-2 cursor-pointer transition-colors ${
                      selectedModules.includes(module.id)
                        ? 'border-blue-500 bg-blue-500/10'
                        : 'border-background-border bg-background-tertiary hover:border-background-elevated'
                    }`}
                  >
                    <input
                      type="checkbox"
                      checked={selectedModules.includes(module.id)}
                      onChange={() => toggleModule(module.id)}
                      className="mt-1 w-4 h-4 text-blue-500 rounded focus:ring-blue-500"
                    />
                    <div>
                      <div className="font-medium text-text-primary">{module.module_name}</div>
                      <div className="text-sm text-text-secondary">
                        {module.week_name ? `Week: ${module.week_name}` : ''}
                        {module.description ? ` — ${module.description.substring(0, 80)}` : ''}
                      </div>
                    </div>
                  </label>
                ))
              )}
            </div>
          </div>
        )}

        {/* AI Generation Settings */}
        <div className="bg-background-secondary rounded-xl border-2 border-background-border p-6">
          <div className="flex items-center gap-2 mb-4">
            <Sparkles className="w-5 h-5 text-purple-400" />
            <h3 className="text-lg font-semibold text-text-primary">AI Generation Settings</h3>
          </div>
          <div className="grid grid-cols-4 gap-4 mb-4">
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-2">
                Question Count
              </label>
              <input
                type="number"
                value={aiSettings.question_count}
                onChange={(e: ChangeEvent<HTMLInputElement>) =>
                  setAiSettings({ ...aiSettings, question_count: parseInt(e.target.value) })
                }
                className="w-full px-3 py-2.5 bg-background-tertiary border border-background-border rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-blue-500"
                min={1}
                max={50}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-2">Easy</label>
              <input
                type="number"
                value={aiSettings.difficulty_easy}
                onChange={(e: ChangeEvent<HTMLInputElement>) =>
                  setAiSettings({ ...aiSettings, difficulty_easy: parseInt(e.target.value) })
                }
                className="w-full px-3 py-2.5 bg-background-tertiary border border-background-border rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-green-500"
                min={0}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-2">Medium</label>
              <input
                type="number"
                value={aiSettings.difficulty_medium}
                onChange={(e: ChangeEvent<HTMLInputElement>) =>
                  setAiSettings({ ...aiSettings, difficulty_medium: parseInt(e.target.value) })
                }
                className="w-full px-3 py-2.5 bg-background-tertiary border border-background-border rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-yellow-500"
                min={0}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-400 mb-2">Hard</label>
              <input
                type="number"
                value={aiSettings.difficulty_hard}
                onChange={(e: ChangeEvent<HTMLInputElement>) =>
                  setAiSettings({ ...aiSettings, difficulty_hard: parseInt(e.target.value) })
                }
                className="w-full px-3 py-2.5 bg-background-tertiary border border-background-border rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-red-500"
                min={0}
              />
            </div>
          </div>
          <div className="flex gap-4 mb-4">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={aiSettings.include_mcq}
                onChange={(e: ChangeEvent<HTMLInputElement>) =>
                  setAiSettings({ ...aiSettings, include_mcq: e.target.checked })
                }
                className="w-4 h-4 text-blue-500 rounded focus:ring-blue-500"
              />
              <span className="text-slate-300">Multiple Choice (MCQ)</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={aiSettings.include_true_false}
                onChange={(e: ChangeEvent<HTMLInputElement>) =>
                  setAiSettings({ ...aiSettings, include_true_false: e.target.checked })
                }
                className="w-4 h-4 text-blue-500 rounded focus:ring-blue-500"
              />
              <span className="text-slate-300">True/False</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={aiSettings.include_multi_select}
                onChange={(e: ChangeEvent<HTMLInputElement>) =>
                  setAiSettings({ ...aiSettings, include_multi_select: e.target.checked })
                }
                className="w-4 h-4 text-blue-500 rounded focus:ring-blue-500"
              />
              <span className="text-slate-300">Multi-Select</span>
            </label>
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-400 mb-2">
              Custom Instructions (Optional)
            </label>
            <textarea
              value={aiSettings.custom_prompt}
              onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                setAiSettings({ ...aiSettings, custom_prompt: e.target.value })
              }
              className="w-full px-3 py-2.5 bg-background-tertiary border border-background-border rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
              rows={3}
              placeholder="e.g., Focus on practical applications, include real-world examples..."
            />
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex gap-4">
          <button
            type="button"
            onClick={handleGenerateAI}
            disabled={generating || selectedModules.length === 0}
            className="flex-1 flex items-center justify-center gap-2 px-4 py-3 bg-purple-500 hover:bg-purple-600 disabled:bg-slate-600 text-white rounded-lg font-medium transition-colors"
          >
            {generating ? (
              <Loader2 className="w-5 h-5 animate-spin" />
            ) : (
              <Sparkles className="w-5 h-5" />
            )}
            {generating ? 'Generating...' : 'Generate Questions with AI'}
          </button>
          <button
            type="submit"
            disabled={loading || !formData.title || !selectedOffering}
            className="flex-1 px-4 py-3 bg-blue-500 hover:bg-blue-600 disabled:bg-slate-600 text-white rounded-lg font-medium transition-colors"
          >
            {loading ? 'Creating...' : 'Create Quiz'}
          </button>
        </div>
      </form>
    </div>
  );
}

export default QuizCreatePage;
