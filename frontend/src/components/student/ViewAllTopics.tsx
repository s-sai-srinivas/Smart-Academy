import { useState, useEffect, useRef, type MouseEvent } from 'react';
import { Link } from 'react-router-dom';
import { coursesAPI } from '../../services/api';
import './ViewAllTopics.css';

export interface LabSession {
  id: number;
  session_name: string;
  progress: number;
  solved: number;
  easy_solved: number;
  easy_total: number;
  medium_solved: number;
  medium_total: number;
  hard_solved: number;
  hard_total: number;
}

export interface CourseOverview {
  id: number;
  course_code: string;
  course_name: string;
  course_type: string;
  credits: number;
  lab_sessions?: LabSession[];
}

export interface ViewAllTopicsProps {
  isOpen: boolean;
  onClose: () => void;
}

function ViewAllTopics({ isOpen, onClose }: ViewAllTopicsProps) {
  const [courses, setCourses] = useState<CourseOverview[]>([]);
  const [loading, setLoading] = useState(false);
  const [expandedCourses, setExpandedCourses] = useState<Record<number, boolean>>({});
  const sidebarRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (isOpen) {
      loadTopicsOverview();
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [isOpen]);

  const loadTopicsOverview = async () => {
    setLoading(true);
    try {
      const data = await coursesAPI.getTopicsOverview();
      const list = (data as CourseOverview[]) || [];
      setCourses(list);
      if (list.length > 0) {
        setExpandedCourses({ [list[0].id]: true });
      }
    } catch (err) {
      console.error('Failed to load topics overview:', err);
    } finally {
      setLoading(false);
    }
  };

  const toggleCourse = (courseId: number) => {
    setExpandedCourses((prev) => ({ ...prev, [courseId]: !prev[courseId] }));
  };

  const handleOverlayClick = (e: MouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) onClose();
  };

  const getActionLabel = (session: LabSession): string => {
    if (session.progress === 100) return 'completed';
    if (session.solved > 0) return 'continue';
    return 'start';
  };

  const getActionText = (session: LabSession): string => {
    if (session.progress === 100) return '✓ Done';
    if (session.solved > 0) return 'Continue';
    return 'Start';
  };

  return (
    <>
      <div className={`topics-overlay ${isOpen ? 'open' : ''}`} onClick={handleOverlayClick} />
      <div ref={sidebarRef} className={`topics-sidebar ${isOpen ? 'open' : ''}`}>
        <div className="topics-sidebar-header">
          <h2>All Topics</h2>
          <button className="topics-sidebar-close" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="topics-sidebar-content">
          {loading ? (
            <div className="topics-loading">
              <div className="topics-spinner" />
              <p>Loading your roadmap...</p>
            </div>
          ) : courses.length === 0 ? (
            <div className="topics-empty">
              <div className="topics-empty-icon">📚</div>
              <h3>No Courses Found</h3>
              <p>You're not enrolled in any courses yet.</p>
            </div>
          ) : (
            courses.map((course) => (
              <div key={course.id} className="subject-accordion">
                <button
                  className={`subject-header ${expandedCourses[course.id] ? 'expanded' : ''}`}
                  onClick={() => toggleCourse(course.id)}
                >
                  <div className="subject-code-badge">
                    <span>{course.course_code}</span>
                  </div>
                  <div className="subject-info">
                    <div className="subject-name">{course.course_name}</div>
                    <div className="subject-meta">
                      <span
                        className={`type-badge ${course.course_type === 'lab' ? 'lab' : 'theory'}`}
                      >
                        {course.course_type}
                      </span>
                      <span>{course.credits} Credits</span>
                      <span>·</span>
                      <span>{course.lab_sessions?.length || 0} Topics</span>
                    </div>
                  </div>
                  <span
                    className={`subject-chevron ${expandedCourses[course.id] ? 'rotated' : ''}`}
                  >
                    ▼
                  </span>
                </button>
                <div className={`subject-sessions ${expandedCourses[course.id] ? 'expanded' : ''}`}>
                  <div className="subject-sessions-inner">
                    {!course.lab_sessions || course.lab_sessions.length === 0 ? (
                      <div className="sessions-empty">No lab sessions available yet</div>
                    ) : (
                      course.lab_sessions.map((session) => {
                        const actionLabel = getActionLabel(session);
                        const actionText = getActionText(session);
                        return (
                          <div key={session.id} className="topic-card">
                            <div className="topic-card-body">
                              <div className="topic-card-top">
                                <span className="topic-card-name">{session.session_name}</span>
                                <span
                                  className={`topic-card-progress-pct ${session.progress === 100 ? 'complete' : ''}`}
                                >
                                  {session.progress}%
                                </span>
                              </div>
                              <div className="topic-progress-bar">
                                <div
                                  className={`topic-progress-fill ${session.progress === 100 ? 'complete' : ''}`}
                                  style={{ width: `${session.progress}%` }}
                                />
                              </div>
                              <div className="topic-difficulty-row">
                                <span className="diff-easy">
                                  Easy:{' '}
                                  <span className="diff-solved">
                                    {session.easy_solved}/{session.easy_total}
                                  </span>
                                </span>
                                <span className="diff-medium">
                                  Med:{' '}
                                  <span className="diff-solved">
                                    {session.medium_solved}/{session.medium_total}
                                  </span>
                                </span>
                                <span className="diff-hard">
                                  Hard:{' '}
                                  <span className="diff-solved">
                                    {session.hard_solved}/{session.hard_total}
                                  </span>
                                </span>
                              </div>
                            </div>
                            <Link
                              to={`/courses/${course.id}/lab/${session.id}`}
                              className={`topic-card-action ${actionLabel}`}
                              onClick={onClose}
                            >
                              {actionText}
                            </Link>
                          </div>
                        );
                      })
                    )}
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </>
  );
}

export default ViewAllTopics;
