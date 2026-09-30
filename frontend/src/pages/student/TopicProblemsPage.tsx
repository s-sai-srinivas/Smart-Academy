import { useState, useEffect, useCallback } from 'react';
import { useParams, Link } from 'react-router-dom';
import { coursesAPI } from '../../services/api';
import { showError } from '../../utils/showAlert';
import Breadcrumb from '../../components/common/Breadcrumb';

interface SessionData {
  topic_name?: string;
  session_name?: string;
  description?: string;
}

interface ProblemItem {
  id: number;
  title: string;
  difficulty: string;
  is_completed?: boolean;
  time_taken_seconds?: number;
  description?: string;
  tags?: string;
}

function TopicProblemsPage() {
  const { courseId, topicId } = useParams<{ courseId: string; topicId: string }>();
  const [session, setSession] = useState<SessionData | null>(null);
  const [problems, setProblems] = useState<ProblemItem[]>([]);
  const [loading, setLoading] = useState(true);

  const loadTopicProblems = useCallback(async () => {
    if (!topicId) return;
    try {
      const data = (await coursesAPI.getTopicProblems(topicId)) as {
        session?: SessionData;
        problems?: ProblemItem[];
      };
      setSession(data.session || null);
      setProblems(data.problems || []);
    } catch (err) {
      console.error('Failed to load problems:', err);
      showError('Failed to load problems');
    } finally {
      setLoading(false);
    }
  }, [topicId]);

  useEffect(() => {
    loadTopicProblems();
  }, [loadTopicProblems]);

  const getDifficultyClass = (difficulty: string) => {
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

  const formatSolveTime = (seconds: number) => {
    if (!seconds || seconds <= 0) return null;
    const wholeSeconds = Math.floor(seconds);
    const hrs = Math.floor(wholeSeconds / 3600);
    const mins = Math.floor((wholeSeconds % 3600) / 60);
    const secs = wholeSeconds % 60;
    if (hrs > 0) {
      return `${hrs}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    }
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <p className="text-text-muted">Loading lab session...</p>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-4xl mx-auto">
      {/* Breadcrumb Navigation */}
      <Breadcrumb
        items={[
          { label: 'Dashboard', to: '/dashboard' },
          { label: 'Courses', to: '/courses' },
          { label: 'Course', to: `/courses/${courseId}` },
          { label: session?.topic_name || session?.session_name || 'Lab Session' },
        ]}
      />

      {/* Lab Session Header */}
      <div className="mb-8">
        <div className="flex items-center justify-between mb-4">
          <h1 className="text-3xl font-bold text-text-primary">
            {session?.topic_name || session?.session_name || 'Lab Session'}
          </h1>
          <span className="badge badge-info">{problems.length} Problems</span>
        </div>
        {session?.description && <p className="text-text-secondary mb-4">{session.description}</p>}
      </div>

      {/* Problems List */}
      {problems.length === 0 ? (
        <div className="card text-center py-16">
          <div className="text-6xl mb-4">📝</div>
          <h3 className="text-xl font-semibold text-text-primary mb-2">No Problems Yet</h3>
          <p className="text-text-secondary">No problems in this lab session yet.</p>
        </div>
      ) : (
        <div className="space-y-3">
          {problems.map((problem) => (
            <Link
              key={problem.id}
              to={`/lab/${topicId}/problem/${problem.id}`}
              state={{ courseId }}
              className="block card hover:border-accent-secondary/50 transition-all"
            >
              <div className="flex items-center gap-4">
                {/* Solved Tick Mark */}
                {problem.is_completed && (
                  <div className="flex-shrink-0">
                    <svg
                      width="24"
                      height="24"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="#10b981"
                      strokeWidth="3"
                    >
                      <polyline points="20 6 9 17 4 12"></polyline>
                    </svg>
                  </div>
                )}
                <div className="flex-1">
                  <div className="flex items-center gap-3 mb-2">
                    <h3 className="text-lg font-semibold text-text-primary hover:text-accent-secondary transition-colors">
                      {problem.title}
                    </h3>
                    <span className={`badge ${getDifficultyClass(problem.difficulty)}`}>
                      {problem.difficulty}
                    </span>
                    {problem.is_completed && (
                      <span className="badge badge-success text-xs">Solved</span>
                    )}
                    {problem.is_completed &&
                      problem.time_taken_seconds &&
                      problem.time_taken_seconds > 0 && (
                        <span className="badge badge-info text-xs">
                          {formatSolveTime(problem.time_taken_seconds)}
                        </span>
                      )}
                  </div>
                  {problem.description && (
                    <p className="text-text-secondary text-sm line-clamp-2">
                      {problem.description}
                    </p>
                  )}
                  {problem.tags && (
                    <div className="flex gap-2 mt-2">
                      {problem.tags
                        .split(',')
                        .slice(0, 3)
                        .map((tag, i) => (
                          <span key={i} className="badge badge-neutral text-xs">
                            {tag.trim()}
                          </span>
                        ))}
                    </div>
                  )}
                </div>
                <div className="text-accent-secondary text-xl">→</div>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

export default TopicProblemsPage;
