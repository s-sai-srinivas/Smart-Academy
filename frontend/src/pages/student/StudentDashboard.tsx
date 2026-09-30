import { useState, useEffect } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import { dashboardAPI } from '../../services/api';
import { showError } from '../../utils/showAlert';
import ViewAllTopics from '../../components/student/ViewAllTopics';
import { BookOpen, ChevronRight } from 'lucide-react';
import './StudentDashboard.css';

interface DashboardStats {
  totalSolved: number;
  totalProblems: number;
  currentStreak: number;
  accuracy: number;
  accuracyChange: number;
  collegeRank: number;
  totalPoints: number;
}

interface TopicItem {
  name: string;
  progress: number;
  easy: number;
  medium: number;
  hard: number;
}

interface ActivityDay {
  date: string;
  count: number;
}

interface LeaderboardEntry {
  rank: number;
  regdno?: string;
  name: string;
  solved: number;
  points: number;
}

interface CourseItem {
  id: number;
  course_code: string;
  course_name: string;
  course_type: string;
  credits: number;
}

function StudentDashboard() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [showTopicsSidebar, setShowTopicsSidebar] = useState(false);

  const [stats, setStats] = useState<DashboardStats>({
    totalSolved: 0,
    totalProblems: 0,
    currentStreak: 0,
    accuracy: 0,
    accuracyChange: 0,
    collegeRank: 1,
    totalPoints: 0,
  });

  const [topics, setTopics] = useState<TopicItem[]>([]);
  const [activity, setActivity] = useState<ActivityDay[]>([]);
  const [leaderboard, setLeaderboard] = useState<LeaderboardEntry[]>([]);
  const [userRank, setUserRank] = useState(0);
  const [courses, setCourses] = useState<CourseItem[]>([]);
  const [showFullLeaderboard, setShowFullLeaderboard] = useState(false);
  const [leaderboardPage, setLeaderboardPage] = useState(0);
  const LEADERBOARD_PAGE_SIZE = 10;

  useEffect(() => {
    if (user) {
      loadDashboardData();
    } else {
      setLoading(false);
    }
  }, [user]);

  const loadDashboardData = async () => {
    try {
      const data = (await dashboardAPI.getData()) as {
        stats?: {
          total_solved?: number;
          total_problems?: number;
          current_streak?: number;
          accuracy?: number;
          accuracy_change?: number;
          college_rank?: number;
          total_points?: number;
        };
        topic_proficiency?: TopicItem[];
        activity?: ActivityDay[];
        leaderboard?: LeaderboardEntry[];
        current_user_rank?: number;
        courses?: CourseItem[];
      };

      setStats({
        totalSolved: data.stats?.total_solved || 0,
        totalProblems: data.stats?.total_problems || 0,
        currentStreak: data.stats?.current_streak || 0,
        accuracy: data.stats?.accuracy || 0,
        accuracyChange: data.stats?.accuracy_change || 0,
        collegeRank: data.stats?.college_rank || 1,
        totalPoints: data.stats?.total_points || 0,
      });

      setTopics(data.topic_proficiency || []);
      setActivity(data.activity || []);
      setLeaderboard(data.leaderboard || []);
      setUserRank(data.current_user_rank || 0);
      setCourses(data.courses || []);
    } catch (err) {
      console.error('Failed to load dashboard:', err);
      showError('Failed to load dashboard data');
    } finally {
      setLoading(false);
    }
  };

  if (!user) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <h2 className="text-2xl font-semibold text-text-primary mb-2">Welcome to CodePlatform</h2>
          <p className="text-text-muted">Please login to view your dashboard</p>
        </div>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <p className="text-text-muted">Loading dashboard...</p>
      </div>
    );
  }

  return (
    <>
      <div className="p-6">
        {/* Stats Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5 mb-6">
          <div className="card">
            <div className="flex items-center justify-between mb-3">
              <span className="text-text-tertiary text-sm font-medium">TOTAL SOLVED</span>
              <span className="text-accent-success text-xl">✔</span>
            </div>
            <div className="text-3xl font-bold text-text-primary">
              {stats.totalSolved}{' '}
              <span className="text-lg text-text-tertiary">/ {stats.totalProblems}</span>
            </div>
          </div>

          <div className="card">
            <div className="flex items-center justify-between mb-3">
              <span className="text-text-tertiary text-sm font-medium">CURRENT STREAK</span>
              <span className="text-accent-primary text-xl">◈</span>
            </div>
            <div className="text-3xl font-bold text-text-primary">
              {stats.currentStreak} <span className="text-lg text-text-tertiary">Days</span>
            </div>
          </div>

          <div className="card">
            <div className="flex items-center justify-between mb-3">
              <span className="text-text-tertiary text-sm font-medium">ACCURACY</span>
              <span className="text-accent-secondary text-xl">◎</span>
            </div>
            <div className="text-3xl font-bold text-text-primary">{stats.accuracy.toFixed(1)}%</div>
            <div className="text-accent-success text-sm mt-1">
              +{stats.accuracyChange.toFixed(1)}% from last week
            </div>
          </div>

          <div className="card">
            <div className="flex items-center justify-between mb-3">
              <span className="text-text-tertiary text-sm font-medium">COLLEGE RANK</span>
              <span className="text-accent-warning text-xl">◆</span>
            </div>
            <div className="text-3xl font-bold text-text-primary">#{stats.collegeRank}</div>
            <div className="text-text-tertiary text-sm mt-1">{stats.totalPoints} points</div>
          </div>
        </div>

        {/* Main Content Grid */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Left Column */}
          <div className="lg:col-span-2 space-y-6">
            {/* Topic Proficiency */}
            <div className="card">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-text-primary">Topic Proficiency</h3>
                <button
                  className="text-accent-secondary text-sm font-medium hover:underline"
                  onClick={() => setShowTopicsSidebar(true)}
                >
                  VIEW ALL TOPICS
                </button>
              </div>
              <div className="space-y-4">
                {topics.length === 0 ? (
                  <p className="text-text-muted">
                    No topics available yet. Solve problems to see your progress!
                  </p>
                ) : (
                  topics.map((topic, index) => (
                    <div key={index} className="space-y-2">
                      <div className="flex items-center justify-between">
                        <span className="font-medium text-text-primary">{topic.name}</span>
                      </div>
                      <div className="flex items-center gap-3">
                        <div className="flex-1 h-2 bg-background-tertiary rounded-full overflow-hidden">
                          <div
                            className="h-full bg-accent-primary rounded-full transition-all"
                            style={{ width: `${topic.progress}%` }}
                          ></div>
                        </div>
                        <span className="text-sm text-text-secondary w-10 text-right">
                          {topic.progress}%
                        </span>
                      </div>
                      <div className="flex gap-3 text-xs">
                        <span className="text-accent-success">{topic.easy} EASY</span>
                        <span className="text-accent-warning">{topic.medium} MED</span>
                        <span className="text-accent-danger">{topic.hard} HARD</span>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>

            {/* Submission Activity */}
            <div className="card">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-text-primary">Submission Streak</h3>
              </div>
              <p className="text-text-secondary text-sm mb-4">
                {activity.reduce((sum, d) => sum + d.count, 0)} submissions in the last year
              </p>

              {activity.length === 0 ? (
                <p className="text-text-muted text-sm">No activity data yet</p>
              ) : (
                <>
                  <div className="activity-heatmap">
                    <div className="heatmap-grid">
                      {(() => {
                        const weeks: (ActivityDay | null)[][] = [];
                        let currentWeek: (ActivityDay | null)[] = [];

                        activity.forEach((day, index) => {
                          const date = new Date(day.date);
                          const dayOfWeek = date.getDay();

                          if (index === 0 && dayOfWeek !== 0) {
                            for (let i = 0; i < dayOfWeek; i++) {
                              currentWeek.push(null);
                            }
                          }

                          currentWeek.push(day);

                          if (dayOfWeek === 6 || index === activity.length - 1) {
                            weeks.push([...currentWeek]);
                            currentWeek = [];
                          }
                        });

                        const getDayClass = (count: number) => {
                          if (count === 0) return 'day-empty';
                          if (count <= 2) return 'day-low';
                          if (count <= 5) return 'day-medium';
                          return 'day-high';
                        };

                        return weeks.map((week, weekIndex) => (
                          <div key={weekIndex} className="heatmap-week">
                            {week.map((day, dayIndex) => (
                              <div
                                key={dayIndex}
                                className={`heatmap-day ${day ? getDayClass(day.count) : 'day-empty'}`}
                                title={day ? `${day.date}: ${day.count} submissions` : ''}
                              />
                            ))}
                          </div>
                        ));
                      })()}
                    </div>
                    <div className="heatmap-legend">
                      <span>Less</span>
                      <div className="legend-item day-empty"></div>
                      <div className="legend-item day-low"></div>
                      <div className="legend-item day-medium"></div>
                      <div className="legend-item day-high"></div>
                      <span>More</span>
                    </div>
                  </div>
                </>
              )}
            </div>

            {/* Current Enrolled Courses */}
            <div className="card">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-text-primary">
                  Current Enrolled Courses
                </h3>
                <button
                  className="text-accent-secondary text-sm font-medium hover:underline"
                  onClick={() => navigate('/courses')}
                >
                  VIEW ALL COURSES
                </button>
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                {courses.length === 0 ? (
                  <p className="text-text-muted">No enrolled courses yet.</p>
                ) : (
                  courses.map((course) => (
                    <div
                      key={course.id}
                      className="flex items-center gap-3 p-3 bg-background-tertiary rounded-lg"
                    >
                      <div className="w-12 h-12 rounded-lg bg-accent-secondary/20 flex items-center justify-center text-accent-secondary font-bold text-xs">
                        {course.course_code}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="font-medium text-text-primary truncate">
                          {course.course_name}
                        </div>
                        <div className="text-sm text-text-secondary truncate">
                          {course.course_type} • {course.credits} Credits
                        </div>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>

          {/* Right Column */}
          <div className="space-y-6">
            {/* Leaderboard */}
            <div className="card">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-text-primary">Leaderboard</h3>
                <span className="px-3 py-1 text-xs font-medium rounded-md bg-accent-primary text-white">
                  College
                </span>
              </div>
              <div className="space-y-2">
                {leaderboard.length === 0 ? (
                  <p className="text-text-muted text-sm">No leaderboard data yet</p>
                ) : (
                  leaderboard.slice(0, 5).map((entry) => (
                    <div
                      key={entry.rank}
                      className={`flex items-center gap-3 p-2 rounded-lg ${
                        entry.regdno === user?.regdno
                          ? 'bg-background-tertiary border border-accent-primary/30'
                          : ''
                      }`}
                    >
                      <span className="w-6 text-center text-text-tertiary text-sm font-medium">
                        {entry.rank}
                      </span>
                      <div className="w-8 h-8 rounded-full bg-gradient-to-br from-accent-secondary to-blue-400"></div>
                      <div className="flex-1 min-w-0">
                        <div className="font-medium text-text-primary text-sm truncate">
                          {entry.name}
                        </div>
                        <div className="text-xs text-text-secondary">
                          {entry.solved} Solved •{' '}
                          {entry.points >= 1000
                            ? `${(entry.points / 1000).toFixed(1)}k`
                            : entry.points}{' '}
                          points
                        </div>
                      </div>
                      {entry.rank === 1 && (
                        <span className="text-xs font-bold text-accent-warning">1st</span>
                      )}
                    </div>
                  ))
                )}

                {userRank > 5 && (
                  <>
                    <div className="text-center text-text-tertiary text-sm">...</div>
                    <div className="flex items-center gap-3 p-2 rounded-lg bg-accent-primary/10 border border-accent-primary/30">
                      <span className="w-6 text-center text-accent-primary text-sm font-medium">
                        {userRank}
                      </span>
                      <div className="w-8 h-8 rounded-full bg-gradient-to-br from-accent-primary to-amber-400"></div>
                      <div className="flex-1 min-w-0">
                        <div className="font-medium text-text-primary text-sm truncate">
                          {user?.name} (You)
                        </div>
                        <div className="text-xs text-text-secondary">
                          {stats.totalSolved} Solved •{' '}
                          {stats.totalPoints >= 1000
                            ? `${(stats.totalPoints / 1000).toFixed(1)}k`
                            : stats.totalPoints}{' '}
                          points
                        </div>
                      </div>
                      <span className="badge badge-warning text-xs">TOP 1%</span>
                    </div>
                  </>
                )}
              </div>
              <button
                className="w-full mt-4 btn btn-secondary text-sm"
                onClick={() => setShowFullLeaderboard(!showFullLeaderboard)}
              >
                {showFullLeaderboard ? 'HIDE FULL LEADERBOARD' : 'VIEW FULL LEADERBOARD'}
              </button>

              {/* Full Leaderboard with Pagination */}
              {showFullLeaderboard && (
                <div className="mt-4 max-h-96 overflow-y-auto">
                  {leaderboard.length === 0 ? (
                    <p className="text-text-muted text-sm text-center py-4">
                      No leaderboard data yet
                    </p>
                  ) : (
                    <>
                      <div className="space-y-2">
                        {leaderboard
                          .slice(
                            leaderboardPage * LEADERBOARD_PAGE_SIZE,
                            (leaderboardPage + 1) * LEADERBOARD_PAGE_SIZE
                          )
                          .map((entry) => (
                            <div
                              key={entry.rank}
                              className={`flex items-center gap-3 p-2 rounded-lg ${
                                entry.regdno === user?.regdno
                                  ? 'bg-background-tertiary border border-accent-primary/30'
                                  : ''
                              }`}
                            >
                              <span className="w-6 text-center text-text-tertiary text-sm font-medium">
                                {entry.rank}
                              </span>
                              <div className="w-8 h-8 rounded-full bg-gradient-to-br from-accent-secondary to-blue-400"></div>
                              <div className="flex-1 min-w-0">
                                <div className="font-medium text-text-primary text-sm truncate">
                                  {entry.name}
                                </div>
                                <div className="text-xs text-text-secondary">
                                  {entry.solved} Solved •{' '}
                                  {entry.points >= 1000
                                    ? `${(entry.points / 1000).toFixed(1)}k`
                                    : entry.points}{' '}
                                  points
                                </div>
                              </div>
                              {entry.rank === 1 && (
                                <span className="text-xs font-bold text-accent-warning">1st</span>
                              )}
                              {entry.rank === 2 && (
                                <span className="text-xs font-bold text-gray-400">2nd</span>
                              )}
                              {entry.rank === 3 && (
                                <span className="text-xs font-bold text-amber-700">3rd</span>
                              )}
                            </div>
                          ))}
                      </div>

                      {/* Pagination Controls */}
                      {leaderboard.length > LEADERBOARD_PAGE_SIZE && (
                        <div className="flex items-center justify-between mt-4 pt-4 border-t border-border">
                          <button
                            className="btn btn-secondary text-xs px-3 py-1"
                            onClick={() => setLeaderboardPage(Math.max(0, leaderboardPage - 1))}
                            disabled={leaderboardPage === 0}
                          >
                            ← Previous
                          </button>
                          <span className="text-text-secondary text-sm">
                            Page {leaderboardPage + 1} of{' '}
                            {Math.ceil(leaderboard.length / LEADERBOARD_PAGE_SIZE)}
                          </span>
                          <button
                            className="btn btn-secondary text-xs px-3 py-1"
                            onClick={() =>
                              setLeaderboardPage(
                                Math.min(
                                  Math.ceil(leaderboard.length / LEADERBOARD_PAGE_SIZE) - 1,
                                  leaderboardPage + 1
                                )
                              )
                            }
                            disabled={
                              leaderboardPage >=
                              Math.ceil(leaderboard.length / LEADERBOARD_PAGE_SIZE) - 1
                            }
                          >
                            Next →
                          </button>
                        </div>
                      )}
                    </>
                  )}
                </div>
              )}
            </div>

            {/* Practice Problems Quick Access */}
            <div className="card">
              <h3 className="text-lg font-semibold text-text-primary mb-4">Practice Problems</h3>
              <p className="text-text-secondary text-sm mb-4">
                Sharpen your coding skills with global practice problems available to all students.
              </p>
              <Link
                to="/practice"
                className="flex items-center justify-between p-3 bg-background-tertiary rounded-lg hover:bg-accent-secondary/10 transition-colors group"
              >
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-lg bg-accent-secondary/20 flex items-center justify-center">
                    <BookOpen className="w-5 h-5 text-accent-secondary" />
                  </div>
                  <div>
                    <div className="font-medium text-text-primary">Practice Now</div>
                    <div className="text-xs text-text-secondary">Solve global problems</div>
                  </div>
                </div>
                <ChevronRight className="w-5 h-5 text-text-muted group-hover:text-accent-secondary transition-colors" />
              </Link>
            </div>
          </div>
        </div>
      </div>

      {/* View All Topics Sidebar */}
      <ViewAllTopics isOpen={showTopicsSidebar} onClose={() => setShowTopicsSidebar(false)} />
    </>
  );
}

export default StudentDashboard;
