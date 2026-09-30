import React, { useState, useEffect, useCallback, type ChangeEvent } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { facultyAnalyticsAPI } from '../../services/api';
import { showError } from '../../utils/showAlert';
import {
  BookOpen,
  Users,
  Target,
  AlertTriangle,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Search,
  Filter,
  Activity,
  BarChart3,
  Clock,
  Award,
  Zap,
  Eye,
} from 'lucide-react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from 'recharts';

// Custom scrollbar and animation styles
const customStyles = `
  .custom-scroll::-webkit-scrollbar { width: 6px; height: 6px; }
  .custom-scroll::-webkit-scrollbar-track { background: rgba(255,255,255,0.02); border-radius: 3px; }
  .custom-scroll::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.1); border-radius: 3px; }
  .custom-scroll::-webkit-scrollbar-thumb:hover { background: rgba(255,255,255,0.15); }
  @keyframes fade-in-up {
    from { opacity: 0; transform: translateY(10px); }
    to { opacity: 1; transform: translateY(0); }
  }
  .animate-fade-in { animation: fade-in-up 0.4s ease-out; }
`;

interface CourseInfo {
  course_code?: string;
  CourseCode?: string;
  course_name?: string;
  CourseName?: string;
  is_active?: boolean;
  IsActive?: boolean;
  courseCategory?: string;
}

interface Section {
  section_id?: number;
  SectionID?: number;
  section_name?: string;
  SectionName?: string;
  student_count?: number;
  StudentCount?: number;
}

interface ProblemRef {
  id: number;
  title: string;
  difficulty?: string;
  completion_rate: number;
  avg_attempts: number;
  students_solved?: number;
}

interface Module {
  id: number;
  name: string;
  order: number;
  topic_name?: string;
  problems_count: number;
  completion_rate: number;
  avg_score: number;
  drop_off_rate: number;
  students_solved?: number;
  problems: ProblemRef[];
}

interface Student {
  id: string;
  name: string;
  roll_number: string;
  problems_solved: number;
  total_problems: number;
  time_spent_hours: number;
  current_streak: number;
  engagement_score: number;
  risk_level: 'active' | 'needs_attention' | 'at_risk';
  last_active: string;
}

interface ProgressDistribution {
  range: string;
  count: number;
  color: string;
}

interface CourseData {
  course: CourseInfo;
  sections: Section[];
  modules: Module[];
  healthMetrics: {
    active_rate: number;
    retention_rate: number;
    drop_off_rate: number;
  };
  progressDistribution: ProgressDistribution[];
  students: Student[];
}

