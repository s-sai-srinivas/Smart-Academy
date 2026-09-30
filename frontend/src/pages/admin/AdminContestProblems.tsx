import { useState, useEffect, useCallback } from 'react';
import { useParams, Link } from 'react-router-dom';
import { contestsAPI } from '../../services/api';
import { showError } from '../../utils/showAlert';
import CreateProblemModal from '../../components/admin/CreateProblemModal';
import ContestQuizModal from '../../components/admin/ContestQuizModal';

interface Contest {
  title?: string;
  start_time?: string;
  problems?: Problem[];
  sections?: ContestSection[];
  has_quiz?: boolean;
}

interface Problem {
  problem_id: string;
  title: string;
  description?: string;
  difficulty?: string;
  tags?: string;
  points?: number;
}

interface ContestProblem {
  id?: number;
  problem_id: number;
  points: number;
  problem_order: number;
  problem?: ProblemDetail;
}

interface ProblemDetail {
  id?: number;
  title: string;
  description?: string;
  difficulty?: string;
  tags?: string;
}

interface ContestQuizOption {
  id: number;
  option_text: string;
  is_correct: boolean;
}

interface ContestQuizQuestion {
  id: number;
  question_text: string;
  marks: number;
  options: ContestQuizOption[];
}

interface ContestQuiz {
  id: number;
  title: string;
  total_marks: number;
  questions: ContestQuizQuestion[];
}

interface ContestSection {
  id: number;
  section_name: string;
  section_type: 'coding' | 'quiz';
  description: string;
  duration_minutes: number;
  order_index: number;
  contest_problems?: ContestProblem[];
  contest_quiz?: ContestQuiz | null;
}

const getDifficultyClass = (difficulty?: string): string => {
  switch (difficulty) {
    case 'easy':
      return 'badge-success';
    case 'medium':
      return 'badge-warning';
    case 'hard':
      return 'badge-danger';
    default:
      return 'badge-neutral';
  }
};

