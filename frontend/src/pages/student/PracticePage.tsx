import { useState, useEffect, type ChangeEvent } from 'react';
import { Link } from 'react-router-dom';
import { practiceAPI, referenceAPI } from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import { showError } from '../../utils/showAlert';
import { BookOpen, CheckCircle, Clock, Tag, ChevronRight } from 'lucide-react';

interface Subject {
  id: number;
  name: string;
}

interface PracticeProblem {
  id: number;
  title: string;
  difficulty?: string;
  is_completed?: boolean;
  tags?: string;
  time_limit?: number;
  subject?: Subject;
  subject_id?: number;
}

function PracticePage() {
  const { user: _user } = useAuth();
  const [problems, setProblems] = useState<PracticeProblem[]>([]);
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [selectedSubject, setSelectedSubject] = useState<number | null>(null);

  useEffect(() => {
    loadProblems();
    loadSubjects();
  }, []);

  const loadProblems = async () => {
    try {
      setLoading(true);
      const data = (await practiceAPI.getProblems()) as {
        problems?: PracticeProblem[];
        subjects?: Subject[];
      };
      setProblems(data.problems || []);
      if (data.subjects) {
        setSubjects(data.subjects);
      }
    } catch (err) {
      showError((err as Error).message || 'Failed to load practice problems');
    } finally {
      setLoading(false);
    }
  };

  const loadSubjects = async () => {
    try {
      const data = (await referenceAPI.getSubjects(false)) as Subject[];
      setSubjects(Array.isArray(data) ? data : []);
    } catch {
      // Silently fail - subjects may already be loaded with problems
    }
  };

  const getDifficultyClass = (difficulty?: string) => {
    switch (difficulty) {
      case 'easy':
        return 'text-green-400 bg-green-400/10';
      case 'medium':
        return 'text-yellow-400 bg-yellow-400/10';
      case 'hard':
        return 'text-red-400 bg-red-400/10';
      default:
        return 'text-gray-400 bg-gray-400/10';
    }
  };

  const filteredProblems = problems.filter((p) => {
    if (filter === 'solved' && !p.is_completed) return false;
    if (filter === 'unsolved' && p.is_completed) return false;
    if (selectedSubject !== null) {
      if (p.subject_id !== selectedSubject && p.subject?.id !== selectedSubject) return false;
    }
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      return p.title?.toLowerCase().includes(q) || p.tags?.toLowerCase().includes(q);
    }
    return true;
  });

  const solvedCount = problems.filter((p) => p.is_completed).length;

  return (
    <div className="space-y-6 max-w-6xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary mb-1">Practice Problems</h1>
          <p className="text-text-secondary text-sm">
            Sharpen your coding skills with global practice problems
          </p>
        </div>
        <div className="flex items-center gap-4 text-sm">
          <div className="flex items-center gap-2 px-3 py-1.5 bg-green-500/10 text-green-400 rounded-lg">
            <CheckCircle className="w-4 h-4" />
            <span>{solvedCount} Solved</span>
          </div>
          <div className="flex items-center gap-2 px-3 py-1.5 bg-blue-500/10 text-blue-400 rounded-lg">
            <BookOpen className="w-4 h-4" />
            <span>{problems.length} Total</span>
          </div>
        </div>
      </div>

      {/* Subject Tabs */}
      {subjects.length > 0 && (
        <div className="flex flex-wrap gap-2">
          <button
            onClick={() => setSelectedSubject(null)}
            className={`px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
              selectedSubject === null
                ? 'bg-accent-secondary text-white'
                : 'bg-background-secondary text-text-secondary hover:text-text-primary border border-background-border'
            }`}
          >
            All Subjects
          </button>
          {subjects.map((subject) => (
            <button
              key={subject.id}
              onClick={() => setSelectedSubject(subject.id)}
              className={`px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
                selectedSubject === subject.id
                  ? 'bg-accent-secondary text-white'
                  : 'bg-background-secondary text-text-secondary hover:text-text-primary border border-background-border'
              }`}
            >
              {subject.name}
            </button>
          ))}
        </div>
      )}

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-3">
        <div className="relative flex-1">
          <input
            type="text"
            placeholder="Search problems by title or tag..."
            value={searchQuery}
            onChange={(e: ChangeEvent<HTMLInputElement>) => setSearchQuery(e.target.value)}
            className="w-full px-4 py-2.5 bg-background-secondary border border-background-border rounded-lg text-text-primary placeholder-text-tertiary text-sm focus:outline-none focus:border-accent-secondary"
          />
        </div>
        <div className="flex gap-2">
          {['all', 'solved', 'unsolved'].map((f) => (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className={`px-4 py-2 text-sm font-medium rounded-lg transition-colors capitalize ${
                filter === f
                  ? 'bg-accent-secondary text-white'
                  : 'bg-background-secondary text-text-secondary hover:text-text-primary border border-background-border'
              }`}
            >
              {f}
            </button>
          ))}
        </div>
      </div>

      {/* Problems List */}
      {loading ? (
        <div className="text-center py-12 text-text-secondary">Loading problems...</div>
      ) : filteredProblems.length === 0 ? (
        <div className="text-center py-12">
          <BookOpen className="w-12 h-12 mx-auto text-text-muted mb-3" />
          <p className="text-text-secondary">No practice problems found</p>
          {problems.length === 0 && (
            <p className="text-text-tertiary text-sm mt-2">Check back later for new problems!</p>
          )}
        </div>
      ) : (
        <div className="space-y-3">
          {filteredProblems.map((problem, index) => (
            <Link
              key={problem.id}
              to={`/practice/${problem.id}`}
              className="block card-elevated hover:border-accent-secondary/30 transition-colors group"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4 min-w-0 flex-1">
                  <div className="flex items-center gap-2 min-w-0">
                    {problem.is_completed && (
                      <CheckCircle className="w-5 h-5 text-green-400 shrink-0" />
                    )}
                    <span className="text-text-tertiary text-sm font-mono w-8">{index + 1}.</span>
                  </div>
                  <div className="min-w-0">
                    <h3 className="text-text-primary font-medium group-hover:text-accent-secondary transition-colors truncate">
                      {problem.title}
                    </h3>
                    <div className="flex items-center gap-3 mt-1">
                      <span
                        className={`text-xs font-medium px-2 py-0.5 rounded ${getDifficultyClass(problem?.difficulty)}`}
                      >
                        {problem?.difficulty
                          ? problem.difficulty.charAt(0).toUpperCase() + problem.difficulty.slice(1)
                          : ''}
                      </span>
                      {problem.tags && (
                        <div className="flex items-center gap-1 text-text-muted text-xs">
                          <Tag className="w-3 h-3" />
                          <span className="truncate max-w-[200px]">{problem.tags}</span>
                        </div>
                      )}
                      {problem.time_limit && (
                        <div className="flex items-center gap-1 text-text-muted text-xs">
                          <Clock className="w-3 h-3" />
                          <span>{problem.time_limit}ms</span>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
                <ChevronRight className="w-5 h-5 text-text-muted group-hover:text-accent-secondary transition-colors shrink-0" />
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

export default PracticePage;
