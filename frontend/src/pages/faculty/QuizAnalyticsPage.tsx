import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { quizAPI } from '../../services/api';
import { showError } from '../../utils/showAlert';
import { ChevronLeft, Users, CheckCircle, Award, Clock } from 'lucide-react';

interface QuizInfo {
  title: string;
  total_marks?: number;
}

interface AttemptStats {
  total_attempts?: number;
  pass_rate?: number;
  average_score?: number;
  average_time_minutes?: number;
}

interface QuestionStat {
  question_text: string;
  question_type: string;
  difficulty: string;
  marks: number;
  correct_percentage: number;
}

interface DifficultyResult {
  correct_percentage: number;
  correct: number;
  total: number;
}

interface TopPerformer {
  student_name: string;
  student_regdno: string;
  marks_obtained: number;
  percentage: number;
}

interface Attempt {
  student_name: string;
  student_regdno: string;
  attempt_number: number;
  marks_obtained: number;
  percentage: number;
  time_taken_minutes?: number;
  status: string;
}

interface AnalyticsData {
  quiz?: QuizInfo;
  attempt_stats?: AttemptStats;
  question_stats?: QuestionStat[];
  difficulty_results?: {
    easy?: DifficultyResult;
    medium?: DifficultyResult;
    hard?: DifficultyResult;
  };
  top_performers?: TopPerformer[];
  attempts?: Attempt[];
}

