import { useState, FormEvent, ChangeEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { contestGenerationAPI } from '../../services/api';
import { getErrorMessage } from '../../utils/error';

interface FormData {
  topics: string;
  requested_count: number;
  versions_per_problem: number;
  difficulty_level: string;
  generation_type: string;
  start_time: string;
  end_time: string;
}

function ContestGenerationRequest() {
  const navigate = useNavigate();
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string>('');
  const [formData, setFormData] = useState<FormData>({
    topics: '',
    requested_count: 5,
    versions_per_problem: 1,
    difficulty_level: 'medium',
    generation_type: 'problems',
    start_time: '',
    end_time: '',
  });

  const handleInputChange = (
    e: ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>
  ) => {
    const { name, value } = e.target;
    setFormData((prev) => ({ ...prev, [name]: value }));
    setError('');
  };

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError('');

    const topicsArray = formData.topics
      .split(',')
      .map((t) => t.trim())
      .filter((t) => t.length > 0);

    if (topicsArray.length === 0) {
      setError('Please enter at least one topic');
      return;
    }

    setLoading(true);

    try {
      const response = (await contestGenerationAPI.generate({
        topics: topicsArray,
        requested_count: parseInt(String(formData.requested_count)),
        generation_type: formData.generation_type,
        difficulty_level: formData.difficulty_level,
        start_time: formData.start_time || undefined,
        end_time: formData.end_time || undefined,
      })) as { job_id: string };

      navigate(`/admin/review-generated-content?type=contest&jobId=${response.job_id}`);
    } catch (err: unknown) {
      const message = getErrorMessage(err, 'Failed to start contest generation. Please try again.');
      console.error('Failed to start generation:', err);
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  const requestedCount = parseInt(String(formData.requested_count)) || 5;
  const versionsPerProblem = parseInt(String(formData.versions_per_problem)) || 1;
  const totalGenerated = requestedCount * versionsPerProblem;

  const isQuizMode = formData.generation_type === 'quiz';
  const isBothMode = formData.generation_type === 'both';

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-text-primary mb-1">AI Contest Generation</h1>
        <p className="text-text-secondary text-sm">
          Generate contest problems and quizzes with AI. No course selection needed.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Form */}
        <div className="lg:col-span-2">
          <form onSubmit={handleSubmit} className="card space-y-5">
            {/* Generation Type */}
            <div className="form-group">
              <label className="form-label">
                Generation Mode <span className="text-accent-danger">*</span>
              </label>
              <div className="flex gap-2">
                {[
                  { value: 'problems', label: 'Problems Only', desc: 'Coding problems' },
                  { value: 'quiz', label: 'Quiz Only', desc: 'MCQ questions' },
                  { value: 'both', label: 'Both', desc: 'Problems + Quiz' },
                ].map((mode) => (
                  <button
                    key={mode.value}
                    type="button"
                    onClick={() =>
                      setFormData((prev) => ({ ...prev, generation_type: mode.value }))
                    }
                    className={`flex-1 p-3 rounded-lg border text-center transition-colors ${
                      formData.generation_type === mode.value
                        ? 'border-accent-primary bg-accent-primary/10 text-accent-primary'
                        : 'border-background-border bg-background-secondary text-text-secondary hover:border-background-border-hover'
                    }`}
                  >
                    <div className="text-sm font-semibold">{mode.label}</div>
                    <div className="text-xs mt-0.5 opacity-70">{mode.desc}</div>
                  </button>
                ))}
              </div>
            </div>

            {/* Topics */}
            <div className="form-group">
              <label className="form-label">
                Topics <span className="text-accent-danger">*</span>
              </label>
              <textarea
                name="topics"
                value={formData.topics}
                onChange={handleInputChange}
                placeholder="e.g., Arrays, Sorting, Binary Search, Dynamic Programming"
                rows={3}
                className="input resize-none"
                required
              />
              <p className="text-text-muted text-xs">Separate multiple topics with commas</p>
            </div>

            {/* Count + Difficulty */}
            <div className="form-row">
              <div className="form-group">
                <label className="form-label">
                  {isQuizMode ? 'Questions Needed' : 'Problems Needed'}{' '}
                  <span className="text-accent-danger">*</span>
                </label>
                <select
                  name="requested_count"
                  value={formData.requested_count}
                  onChange={handleInputChange}
                  className="select"
                >
                  {[1, 2, 3, 4, 5, 6, 7, 8, 9, 10].map((n) => (
                    <option key={n} value={n}>
                      {n}{' '}
                      {n === 1
                        ? isQuizMode
                          ? 'question'
                          : 'problem'
                        : isQuizMode
                          ? 'questions'
                          : 'problems'}
                    </option>
                  ))}
                </select>
              </div>
              <div className="form-group">
                <label className="form-label">Difficulty</label>
                <select
                  name="difficulty_level"
                  value={formData.difficulty_level}
                  onChange={handleInputChange}
                  className="select"
                >
                  <option value="easy">Easy</option>
                  <option value="medium">Medium</option>
                  <option value="hard">Hard</option>
                  <option value="mixed">Mixed</option>
                </select>
              </div>
            </div>

            {/* Versions per Problem (only for problem modes) */}
            {!isQuizMode && (
              <div className="form-group">
                <label className="form-label">Versions per Problem</label>
                <select
                  name="versions_per_problem"
                  value={formData.versions_per_problem}
                  onChange={handleInputChange}
                  className="select"
                >
                  <option value={1}>1x - Exact count (fastest)</option>
                  <option value={2}>2x - Double for selection</option>
                  <option value={3}>3x - Triple for more variety</option>
                </select>
                <p className="text-text-muted text-xs">
                  {versionsPerProblem === 1
                    ? 'Generates exactly the number of problems you need'
                    : `Generates ${totalGenerated} total problems so you can pick the best ${requestedCount}`}
                </p>
              </div>
            )}

            {/* Optional Timing */}
            <div className="form-row">
              <div className="form-group">
                <label className="form-label">
                  Contest Start <span className="text-text-muted font-normal">(optional)</span>
                </label>
                <input
                  type="datetime-local"
                  name="start_time"
                  value={formData.start_time}
                  onChange={handleInputChange}
                  className="input"
                />
              </div>
              <div className="form-group">
                <label className="form-label">
                  Contest End <span className="text-text-muted font-normal">(optional)</span>
                </label>
                <input
                  type="datetime-local"
                  name="end_time"
                  value={formData.end_time}
                  onChange={handleInputChange}
                  className="input"
                />
              </div>
            </div>

            {/* Inline error */}
            {error && (
              <div className="flex items-center gap-2 px-4 py-3 rounded-lg bg-accent-danger/10 border border-accent-danger/20 text-accent-danger text-sm">
                <svg
                  className="w-4 h-4 shrink-0"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
                {error}
              </div>
            )}

            {/* Actions */}
            <div className="flex gap-3 pt-2">
              <button
                type="button"
                onClick={() => navigate(-1)}
                className="btn btn-secondary flex-1"
              >
                Cancel
              </button>
              <button type="submit" disabled={loading} className="btn btn-primary flex-1">
                {loading ? (
                  <>
                    <svg className="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
                      <circle
                        className="opacity-25"
                        cx="12"
                        cy="12"
                        r="10"
                        stroke="currentColor"
                        strokeWidth="4"
                      />
                      <path
                        className="opacity-75"
                        fill="currentColor"
                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
                      />
                    </svg>
                    Starting Generation...
                  </>
                ) : (
                  <>
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M13 10V3L4 14h7v7l9-11h-7z"
                      />
                    </svg>
                    Generate {isQuizMode ? 'Quiz' : isBothMode ? 'Content' : 'Problems'}
                  </>
                )}
              </button>
            </div>
          </form>
        </div>

        {/* Info Sidebar */}
        <div className="space-y-4">
          {/* Live Preview Card */}
          <div className="card space-y-3">
            <h3 className="text-sm font-semibold text-text-primary">Generation Preview</h3>
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-text-secondary">Mode</span>
                <span className="font-medium text-text-primary capitalize">
                  {formData.generation_type === 'problems'
                    ? 'Problems Only'
                    : formData.generation_type === 'quiz'
                      ? 'Quiz Only'
                      : 'Problems + Quiz'}
                </span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-text-secondary">
                  {isQuizMode ? 'Questions' : 'Problems'} to generate
                </span>
                <span className="font-medium text-text-primary">{requestedCount}</span>
              </div>
              {!isQuizMode && (
                <div className="flex justify-between text-sm">
                  <span className="text-text-secondary">Versions per problem</span>
                  <span className="font-medium text-text-primary">{versionsPerProblem}x</span>
                </div>
              )}
              <div className="flex justify-between text-sm">
                <span className="text-text-secondary">Difficulty</span>
                <span className="font-medium text-text-primary capitalize">
                  {formData.difficulty_level}
                </span>
              </div>
            </div>
            <div className="h-px bg-background-border" />
            <p className="text-xs text-text-muted text-center">
              AI generates content based on topics - no course needed
            </p>
          </div>

          {/* Steps */}
          <div className="card space-y-3">
            <h3 className="text-sm font-semibold text-text-primary">What happens next?</h3>
            <ol className="space-y-3">
              {[
                `AI generates ${!isQuizMode ? requestedCount + ' problems with test cases' : ''}${isBothMode ? ' and ' : ''}${isQuizMode || isBothMode ? requestedCount + ' quiz questions' : ''}`,
                'You review and approve the generated content',
                'Approved content is saved and can be added to contests',
              ].map((step, i) => (
                <li key={i} className="flex items-start gap-3 text-sm text-text-secondary">
                  <span className="w-5 h-5 rounded-full bg-accent-primary/10 text-accent-primary text-xs font-bold flex items-center justify-center shrink-0 mt-0.5">
                    {i + 1}
                  </span>
                  {step}
                </li>
              ))}
            </ol>
          </div>
        </div>
      </div>
    </div>
  );
}

export default ContestGenerationRequest;