const AdminContestProblems = () => {
  const { contestId } = useParams<{ contestId: string }>();
  const numericContestId = Number(contestId);
  const [loading, setLoading] = useState<boolean>(true);
  const [contest, setContest] = useState<Contest | null>(null);
  const [sections, setSections] = useState<ContestSection[]>([]);
  const [legacyProblems, setLegacyProblems] = useState<Problem[]>([]);

  // Modal states
  const [showCreateModal, setShowCreateModal] = useState<boolean>(false);
  const [activeSectionId, setActiveSectionId] = useState<number | undefined>(undefined);
  const [showQuizModal, setShowQuizModal] = useState<boolean>(false);
  const [quizModalSection, setQuizModalSection] = useState<ContestSection | null>(null);

  const loadContestData = useCallback(async () => {
    try {
      setLoading(true);
      // Load contest details (legacy problems) and sections in parallel
      const [contestData, sectionsData] = await Promise.all([
        contestsAPI.getById(numericContestId) as Promise<Contest>,
        contestsAPI.getContestSections(numericContestId),
      ]);

      setContest(contestData);
      setLegacyProblems(contestData?.problems || []);

      // Handle sections response
      const secs = (sectionsData as { sections?: ContestSection[] })?.sections || [];
      setSections(secs);
    } catch (err) {
      console.error('Error loading contest data:', err);
      showError(err instanceof Error ? err.message : 'Failed to load contest');
    } finally {
      setLoading(false);
    }
  }, [numericContestId]);

  useEffect(() => {
    loadContestData();
  }, [contestId, loadContestData]);

  const handleRemoveProblem = async (problemId: string) => {
    if (!confirm('Remove this problem from the contest?')) return;
    try {
      await contestsAPI.removeProblem(numericContestId, problemId);
      loadContestData();
    } catch (err) {
      console.error('Error removing problem:', err);
      showError(err instanceof Error ? err.message : 'Failed to remove problem');
    }
  };

  const handleCreateSuccess = () => {
    setShowCreateModal(false);
    setActiveSectionId(undefined);
    loadContestData();
  };

  const openAddProblem = (sectionId?: number) => {
    setActiveSectionId(sectionId);
    setShowCreateModal(true);
  };

  const openQuizModal = (section: ContestSection) => {
    setQuizModalSection(section);
    setShowQuizModal(true);
  };

  const handleQuizSuccess = () => {
    setShowQuizModal(false);
    setQuizModalSection(null);
    loadContestData();
  };

  const formatDateTime = (dateStr?: string) => {
    if (!dateStr) return '-';
    return new Date(dateStr).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      hour12: true,
    });
  };

  if (loading) {
    return (
      <div className="p-6">
        <Link to="/admin/contests" className="btn btn-secondary mb-4 inline-flex">
          ← Back to Contests
        </Link>
        <div className="card text-center py-16">
          <p className="text-text-muted">Loading...</p>
        </div>
      </div>
    );
  }

  const hasSections = sections.length > 0;

  return (
    <div className="p-6 max-w-5xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-4">
          <Link to="/admin/contests" className="btn btn-secondary">
            ← Back
          </Link>
          <div>
            <h1 className="text-2xl font-bold text-text-primary">
              {contest?.title || 'Contest Problems'}
            </h1>
            <p className="text-sm text-text-muted mt-0.5">
              {hasSections
                ? `${sections.length} section${sections.length !== 1 ? 's' : ''}`
                : `${legacyProblems.length} problem${legacyProblems.length !== 1 ? 's' : ''}`}{' '}
              &nbsp;·&nbsp;
              {formatDateTime(contest?.start_time)}
            </p>
          </div>
        </div>
        {!hasSections && (
          <button className="btn btn-primary" onClick={() => openAddProblem()}>
            + Add Problem
          </button>
        )}
      </div>

      {/* Section-aware layout */}
      {hasSections ? (
        <div className="space-y-6">
          {sections.map((section, idx) => (
            <div key={section.id} className="card">
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-3">
                  <span className="text-sm font-mono text-text-muted">#{idx + 1}</span>
                  <h3 className="text-lg font-semibold text-text-primary">
                    {section.section_name}
                  </h3>
                  <span
                    className={`badge ${section.section_type === 'coding' ? 'badge-success' : 'badge-warning'}`}
                  >
                    {section.section_type === 'coding' ? 'Coding' : 'Quiz'}
                  </span>
                  {section.section_type === 'quiz' && (
                    <>
                      <span className="text-xs text-text-secondary">
                        {section.contest_quiz?.questions?.length || 0} questions
                      </span>
                      <span className="text-xs text-text-secondary">
                        {section.contest_quiz?.questions?.length || 0} marks
                      </span>
                      {section.duration_minutes > 0 && (
                        <span className="text-xs text-text-muted">
                          {section.duration_minutes} min
                        </span>
                      )}
                    </>
                  )}
                  {section.section_type === 'coding' && section.duration_minutes > 0 && (
                    <span className="text-xs text-text-muted">{section.duration_minutes} min</span>
                  )}
                </div>
                <div className="flex gap-2">
                  {section.section_type === 'coding' ? (
                    <button
                      className="btn btn-primary text-sm px-3 py-1.5"
                      onClick={() => openAddProblem(section.id)}
                    >
                      + Add Problem
                    </button>
                  ) : (
                    <button
                      className="btn btn-primary text-sm px-3 py-1.5"
                      onClick={() => openQuizModal(section)}
                    >
                      {section.contest_quiz?.questions && section.contest_quiz.questions.length > 0
                        ? 'Edit Questions'
                        : '+ Add Questions'}
                    </button>
                  )}
                </div>
              </div>

              {section.description && (
                <p className="text-sm text-text-secondary mb-4">{section.description}</p>
              )}

              {/* Coding section content */}
              {section.section_type === 'coding' && (
                <div>
                  {!section.contest_problems || section.contest_problems.length === 0 ? (
                    <div className="text-center py-8 bg-background-tertiary rounded-lg border border-dashed border-background-border">
                      <p className="text-text-muted text-sm mb-3">
                        No problems in this section yet
                      </p>
                      <button
                        className="btn btn-secondary text-sm px-3 py-1.5"
                        onClick={() => openAddProblem(section.id)}
                      >
                        + Add Problem
                      </button>
                    </div>
                  ) : (
                    <div className="card p-0 overflow-hidden">
                      <table className="w-full">
                        <thead>
                          <tr className="border-b border-border-light bg-bg-secondary">
                            <th className="text-left text-sm font-medium text-text-muted px-4 py-3 w-8">
                              #
                            </th>
                            <th className="text-left text-sm font-medium text-text-muted px-4 py-3">
                              Title
                            </th>
                            <th className="text-left text-sm font-medium text-text-muted px-4 py-3">
                              Difficulty
                            </th>
                            <th className="text-right text-sm font-medium text-text-muted px-4 py-3">
                              Points
                            </th>
                            <th className="text-right text-sm font-medium text-text-muted px-4 py-3">
                              Actions
                            </th>
                          </tr>
                        </thead>
                        <tbody>
                          {section.contest_problems.map((cp, pIdx) => (
                            <tr
                              key={cp.id || pIdx}
                              className="border-b border-border-light last:border-0 hover:bg-bg-secondary transition-colors"
                            >
                              <td className="px-4 py-3 text-text-muted text-sm">{pIdx + 1}</td>
                              <td className="px-4 py-3">
                                <div className="font-medium text-text-primary">
                                  {cp.problem?.title || 'Unknown Problem'}
                                </div>
                                {cp.problem?.description && (
                                  <div className="text-xs text-text-muted mt-0.5 max-w-xs truncate">
                                    {cp.problem.description}
                                  </div>
                                )}
                              </td>
                              <td className="px-4 py-3">
                                <span
                                  className={`badge ${getDifficultyClass(cp.problem?.difficulty)}`}
                                >
                                  {cp.problem?.difficulty || '—'}
                                </span>
                              </td>
                              <td className="px-4 py-3 text-right">
                                <span className="font-semibold text-accent-primary">
                                  {cp.points ?? 100}
                                </span>
                              </td>
                              <td className="px-4 py-3 text-right">
                                <button
                                  onClick={() => handleRemoveProblem(String(cp.problem_id))}
                                  className="btn btn-danger px-3 py-1 text-sm"
                                >
                                  Remove
                                </button>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              )}

              {/* Quiz section content */}
              {section.section_type === 'quiz' && (
                <div>
                  {!section.contest_quiz ||
                  !section.contest_quiz.questions ||
                  section.contest_quiz.questions.length === 0 ? (
                    <div className="text-center py-8 bg-background-tertiary rounded-lg border border-dashed border-background-border">
                      <p className="text-text-muted text-sm mb-3">
                        No questions in this section yet
                      </p>
                      <button
                        className="btn btn-secondary text-sm px-3 py-1.5"
                        onClick={() => openQuizModal(section)}
                      >
                        + Add Questions
                      </button>
                    </div>
                  ) : (
                    <div className="bg-background-tertiary rounded-lg border border-background-border p-4">
                      <div className="space-y-2">
                        {section.contest_quiz.questions.map((q, qIdx) => (
                          <div key={q.id || qIdx} className="flex items-center gap-3 text-sm">
                            <span className="text-text-muted w-6">{qIdx + 1}.</span>
                            <span className="text-text-primary flex-1 truncate">
                              {q.question_text}
                            </span>
                            <span className="text-text-muted text-xs">1 mark</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      ) : (
        /* Legacy flat problem list (no sections) */
        <>
          {legacyProblems.length === 0 ? (
            <div className="card text-center py-16">
              <div className="text-6xl mb-4">🧩</div>
              <h3 className="text-xl font-semibold text-text-primary mb-2">No problems yet</h3>
              <p className="text-text-secondary mb-6">
                Add problems to this contest to get started
              </p>
              <button className="btn btn-primary" onClick={() => openAddProblem()}>
                + Add Problem
              </button>
            </div>
          ) : (
            <div className="card p-0 overflow-hidden">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-border-light bg-bg-secondary">
                    <th className="text-left text-sm font-medium text-text-muted px-4 py-3 w-8">
                      #
                    </th>
                    <th className="text-left text-sm font-medium text-text-muted px-4 py-3">
                      Title
                    </th>
                    <th className="text-left text-sm font-medium text-text-muted px-4 py-3">
                      Difficulty
                    </th>
                    <th className="text-left text-sm font-medium text-text-muted px-4 py-3">
                      Tags
                    </th>
                    <th className="text-right text-sm font-medium text-text-muted px-4 py-3">
                      Points
                    </th>
                    <th className="text-right text-sm font-medium text-text-muted px-4 py-3">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {legacyProblems.map((problem, index) => (
                    <tr
                      key={problem.problem_id}
                      className="border-b border-border-light last:border-0 hover:bg-bg-secondary transition-colors"
                    >
                      <td className="px-4 py-3 text-text-muted text-sm">{index + 1}</td>
                      <td className="px-4 py-3">
                        <div className="font-medium text-text-primary">{problem.title}</div>
                        {problem.description && (
                          <div className="text-xs text-text-muted mt-0.5 max-w-xs truncate">
                            {problem.description}
                          </div>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <span className={`badge ${getDifficultyClass(problem.difficulty)}`}>
                          {problem.difficulty || '—'}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        {problem.tags ? (
                          <div className="flex gap-1 flex-wrap">
                            {problem.tags
                              .split(',')
                              .slice(0, 2)
                              .map((tag, i) => (
                                <span key={i} className="badge badge-neutral text-xs">
                                  {tag.trim()}
                                </span>
                              ))}
                            {problem.tags.split(',').length > 2 && (
                              <span className="badge badge-neutral text-xs">
                                +{problem.tags.split(',').length - 2}
                              </span>
                            )}
                          </div>
                        ) : (
                          <span className="text-text-muted">—</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <span className="font-semibold text-accent-primary">
                          {problem.points ?? 100}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-right">
                        <button
                          onClick={() => handleRemoveProblem(problem.problem_id)}
                          className="btn btn-danger px-3 py-1 text-sm"
                        >
                          Remove
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {showCreateModal && (
        <CreateProblemModal
          onClose={() => {
            setShowCreateModal(false);
            setActiveSectionId(undefined);
          }}
          onSuccess={handleCreateSuccess}
          contestId={numericContestId}
          contestPoints={100}
          sectionId={activeSectionId}
        />
      )}

      {showQuizModal && quizModalSection && (
        <ContestQuizModal
          contestId={numericContestId}
          sectionId={quizModalSection.id}
          sectionName={quizModalSection.section_name}
          onClose={() => {
            setShowQuizModal(false);
            setQuizModalSection(null);
          }}
          onSuccess={handleQuizSuccess}
        />
      )}
    </div>
  );
};

export default AdminContestProblems;