function QuizAnalyticsPage() {
  const { quizId } = useParams<{ quizId: string }>();
  const numericQuizId = Number(quizId);
  const navigate = useNavigate();
  const [loading, setLoading] = useState<boolean>(true);
  const [analytics, setAnalytics] = useState<AnalyticsData | null>(null);
  const [selectedTab, setSelectedTab] = useState<string>('overview');

  const loadAnalytics = useCallback(async () => {
    try {
      const data = (await quizAPI.getAnalytics(numericQuizId)) as AnalyticsData;
      setAnalytics(data);
    } catch (err) {
      console.error('Failed to load analytics:', err);
      showError('Failed to load analytics');
    } finally {
      setLoading(false);
    }
  }, [numericQuizId]);

  useEffect(() => {
    loadAnalytics();
  }, [loadAnalytics]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <div className="w-8 h-8 border-2 border-background-border border-t-gray-500 rounded-full animate-spin mx-auto mb-3" />
          <p className="text-text-muted">Loading analytics...</p>
        </div>
      </div>
    );
  }

  if (!analytics) {
    return (
      <div className="text-center py-12">
        <p className="text-text-muted">Failed to load analytics</p>
      </div>
    );
  }

  const { quiz, attempt_stats, question_stats, difficulty_results, top_performers } = analytics;

  return (
    <div className="space-y-6 max-w-7xl">
      {/* Header */}
      <div className="flex items-center gap-4">
        <button
          onClick={() => navigate('/faculty/quizzes')}
          className="p-2 hover:bg-background-tertiary rounded-lg transition-colors"
        >
          <ChevronLeft className="w-5 h-5 text-text-secondary" />
        </button>
        <div>
          <h1 className="text-3xl font-bold text-text-primary mb-2">Quiz Analytics</h1>
          <p className="text-text-secondary">{quiz?.title}</p>
        </div>
      </div>

      {/* Stats Overview */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-background-secondary rounded-xl border-2 border-background-border p-6">
          <div className="flex items-center justify-between mb-2">
            <Users className="w-5 h-5 text-blue-400" />
            <span className="text-text-muted text-sm">Total Attempts</span>
          </div>
          <div className="text-3xl font-bold text-text-primary">
            {attempt_stats?.total_attempts || 0}
          </div>
        </div>
        <div className="bg-background-secondary rounded-xl border-2 border-background-border p-6">
          <div className="flex items-center justify-between mb-2">
            <CheckCircle className="w-5 h-5 text-green-400" />
            <span className="text-text-muted text-sm">Pass Rate</span>
          </div>
          <div className="text-3xl font-bold text-text-primary">
            {attempt_stats?.pass_rate || 0}%
          </div>
        </div>
        <div className="bg-background-secondary rounded-xl border-2 border-background-border p-6">
          <div className="flex items-center justify-between mb-2">
            <Award className="w-5 h-5 text-yellow-400" />
            <span className="text-text-muted text-sm">Average Score</span>
          </div>
          <div className="text-3xl font-bold text-text-primary">
            {attempt_stats?.average_score || 0}/{quiz?.total_marks}
          </div>
        </div>
        <div className="bg-background-secondary rounded-xl border-2 border-background-border p-6">
          <div className="flex items-center justify-between mb-2">
            <Clock className="w-5 h-5 text-purple-400" />
            <span className="text-text-muted text-sm">Avg Time Taken</span>
          </div>
          <div className="text-3xl font-bold text-text-primary">
            {attempt_stats?.average_time_minutes || 0} min
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 border-b border-background-border">
        <button
          onClick={() => setSelectedTab('overview')}
          className={`px-4 py-2 font-medium transition-colors ${
            selectedTab === 'overview'
              ? 'text-blue-400 border-b-2 border-blue-500'
              : 'text-text-secondary hover:text-text-primary'
          }`}
        >
          Overview
        </button>
        <button
          onClick={() => setSelectedTab('questions')}
          className={`px-4 py-2 font-medium transition-colors ${
            selectedTab === 'questions'
              ? 'text-blue-400 border-b-2 border-blue-500'
              : 'text-text-secondary hover:text-text-primary'
          }`}
        >
          Question Analysis
        </button>
        <button
          onClick={() => setSelectedTab('attempts')}
          className={`px-4 py-2 font-medium transition-colors ${
            selectedTab === 'attempts'
              ? 'text-blue-400 border-b-2 border-blue-500'
              : 'text-text-secondary hover:text-text-primary'
          }`}
        >
          Student Attempts
        </button>
      </div>

      {/* Tab Content */}
      {selectedTab === 'overview' && (
        <div className="space-y-6">
          {/* Difficulty Results */}
          <div className="bg-background-secondary rounded-xl border-2 border-background-border p-6">
            <h3 className="text-lg font-semibold text-text-primary mb-4">
              Performance by Difficulty
            </h3>
            <div className="grid grid-cols-3 gap-4">
              {difficulty_results?.easy && (
                <div className="text-center p-4 bg-green-500/10 rounded-lg border border-green-500/20">
                  <div className="text-green-400 font-medium mb-2">Easy</div>
                  <div className="text-2xl font-bold text-text-primary">
                    {difficulty_results.easy.correct_percentage}%
                  </div>
                  <div className="text-sm text-text-secondary">
                    {difficulty_results.easy.correct} / {difficulty_results.easy.total} correct
                  </div>
                </div>
              )}
              {difficulty_results?.medium && (
                <div className="text-center p-4 bg-yellow-500/10 rounded-lg border border-yellow-500/20">
                  <div className="text-yellow-400 font-medium mb-2">Medium</div>
                  <div className="text-2xl font-bold text-text-primary">
                    {difficulty_results.medium.correct_percentage}%
                  </div>
                  <div className="text-sm text-text-secondary">
                    {difficulty_results.medium.correct} / {difficulty_results.medium.total} correct
                  </div>
                </div>
              )}
              {difficulty_results?.hard && (
                <div className="text-center p-4 bg-red-500/10 rounded-lg border border-red-500/20">
                  <div className="text-red-400 font-medium mb-2">Hard</div>
                  <div className="text-2xl font-bold text-text-primary">
                    {difficulty_results.hard.correct_percentage}%
                  </div>
                  <div className="text-sm text-text-secondary">
                    {difficulty_results.hard.correct} / {difficulty_results.hard.total} correct
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Top Performers */}
          <div className="bg-background-secondary rounded-xl border-2 border-background-border p-6">
            <h3 className="text-lg font-semibold text-text-primary mb-4">Top Performers</h3>
            {top_performers && top_performers.length > 0 ? (
              <div className="space-y-3">
                {top_performers.map((student, index) => (
                  <div
                    key={index}
                    className="flex items-center justify-between p-4 bg-background-tertiary rounded-lg"
                  >
                    <div className="flex items-center gap-3">
                      <div
                        className={`w-8 h-8 rounded-full flex items-center justify-center font-bold ${
                          index === 0
                            ? 'bg-yellow-500/20 text-yellow-400'
                            : index === 1
                              ? 'bg-background-tertiary text-text-secondary'
                              : index === 2
                                ? 'bg-orange-500/20 text-orange-400'
                                : 'bg-background-tertiary text-text-muted'
                        }`}
                      >
                        {index + 1}
                      </div>
                      <div>
                        <div className="font-medium text-text-primary">{student.student_name}</div>
                        <div className="text-sm text-text-secondary">{student.student_regdno}</div>
                      </div>
                    </div>
                    <div className="text-right">
                      <div className="text-xl font-bold text-text-primary">
                        {student.marks_obtained}/{quiz?.total_marks}
                      </div>
                      <div className="text-sm text-text-secondary">{student.percentage}%</div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-text-muted text-center py-8">No attempts yet</p>
            )}
          </div>
        </div>
      )}

      {selectedTab === 'questions' && (
        <div className="bg-background-secondary rounded-xl border-2 border-background-border">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-background-border">
                  <th className="text-left py-3 px-4 text-sm font-medium text-text-secondary">
                    Question
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-text-secondary">
                    Type
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-text-secondary">
                    Difficulty
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-text-secondary">
                    Marks
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-text-secondary">
                    Correct %
                  </th>
                </tr>
              </thead>
              <tbody>
                {question_stats && question_stats.length > 0 ? (
                  question_stats.map((q, index) => (
                    <tr key={index} className="border-b border-background-border">
                      <td className="py-4 px-4">
                        <div className="max-w-md truncate text-text-primary">{q.question_text}</div>
                      </td>
                      <td className="py-4 px-4">
                        <span className="px-2.5 py-1 bg-background-tertiary text-text-secondary rounded-md text-xs">
                          {q.question_type}
                        </span>
                      </td>
                      <td className="py-4 px-4">
                        <span
                          className={`px-2.5 py-1 rounded-md text-xs font-medium border ${
                            q.difficulty === 'easy'
                              ? 'bg-green-500/10 text-green-400 border-green-500/20'
                              : q.difficulty === 'medium'
                                ? 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20'
                                : 'bg-red-500/10 text-red-400 border-red-500/20'
                          }`}
                        >
                          {q.difficulty}
                        </span>
                      </td>
                      <td className="py-4 px-4 text-text-secondary">{q.marks}</td>
                      <td className="py-4 px-4">
                        <div className="flex items-center gap-2">
                          <div className="flex-1 h-2 bg-slate-700 rounded-full overflow-hidden">
                            <div
                              className="h-full bg-blue-500 rounded-full"
                              style={{ width: `${q.correct_percentage}%` }}
                            />
                          </div>
                          <span className="text-text-primary font-medium">
                            {q.correct_percentage}%
                          </span>
                        </div>
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={5} className="py-8 text-center text-text-muted">
                      No question data available
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {selectedTab === 'attempts' && (
        <div className="bg-background-secondary rounded-xl border-2 border-background-border">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-background-border">
                  <th className="text-left py-3 px-4 text-sm font-medium text-text-secondary">
                    Student
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-text-secondary">
                    Attempt #
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-text-secondary">
                    Score
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-text-secondary">
                    Percentage
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-text-secondary">
                    Time Taken
                  </th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-text-secondary">
                    Status
                  </th>
                </tr>
              </thead>
              <tbody>
                {analytics.attempts && analytics.attempts.length > 0 ? (
                  analytics.attempts.map((attempt, index) => (
                    <tr
                      key={index}
                      className="border-b border-background-border hover:bg-background-tertiary/50"
                    >
                      <td className="py-4 px-4">
                        <div>
                          <div className="font-medium text-text-primary">
                            {attempt.student_name}
                          </div>
                          <div className="text-sm text-text-muted">{attempt.student_regdno}</div>
                        </div>
                      </td>
                      <td className="py-4 px-4 text-text-secondary">#{attempt.attempt_number}</td>
                      <td className="py-4 px-4 text-text-primary font-medium">
                        {attempt.marks_obtained}/{quiz?.total_marks}
                      </td>
                      <td className="py-4 px-4">
                        <span
                          className={`px-2.5 py-1 rounded-md text-xs font-medium ${
                            attempt.percentage >= 80
                              ? 'bg-green-500/10 text-green-400'
                              : attempt.percentage >= 60
                                ? 'bg-blue-500/10 text-blue-400'
                                : attempt.percentage >= 40
                                  ? 'bg-yellow-500/10 text-yellow-400'
                                  : 'bg-red-500/10 text-red-400'
                          }`}
                        >
                          {attempt.percentage}%
                        </span>
                      </td>
                      <td className="py-4 px-4 text-text-secondary">
                        {attempt.time_taken_minutes || 0} min
                      </td>
                      <td className="py-4 px-4">
                        <span
                          className={`px-2.5 py-1 rounded-md text-xs font-medium border ${
                            attempt.status === 'passed'
                              ? 'bg-green-500/10 text-green-400 border-green-500/20'
                              : 'bg-red-500/10 text-red-400 border-red-500/20'
                          }`}
                        >
                          {attempt.status}
                        </span>
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={6} className="py-8 text-center text-text-muted">
                      No attempts yet
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}

export default QuizAnalyticsPage;