const CourseDashboard = () => {
  const { id } = useParams<{ id: string }>();
  const numericId = Number(id);
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState<string>('overview');
  const [loading, setLoading] = useState<boolean>(true);
  const [courseData, setCourseData] = useState<CourseData | null>(null);
  const [selectedSection, setSelectedSection] = useState<number | null>(null);
  const [expandedModules, setExpandedModules] = useState<Record<number, boolean>>({});
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [riskFilter, setRiskFilter] = useState<string>('all');
  const [loadingSection, setLoadingSection] = useState<boolean>(false);

  const loadCourseData = useCallback(async () => {
    try {
      const data = (await facultyAnalyticsAPI.getCourseAnalytics(numericId)) as {
        course: CourseInfo;
        sections?: Section[];
      };

      if (!data || !data.course) {
        throw new Error('Invalid course data received');
      }

      const transformedData: CourseData = {
        course: data.course,
        sections: data.sections || [],
        modules: [],
        healthMetrics: { active_rate: 0, retention_rate: 0, drop_off_rate: 0 },
        progressDistribution: [],
        students: [],
      };

      setCourseData(transformedData);

      if (data.sections && data.sections.length > 0) {
        setSelectedSection(data.sections[0].section_id || data.sections[0].SectionID || null);
      }
    } catch (error) {
      console.error('Failed to load course data:', error);
      showError('Failed to load course data');
    } finally {
      setLoading(false);
    }
  }, [numericId]);

  useEffect(() => {
    loadCourseData();
  }, [loadCourseData]);

  const loadSectionDetails = useCallback(
    async (sectionId: number) => {
      setLoadingSection(true);
      try {
        const sectionData = (await facultyAnalyticsAPI.getSectionAnalytics(
          numericId,
          sectionId
        )) as {
          session_stats?: Array<{
            session_id: number;
            session_name: string;
            session_order: number;
            topic_name: string;
            total_problems: number;
            percentage: number;
            students_solved?: number;
            problems?: Array<{
              id: number;
              title: string;
              difficulty: string;
              completion_rate: number;
              avg_attempts: number;
              students_solved?: number;
            }>;
          }>;
          topic_stats?: Array<{
            topic_name: string;
            total_problems: number;
            percentage: number;
            students_solved?: number;
            problems?: Array<{
              id: number;
              title: string;
              difficulty: string;
              completion_rate: number;
              avg_attempts: number;
              students_solved?: number;
            }>;
          }>;
          section?: { total_students?: number };
          students?: Array<{
            regdno?: string;
            name: string;
            roll_number?: string;
            problems_solved?: number;
            time_spent_seconds?: number;
            current_streak?: number;
            last_active?: string;
          }>;
        };

        const modules = (sectionData.session_stats || []).map((session) => ({
          id: session.session_id,
          name: session.session_name,
          order: session.session_order,
          topic_name: session.topic_name,
          problems_count: session.total_problems,
          completion_rate: session.percentage,
          avg_score: session.percentage,
          drop_off_rate: Math.max(0, 100 - session.percentage),
          students_solved: session.students_solved,
          problems: (session.problems || []).map((p) => ({
            id: p.id,
            title: p.title,
            difficulty: p.difficulty,
            completion_rate: Math.round(p.completion_rate * 10) / 10,
            avg_attempts: Math.round(p.avg_attempts * 10) / 10,
            students_solved: p.students_solved,
          })),
        }));

        // If no session stats (e.g., theory course), try topic_stats for backward compatibility
        const fallbackModules =
          !sectionData.session_stats?.length && sectionData.topic_stats
            ? (sectionData.topic_stats || []).map((topic, index) => ({
                id: index + 1,
                name: topic.topic_name,
                order: index + 1,
                topic_name: topic.topic_name,
                problems_count: topic.total_problems,
                completion_rate: topic.percentage,
                avg_score: topic.percentage,
                drop_off_rate: Math.max(0, 100 - topic.percentage),
                students_solved: topic.students_solved,
                problems: (topic.problems || []).map((p) => ({
                  id: p.id,
                  title: p.title,
                  difficulty: p.difficulty,
                  completion_rate: Math.round(p.completion_rate * 10) / 10,
                  avg_attempts: Math.round(p.avg_attempts * 10) / 10,
                  students_solved: p.students_solved,
                })),
              }))
            : [];

        const finalModules = modules.length > 0 ? modules : fallbackModules;

        const totalStudents = sectionData.section?.total_students || 0;
        const activeStudents =
          sectionData.students?.filter((s) => (s.current_streak || 0) > 0).length || 0;
        const activeRate = totalStudents > 0 ? (activeStudents / totalStudents) * 100 : 0;

        // Calculate total problems across all sessions for progress baseline
        const totalCourseProblems = finalModules.reduce((sum, m) => sum + m.problems_count, 0) || 1;

        const progressBuckets: Record<string, number> = {
          '0-25%': 0,
          '26-50%': 0,
          '51-75%': 0,
          '76-100%': 0,
        };
        sectionData.students?.forEach((student) => {
          const solved = student.problems_solved || 0;
          const progress = Math.min((solved / totalCourseProblems) * 100, 100);
          if (progress <= 25) progressBuckets['0-25%']++;
          else if (progress <= 50) progressBuckets['26-50%']++;
          else if (progress <= 75) progressBuckets['51-75%']++;
          else progressBuckets['76-100%']++;
        });

        const progressDistribution: ProgressDistribution[] = [
          { range: '0-25%', count: progressBuckets['0-25%'], color: 'var(--red-500)' },
          { range: '26-50%', count: progressBuckets['26-50%'], color: 'var(--amber-500)' },
          { range: '51-75%', count: progressBuckets['51-75%'], color: 'var(--text-muted)' },
          { range: '76-100%', count: progressBuckets['76-100%'], color: 'var(--emerald-500)' },
        ];

        const students: Student[] = (sectionData.students || []).map((student) => {
          const problemsSolved = student.problems_solved || 0;
          const timeSpentHours = (student.time_spent_seconds || 0) / 3600;
          const streak = student.current_streak || 0;

          let engagementScore = 0;
          engagementScore += Math.min(problemsSolved * 2, 50);
          engagementScore += Math.min(timeSpentHours * 2, 30);
          engagementScore += Math.min(streak * 5, 20);

          let riskLevel: Student['risk_level'] = 'active';
          const lastActiveDays = student.last_active
            ? Math.floor(
                (Date.now() - new Date(student.last_active).getTime()) / (1000 * 60 * 60 * 24)
              )
            : 999;

          if (engagementScore < 30) {
            riskLevel = 'at_risk';
          } else if (engagementScore < 50) {
            riskLevel = 'needs_attention';
          }

          return {
            id: student.regdno || student.name,
            name: student.name,
            roll_number: student.roll_number || student.regdno || '',
            problems_solved: problemsSolved,
            total_problems: totalCourseProblems,
            time_spent_hours: timeSpentHours,
            current_streak: streak,
            engagement_score: Math.min(100, Math.round(engagementScore)),
            risk_level: riskLevel,
            last_active:
              lastActiveDays === 0
                ? 'Today'
                : lastActiveDays === 1
                  ? 'Yesterday'
                  : lastActiveDays < 7
                    ? `${lastActiveDays} days ago`
                    : lastActiveDays >= 999
                      ? 'Never'
                      : `${Math.floor(lastActiveDays / 7)} week${lastActiveDays > 14 ? 's' : ''} ago`,
          };
        });

        setCourseData((prev) => ({
          ...(prev as CourseData),
          modules: finalModules,
          healthMetrics: {
            active_rate: Math.round(activeRate),
            retention_rate: Math.round(activeRate * 0.95),
            drop_off_rate: Math.round(100 - activeRate),
          },
          progressDistribution,
          students,
        }));
      } catch (error) {
        console.error('Failed to load section details:', error);
      } finally {
        setLoadingSection(false);
      }
    },
    [numericId]
  );

  useEffect(() => {
    if (selectedSection) {
      loadSectionDetails(selectedSection);
    }
  }, [selectedSection, loadSectionDetails]);

  const toggleModule = (moduleId: number) => {
    setExpandedModules((prev) => ({
      ...prev,
      [moduleId]: !prev[moduleId],
    }));
  };

  const getEngagementColor = (score: number) => {
    if (score >= 80) return 'from-emerald-500/20 to-emerald-500/5 border-emerald-500/30';
    if (score >= 60)
      return 'from-background-tertiary to-background-secondary border-background-border';
    if (score >= 40) return 'from-amber-500/20 to-amber-500/5 border-amber-500/30';
    return 'from-red-500/20 to-red-500/5 border-red-500/30';
  };

  const getRiskBadge = (level: string) => {
    const config: Record<string, { label: string; icon: React.ElementType; className: string }> = {
      active: {
        label: 'Active',
        icon: CheckCircle2,
        className: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20',
      },
      needs_attention: {
        label: 'Needs Attention',
        icon: AlertTriangle,
        className: 'bg-accent-secondary/10 text-amber-400 border-amber-500/20',
      },
      at_risk: {
        label: 'At Risk',
        icon: AlertTriangle,
        className: 'bg-accent-danger/10 text-accent-danger border-red-500/20',
      },
    };
    return config[level] || config.active;
  };

  const filteredStudents =
    courseData?.students?.filter((student) => {
      const matchesSearch =
        student.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        student.roll_number.toLowerCase().includes(searchQuery.toLowerCase());
      const matchesRisk = riskFilter === 'all' || student.risk_level === riskFilter;
      return matchesSearch && matchesRisk;
    }) || [];

  const getDifficultyColor = (difficulty?: string) => {
    switch (difficulty?.toLowerCase()) {
      case 'easy':
        return 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20';
      case 'medium':
        return 'text-amber-400 bg-accent-secondary/10 border-amber-500/20';
      case 'hard':
        return 'text-accent-danger bg-accent-danger/10 border-red-500/20';
      default:
        return 'text-text-secondary bg-background-tertiary border-background-border';
    }
  };

  const tabs = [
    { id: 'overview', label: 'Course Overview', icon: BookOpen },
    { id: 'analytics', label: 'Course Analytics', icon: BarChart3 },
    { id: 'students', label: 'Student Analytics', icon: Users },
  ];

  if (loading) {
    return (
      <div className="min-h-screen bg-background-primary flex items-center justify-center">
        <div className="text-center">
          <div className="w-12 h-12 border-2 border-background-border border-t-text-secondary rounded-full animate-spin mx-auto mb-4" />
          <p className="text-text-secondary">Loading course dashboard...</p>
        </div>
      </div>
    );
  }

  if (!courseData) {
    return (
      <div className="min-h-screen bg-background-primary flex items-center justify-center">
        <div className="text-center max-w-md mx-auto p-8 bg-background-secondary rounded-xl border-2 border-background-border">
          <div className="w-16 h-16 rounded-full bg-accent-danger/10 border border-red-500/20 flex items-center justify-center mx-auto mb-4">
            <AlertTriangle className="w-8 h-8 text-accent-danger" />
          </div>
          <h2 className="text-2xl font-bold text-text-primary mb-2">Error Loading Course</h2>
          <p className="text-text-secondary mb-6">
            Unable to load course data. You may not have access to this course.
          </p>
          <button
            onClick={() => navigate('/faculty')}
            className="px-6 py-3 bg-background-tertiary hover:bg-background-elevated text-text-primary rounded-lg font-medium transition-colors border-2 border-background-border"
          >
            Back to Dashboard
          </button>
        </div>
      </div>
    );
  }

  const totalStudents =
    courseData.sections?.reduce((sum, s) => sum + (s.student_count || 0), 0) || 0;
  const totalModules = courseData.modules?.length || 0;
  const totalProblems = courseData.modules?.reduce((sum, m) => sum + m.problems_count, 0) || 0;
  return (
    <div className="min-h-screen bg-background-primary">
      <style>{customStyles}</style>

      {/* Header */}
      <header className="border-b border-background-border bg-background-secondary/80 backdrop-blur-xl sticky top-0 z-40">
        <div className="max-w-[1800px] mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <button
                onClick={() => navigate('/faculty')}
                className="p-2 -ml-2 rounded-lg hover:bg-background-tertiary transition-colors text-text-secondary hover:text-text-primary"
              >
                <ChevronRight className="w-5 h-5 rotate-180" />
              </button>
              <div>
                <div className="flex items-center gap-3">
                  <span className="px-2.5 py-1 text-xs font-semibold tracking-wide rounded-md bg-accent-primary/10 text-accent-primary border border-accent-primary/20">
                    {courseData.course.course_code || courseData.course.CourseCode}
                  </span>
                  <h1 className="text-2xl font-bold text-text-primary tracking-tight">
                    {courseData.course.course_name || courseData.course.CourseName}
                  </h1>
                </div>
                <p className="text-sm text-text-muted mt-1">
                  {courseData.sections?.length} section
                  {courseData.sections?.length !== 1 ? 's' : ''} · {totalStudents} student
                  {totalStudents !== 1 ? 's' : ''} enrolled
                </p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              {loadingSection && (
                <div className="w-5 h-5 border-2 border-background-border border-t-accent-primary rounded-full animate-spin" />
              )}
              {courseData.sections && courseData.sections.length > 1 && (
                <select
                  value={selectedSection || ''}
                  onChange={(e: ChangeEvent<HTMLSelectElement>) =>
                    setSelectedSection(parseInt(e.target.value))
                  }
                  className="px-4 py-2 bg-background-tertiary border-2 border-background-border rounded-lg text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-accent-primary/50"
                >
                  {courseData.sections.map((section) => (
                    <option
                      key={section.section_id || section.SectionID}
                      value={section.section_id || section.SectionID}
                    >
                      {section.section_name || section.SectionName} (
                      {section.student_count || section.StudentCount} students)
                    </option>
                  ))}
                </select>
              )}
            </div>
          </div>

          {/* Tabs */}
          <div className="flex gap-1 mt-4 -mx-2 px-2">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-2 px-4 py-2.5 rounded-lg text-sm font-medium transition-all relative ${
                  activeTab === tab.id
                    ? 'text-accent-primary bg-accent-primary/10'
                    : 'text-text-muted hover:text-text-secondary hover:bg-background-tertiary'
                }`}
              >
                <tab.icon className="w-4 h-4" />
                {tab.label}
                {activeTab === tab.id && (
                  <div className="absolute bottom-0 left-4 right-4 h-0.5 bg-gradient-to-r from-blue-500 to-blue-600 rounded-full" />
                )}
              </button>
            ))}
          </div>
        </div>
      </header>

      {/* Content */}
      <main className="max-w-[1800px] mx-auto px-6 py-8 animate-fade-in">
        {activeTab === 'overview' && (
          <div className="space-y-6">
            {/* Summary Metrics */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
              {[
                { label: 'Total Students', value: totalStudents, icon: Users, color: 'blue' },
                { label: 'Topics', value: totalModules, icon: BookOpen, color: 'purple' },
                { label: 'Problems', value: totalProblems, icon: Target, color: 'emerald' },
                {
                  label: 'Status',
                  value:
                    courseData.course.is_active !== undefined
                      ? courseData.course.is_active
                        ? 'Active'
                        : 'Inactive'
                      : courseData.course.IsActive !== undefined
                        ? courseData.course.IsActive
                          ? 'Active'
                          : 'Inactive'
                        : 'Active',
                  icon: Activity,
                  color: 'green',
                },
              ].map((metric, i) => (
                <div
                  key={i}
                  className="group relative p-5 bg-background-secondary rounded-xl border-2 border-background-border hover:border-background-elevated transition-all"
                >
                  <div className="relative">
                    <div className="flex items-center justify-between mb-3">
                      <span className="text-xs font-medium text-text-muted uppercase tracking-wider">
                        {metric.label}
                      </span>
                      <metric.icon className={`w-4 h-4 text-${metric.color}-400`} />
                    </div>
                    <p className="text-3xl font-bold text-text-primary tracking-tight">
                      {metric.value}
                    </p>
                  </div>
                </div>
              ))}
            </div>

            {/* Course Structure */}
            <div className="bg-background-secondary rounded-xl border-2 border-background-border overflow-hidden">
              <div className="px-6 py-4 border-b border-background-border">
                <h2 className="text-lg font-semibold text-text-primary flex items-center gap-2">
                  <BookOpen className="w-5 h-5 text-accent-primary" />
                  {courseData.course.courseCategory === 'lab' ? 'Lab Sessions' : 'Course Modules'}
                </h2>
              </div>
              <div className="divide-y divide-background-border custom-scroll max-h-[600px] overflow-y-auto">
                {!courseData.modules || courseData.modules.length === 0 ? (
                  <div className="px-6 py-12 text-center">
                    <BookOpen className="w-12 h-12 mx-auto mb-3 text-slate-600" />
                    <p className="text-text-secondary">
                      {courseData.course.courseCategory === 'lab'
                        ? 'No lab sessions available for this course yet.'
                        : 'No modules available for this course yet.'}
                    </p>
                    <p className="text-sm text-slate-600 mt-1">
                      {courseData.course.courseCategory === 'lab'
                        ? 'Lab sessions will appear once they are created and students start solving problems.'
                        : 'Modules will appear once students start solving problems.'}
                    </p>
                  </div>
                ) : (
                  courseData.modules.map((module, idx) => (
                    <div
                      key={module.id}
                      className="hover:bg-background-elevated/30 transition-colors"
                    >
                      <button
                        onClick={() => toggleModule(module.id)}
                        className="w-full px-6 py-4 flex items-center justify-between text-left hover:bg-background-elevated/30 transition-colors"
                      >
                        <div className="flex items-center gap-4 flex-1">
                          <span className="flex items-center justify-center w-8 h-8 rounded-lg bg-background-tertiary text-sm font-semibold text-text-secondary">
                            {idx + 1}
                          </span>
                          <div className="flex-1 min-w-0">
                            <h3 className="text-text-primary font-medium truncate">
                              {module.name}
                            </h3>
                            <p className="text-sm text-text-muted">
                              {module.problems_count} problems
                              {module.topic_name && module.topic_name !== module.name && (
                                <span className="text-text-secondary"> · {module.topic_name}</span>
                              )}
                            </p>
                          </div>
                        </div>
                        <div className="flex items-center gap-6">
                          {expandedModules[module.id] ? (
                            <ChevronDown className="w-5 h-5 text-text-muted" />
                          ) : (
                            <ChevronRight className="w-5 h-5 text-text-muted" />
                          )}
                        </div>
                      </button>

                      {/* Expanded Problems */}
                      {expandedModules[module.id] &&
                        module.problems &&
                        module.problems.length > 0 && (
                          <div className="border-t border-background-border bg-background-primary">
                            <div className="px-6 pb-4 pt-4 space-y-2">
                              {module.problems.map((problem) => (
                                <div
                                  key={problem.id}
                                  className="flex items-center justify-between p-3 bg-background-tertiary rounded-lg border-2 border-background-border"
                                >
                                  <div className="flex items-center gap-3">
                                    <Target className="w-4 h-4 text-text-muted" />
                                    <span className="text-text-primary font-medium">
                                      {problem.title}
                                    </span>
                                    <span
                                      className={`px-2 py-0.5 text-xs font-medium rounded border ${getDifficultyColor(problem.difficulty)}`}
                                    >
                                      {problem.difficulty}
                                    </span>
                                  </div>
                                  <div className="flex items-center gap-6 text-sm">
                                    <div className="text-center">
                                      <span className="text-text-secondary">
                                        {problem.completion_rate}% complete
                                      </span>
                                    </div>
                                    <div className="text-text-muted">
                                      {problem.avg_attempts} avg attempts
                                    </div>
                                  </div>
                                </div>
                              ))}
                            </div>
                          </div>
                        )}
                      {expandedModules[module.id] &&
                        (!module.problems || module.problems.length === 0) && (
                          <div className="border-t border-background-border bg-background-primary px-6 py-6 text-center">
                            <Target className="w-8 h-8 mx-auto mb-2 text-slate-600" />
                            <p className="text-text-muted text-sm">
                              No problems have been solved in this topic yet.
                            </p>
                          </div>
                        )}
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        )}

        {activeTab === 'analytics' && (
          <div className="space-y-6">
            {/* Charts Row */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* Progress Distribution */}
              <div className="bg-background-secondary rounded-xl border-2 border-background-border p-6">
                <h3 className="text-lg font-semibold text-text-primary mb-6 flex items-center gap-2">
                  <BarChart3 className="w-5 h-5 text-accent-primary" />
                  Progress Distribution
                </h3>
                {courseData.progressDistribution.every((d) => d.count === 0) ? (
                  <div className="flex items-center justify-center h-[250px]">
                    <p className="text-text-muted">
                      No progress data yet. Waiting for student submissions.
                    </p>
                  </div>
                ) : (
                  <ResponsiveContainer width="100%" height={250}>
                    <BarChart data={courseData.progressDistribution} layout="vertical">
                      <CartesianGrid
                        strokeDasharray="3 3"
                        stroke="rgba(255,255,255,0.05)"
                        horizontal={false}
                      />
                      <XAxis
                        type="number"
                        stroke="rgba(255,255,255,0.1)"
                        tick={{ fill: 'rgba(255,255,255,0.5)', fontSize: 12 }}
                      />
                      <YAxis
                        type="category"
                        dataKey="range"
                        stroke="rgba(255,255,255,0.1)"
                        tick={{ fill: 'rgba(255,255,255,0.5)', fontSize: 12 }}
                        width={60}
                      />
                      <Tooltip
                        contentStyle={{
                          backgroundColor: 'var(--bg-secondary)',
                          border: '1px solid var(--bg-border)',
                          borderRadius: '8px',
                        }}
                        itemStyle={{ color: 'var(--text-primary)' }}
                        labelStyle={{ color: 'var(--text-secondary)' }}
                      />
                      <Bar dataKey="count" radius={[0, 6, 6, 0]}>
                        {courseData.progressDistribution.map((entry, index) => (
                          <Cell key={`cell-${index}`} fill={entry.color} />
                        ))}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                )}
              </div>

              {/* Module Performance */}
              <div className="bg-background-secondary rounded-xl border-2 border-background-border p-6">
                <h3 className="text-lg font-semibold text-text-primary mb-6 flex items-center gap-2">
                  <Award className="w-5 h-5 text-accent-secondary" />
                  Topic Performance Comparison
                </h3>
                {!courseData.modules || courseData.modules.length === 0 ? (
                  <div className="flex items-center justify-center h-[200px]">
                    <p className="text-text-muted">No topic data available yet.</p>
                  </div>
                ) : (
                  <div className="space-y-4">
                    {courseData.modules.map((module) => (
                      <div key={module.id}>
                        <div className="flex items-center justify-between mb-2">
                          <span className="text-sm text-text-secondary truncate max-w-[200px]">
                            {module.name}
                          </span>
                          <span className="text-sm font-semibold text-text-primary">
                            {module.avg_score.toFixed(1)}%
                          </span>
                        </div>
                        <div className="h-2 bg-background-elevated rounded-full overflow-hidden">
                          <div
                            className={`h-full rounded-full transition-all ${
                              module.avg_score >= 75
                                ? 'bg-emerald-500'
                                : module.avg_score >= 60
                                  ? 'bg-accent-primary'
                                  : module.avg_score >= 45
                                    ? 'bg-accent-secondary'
                                    : 'bg-accent-danger'
                            }`}
                            style={{ width: `${module.avg_score}%` }}
                          />
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>

            {/* Problem Performance Insights */}
            <div className="bg-background-secondary rounded-xl border-2 border-background-border overflow-hidden">
              <div className="px-6 py-4 border-b border-background-border">
                <h3 className="text-lg font-semibold text-text-primary flex items-center gap-2">
                  <Eye className="w-5 h-5 text-cyan-400" />
                  Topic-Level Insights
                </h3>
              </div>
              {!courseData.modules || courseData.modules.length === 0 ? (
                <div className="px-6 py-12 text-center">
                  <Eye className="w-12 h-12 mx-auto mb-3 text-slate-600" />
                  <p className="text-text-muted">No topic insights available yet.</p>
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-background-border">
                        <th className="px-6 py-4 text-left text-xs font-semibold text-text-muted uppercase tracking-wider">
                          Topic
                        </th>
                        <th className="px-6 py-4 text-left text-xs font-semibold text-text-muted uppercase tracking-wider">
                          Problems
                        </th>
                        <th className="px-6 py-4 text-left text-xs font-semibold text-text-muted uppercase tracking-wider">
                          Students Solved
                        </th>
                        <th className="px-6 py-4 text-left text-xs font-semibold text-text-muted uppercase tracking-wider">
                          Completion
                        </th>
                        <th className="px-6 py-4 text-left text-xs font-semibold text-text-muted uppercase tracking-wider">
                          Insight
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-background-border">
                      {courseData.modules.map((module) => {
                        let insight = 'Normal';
                        let insightColor = 'text-text-secondary';
                        if (module.completion_rate < 30) {
                          insight = 'Low engagement — needs review';
                          insightColor = 'text-accent-danger';
                        } else if (module.completion_rate < 50) {
                          insight = 'Below average — monitor closely';
                          insightColor = 'text-amber-400';
                        } else if (module.completion_rate > 80) {
                          insight = 'Well understood';
                          insightColor = 'text-emerald-400';
                        }
                        return (
                          <tr
                            key={module.id}
                            className="hover:bg-background-elevated/30 transition-colors"
                          >
                            <td className="px-6 py-4 font-medium text-text-primary">
                              {module.name}
                            </td>
                            <td className="px-6 py-4 text-text-secondary">
                              {module.problems_count}
                            </td>
                            <td className="px-6 py-4 text-text-secondary">
                              {module.students_solved || 0}
                            </td>
                            <td className="px-6 py-4">
                              <div className="flex items-center gap-2">
                                <div className="w-20 h-1.5 bg-background-elevated rounded-full overflow-hidden">
                                  <div
                                    className={`h-full rounded-full ${
                                      module.completion_rate >= 75
                                        ? 'bg-emerald-500'
                                        : module.completion_rate >= 50
                                          ? 'bg-accent-primary'
                                          : 'bg-accent-danger'
                                    }`}
                                    style={{ width: `${module.completion_rate}%` }}
                                  />
                                </div>
                                <span className="text-text-primary font-medium">
                                  {module.completion_rate.toFixed(1)}%
                                </span>
                              </div>
                            </td>
                            <td className={`px-6 py-4 text-xs font-medium ${insightColor}`}>
                              {insight}
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        )}

        {activeTab === 'students' && (
          <div className="space-y-6">
            {/* Filters */}
            <div className="flex flex-col sm:flex-row gap-4">
              <div className="relative flex-1 max-w-md">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" />
                <input
                  type="text"
                  placeholder="Search students..."
                  value={searchQuery}
                  onChange={(e: ChangeEvent<HTMLInputElement>) => setSearchQuery(e.target.value)}
                  className="w-full pl-10 pr-4 py-2.5 bg-background-secondary border-2 border-background-border rounded-lg text-sm text-text-primary placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-accent-primary/50"
                />
              </div>
              <div className="flex items-center gap-2">
                <Filter className="w-4 h-4 text-text-muted" />
                <select
                  value={riskFilter}
                  onChange={(e: ChangeEvent<HTMLSelectElement>) => setRiskFilter(e.target.value)}
                  className="px-4 py-2.5 bg-background-secondary border-2 border-background-border rounded-lg text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-accent-primary/50"
                >
                  <option value="all">All Students</option>
                  <option value="active">Active</option>
                  <option value="needs_attention">Needs Attention</option>
                  <option value="at_risk">At Risk</option>
                </select>
              </div>
            </div>

            {/* Stats Summary */}
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
              {[
                { label: 'Total', value: courseData.students?.length || 0, color: 'slate' },
                {
                  label: 'Active',
                  value: courseData.students?.filter((s) => s.risk_level === 'active').length || 0,
                  color: 'emerald',
                },
                {
                  label: 'Needs Attention',
                  value:
                    courseData.students?.filter((s) => s.risk_level === 'needs_attention').length ||
                    0,
                  color: 'amber',
                },
                {
                  label: 'At Risk',
                  value: courseData.students?.filter((s) => s.risk_level === 'at_risk').length || 0,
                  color: 'red',
                },
              ].map((stat, i) => (
                <div
                  key={i}
                  className="p-4 bg-background-secondary rounded-xl border-2 border-background-border"
                >
                  <p className="text-xs text-text-muted uppercase tracking-wider mb-1">
                    {stat.label}
                  </p>
                  <p className={`text-2xl font-bold text-${stat.color}-400`}>{stat.value}</p>
                </div>
              ))}
            </div>

            {/* Student List */}
            <div className="bg-background-secondary rounded-xl border-2 border-background-border overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-background-border">
                      <th className="px-6 py-4 text-left text-xs font-semibold text-text-muted uppercase tracking-wider">
                        Student
                      </th>
                      <th className="px-6 py-4 text-left text-xs font-semibold text-text-muted uppercase tracking-wider">
                        Engagement
                      </th>
                      <th className="px-6 py-4 text-left text-xs font-semibold text-text-muted uppercase tracking-wider">
                        Problems
                      </th>
                      <th className="px-6 py-4 text-left text-xs font-semibold text-text-muted uppercase tracking-wider">
                        Streak
                      </th>
                      <th className="px-6 py-4 text-left text-xs font-semibold text-text-muted uppercase tracking-wider">
                        Status
                      </th>
                      <th className="px-6 py-4 text-left text-xs font-semibold text-text-muted uppercase tracking-wider">
                        Last Active
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-background-border">
                    {filteredStudents.map((student) => {
                      const riskBadge = getRiskBadge(student.risk_level);
                      const progressPct =
                        student.total_problems > 0
                          ? Math.min(100, (student.problems_solved / student.total_problems) * 100)
                          : 0;
                      return (
                        <tr
                          key={student.id}
                          className="hover:bg-background-elevated/30 transition-colors cursor-pointer group"
                          onClick={() =>
                            navigate(`/faculty/student/${student.roll_number}/analytics`)
                          }
                        >
                          <td className="px-6 py-4">
                            <div>
                              <p className="font-medium text-text-primary group-hover:text-accent-primary transition-colors">
                                {student.name}
                              </p>
                              <p className="text-sm text-text-muted">{student.roll_number}</p>
                            </div>
                          </td>
                          <td className="px-6 py-4">
                            <div
                              className={`relative px-3 py-2 rounded-lg border bg-gradient-to-br ${getEngagementColor(student.engagement_score)}`}
                            >
                              <div className="flex items-center gap-2">
                                <span className="text-lg font-bold text-text-primary">
                                  {student.engagement_score}
                                </span>
                                <span className="text-xs text-text-secondary">/ 100</span>
                              </div>
                            </div>
                          </td>
                          <td className="px-6 py-4">
                            <div className="flex items-center gap-3">
                              <div className="flex items-center gap-2">
                                <Target className="w-4 h-4 text-slate-600" />
                                <span className="text-text-primary font-medium">
                                  {student.problems_solved}
                                </span>
                                <span className="text-slate-600">/ {student.total_problems}</span>
                              </div>
                              <div className="w-16 h-1.5 bg-background-elevated rounded-full overflow-hidden">
                                <div
                                  className={`h-full rounded-full ${
                                    progressPct >= 75
                                      ? 'bg-emerald-500'
                                      : progressPct >= 50
                                        ? 'bg-accent-primary'
                                        : progressPct >= 25
                                          ? 'bg-accent-secondary'
                                          : 'bg-accent-danger'
                                  }`}
                                  style={{ width: `${progressPct}%` }}
                                />
                              </div>
                            </div>
                          </td>
                          <td className="px-6 py-4">
                            <div className="flex items-center gap-1.5">
                              <Zap
                                className={`w-3.5 h-3.5 ${student.current_streak > 0 ? 'text-amber-400' : 'text-slate-600'}`}
                              />
                              <span
                                className={`font-medium ${student.current_streak > 0 ? 'text-amber-400' : 'text-slate-600'}`}
                              >
                                {student.current_streak} day
                                {student.current_streak !== 1 ? 's' : ''}
                              </span>
                            </div>
                          </td>
                          <td className="px-6 py-4">
                            <span
                              className={`inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-md border ${riskBadge.className}`}
                            >
                              <riskBadge.icon className="w-3.5 h-3.5" />
                              {riskBadge.label}
                            </span>
                          </td>
                          <td className="px-6 py-4 text-sm text-text-secondary">
                            <div className="flex items-center gap-1.5">
                              <Clock className="w-3.5 h-3.5" />
                              {student.last_active}
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
              {filteredStudents.length === 0 && (
                <div className="px-6 py-12 text-center text-text-muted">
                  <Users className="w-12 h-12 mx-auto mb-3 opacity-50" />
                  <p>
                    {!courseData.students || courseData.students.length === 0
                      ? 'No students enrolled in this section yet.'
                      : 'No students found matching your filters.'}
                  </p>
                </div>
              )}
            </div>
          </div>
        )}
      </main>
    </div>
  );
};

export default CourseDashboard;
