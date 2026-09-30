interface StudentInfo {
  name?: string;
  roll_number?: string;
  section_name?: string;
}

interface TopicPerformance {
  topic: string;
  solved: number;
  total: number;
}

interface StudentStats {
  total_time_seconds?: number;
  problems_solved?: number;
  current_streak?: number;
  accuracy?: number;
  total_submissions?: number;
  passed_submissions?: number;
  longest_streak?: number;
}

export interface StudentAnalytics {
  stats?: StudentStats;
  topic_performance?: TopicPerformance[];
  last_active?: string;
}

export interface StudentProfileCardProps {
  studentAnalytics?: StudentAnalytics | null;
  studentInfo?: StudentInfo;
  loading?: boolean;
  onBack?: () => void;
  backLabel?: string;
}

const formatTime = (seconds?: number): string => {
  if (!seconds) return '0m';
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
};

const formatDate = (dateString?: string): string => {
  if (!dateString || dateString === '0001-01-01T00:00:00Z') return 'Never';
  const date = new Date(dateString);
  const now = new Date();
  const diffTime = Math.abs(now.getTime() - date.getTime());
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
  if (diffDays === 0) return 'Today';
  if (diffDays === 1) return 'Yesterday';
  if (diffDays < 7) return `${diffDays} days ago`;
  return date.toLocaleDateString();
};

export default function StudentProfileCard({
  studentAnalytics,
  studentInfo,
  loading = false,
  onBack,
  backLabel = '← Back to Students',
}: StudentProfileCardProps) {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        {onBack && (
          <button
            onClick={onBack}
            className="text-accent-primary hover:text-accent-secondary flex items-center gap-1"
          >
            {backLabel}
          </button>
        )}
        <div className="flex-1">
          <h1 className="text-2xl font-bold text-text-primary">{studentInfo?.name}</h1>
          {(studentInfo?.roll_number || studentInfo?.section_name) && (
            <p className="text-text-secondary text-sm">
              {studentInfo?.roll_number}
              {studentInfo?.section_name && ` • Section ${studentInfo.section_name}`}
            </p>
          )}
        </div>
      </div>

      {loading ? (
        <div className="text-center py-12">
          <div className="inline-block animate-spin h-8 w-8 border-4 border-accent-primary border-t-transparent rounded-full"></div>
          <p className="text-text-muted mt-3">Loading analytics...</p>
        </div>
      ) : studentAnalytics ? (
        <>
          <div className="grid grid-cols-1 lg:grid-cols-4 gap-4">
            <div className="card p-4 text-center">
              <div className="text-3xl font-bold text-accent-primary">
                {formatTime(studentAnalytics.stats?.total_time_seconds)}
              </div>
              <div className="text-text-secondary text-sm mt-1">Total Time</div>
            </div>
            <div className="card p-4 text-center">
              <div className="text-3xl font-bold text-accent-primary">
                {studentAnalytics.stats?.problems_solved || 0}
              </div>
              <div className="text-text-secondary text-sm mt-1">Problems Solved</div>
            </div>
            <div className="card p-4 text-center">
              <div className="text-3xl font-bold text-warning">
                {studentAnalytics.stats?.current_streak || 0}
              </div>
              <div className="text-text-secondary text-sm mt-1">Day Streak</div>
            </div>
            <div className="card p-4 text-center">
              <div className="text-3xl font-bold text-success">
                {studentAnalytics.stats?.accuracy?.toFixed(1) || 0}%
              </div>
              <div className="text-text-secondary text-sm mt-1">Accuracy</div>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {studentAnalytics.topic_performance &&
              studentAnalytics.topic_performance.length > 0 && (
                <div className="card p-4">
                  <h3 className="text-lg font-semibold text-text-primary mb-4">
                    Topic Performance
                  </h3>
                  <div className="space-y-3">
                    {studentAnalytics.topic_performance.map((topic, idx) => (
                      <div key={idx}>
                        <div className="flex justify-between text-sm mb-1">
                          <span className="text-text-primary">{topic.topic}</span>
                          <span className="text-text-secondary">
                            {topic.solved}/{topic.total}
                          </span>
                        </div>
                        <div className="w-full bg-background-secondary rounded-full h-2">
                          <div
                            className="bg-accent-primary h-2 rounded-full"
                            style={{
                              width: `${topic.total > 0 ? (topic.solved / topic.total) * 100 : 0}%`,
                            }}
                          ></div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

            <div className="card p-4">
              <h3 className="text-lg font-semibold text-text-primary mb-2">Activity</h3>
              <p className="text-text-secondary">
                Last active:{' '}
                <span className="text-text-primary font-medium">
                  {formatDate(studentAnalytics.last_active)}
                </span>
              </p>
              <div className="mt-4 text-sm text-text-muted">
                <p>Submissions: {studentAnalytics.stats?.total_submissions || 0}</p>
                <p>Passed: {studentAnalytics.stats?.passed_submissions || 0}</p>
                <p>Longest Streak: {studentAnalytics.stats?.longest_streak || 0} days</p>
              </div>
            </div>
          </div>
        </>
      ) : (
        <div className="card text-center py-12">
          <p className="text-text-muted">Unable to load student analytics</p>
        </div>
      )}
    </div>
  );
}
