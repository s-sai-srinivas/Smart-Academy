import { useState, useRef, useEffect, useCallback } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import { useContestMode } from '../../context/ContestModeContext';
import { useTheme } from '../../context/ThemeContext';
import { secureTokenStorage } from '../../services/secureStorage';

export const STREAK_UPDATED_EVENT = 'streakUpdated';

function Navbar() {
  const { user, logout, isAdmin, isSuperAdmin, isHOD, isFaculty, isPrincipal, isStudent } =
    useAuth();
  const { isInContestMode, activeContestId, contestTitle } = useContestMode();
  const { theme, toggleTheme } = useTheme();
  const navigate = useNavigate();
  const location = useLocation();
  const [showProfileMenu, setShowProfileMenu] = useState(false);
  const profileMenuRef = useRef<HTMLDivElement>(null);
  const [streak, setStreak] = useState(0);
  const isStudentUser = user?.role === 'student';

  const fetchStreak = useCallback(async (): Promise<number> => {
    try {
      const token = secureTokenStorage.getToken();
      const authValue = token === 'httpOnly' ? 'httpOnly' : `Bearer ${token}`;
      const response = await fetch('/api/me', { headers: { Authorization: authValue } });
      if (response.ok) {
        const data = (await response.json()) as { streak?: number };
        return data.streak || 0;
      }
    } catch (err) {
      console.warn('Could not fetch streak:', err);
    }
    return 0;
  }, []);

  useEffect(() => {
    if (user && isStudentUser) {
      fetchStreak().then(setStreak);
    }
  }, [user, isStudentUser, fetchStreak]);

  useEffect(() => {
    const handleStreakUpdate = () => {
      if (user && isStudentUser) fetchStreak().then(setStreak);
    };
    window.addEventListener(STREAK_UPDATED_EVENT, handleStreakUpdate);
    return () => window.removeEventListener(STREAK_UPDATED_EVENT, handleStreakUpdate);
  }, [user, isStudentUser, fetchStreak]);

  useEffect(() => {
    const handleClickOutside = (e: globalThis.MouseEvent) => {
      if (profileMenuRef.current && !profileMenuRef.current.contains(e.target as Node)) {
        setShowProfileMenu(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  if (isInContestMode && activeContestId) {
    const contestIdStr = String(activeContestId);
    const isOnLeaderboard = location.pathname === `/contests/${contestIdStr}/leaderboard`;
    const isOnProblems = location.pathname === `/contests/${contestIdStr}/problems`;

    return (
      <nav className="sticky top-0 z-50 bg-background-secondary border-b border-background-border contest-mode-navbar">
        <div className="flex items-center justify-between px-6 py-3">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2 text-accent-warning">
              <svg
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
              >
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
                <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
              </svg>
              <span className="text-sm font-semibold">Contest Mode Active</span>
            </div>
            <span className="text-text-primary font-medium">{contestTitle || 'Contest'}</span>
          </div>
          <div className="flex items-center gap-2">
            {!isOnProblems && (
              <Link
                to={`/contests/${activeContestId}/problems`}
                className="px-4 py-2 text-sm font-medium text-text-secondary hover:text-text-primary hover:bg-background-tertiary rounded-lg transition-colors"
              >
                ← Back to Problems
              </Link>
            )}
            {!isOnLeaderboard && (
              <Link
                to={`/contests/${activeContestId}/leaderboard`}
                className="px-4 py-2 text-sm font-medium text-text-secondary hover:text-text-primary hover:bg-background-tertiary rounded-lg transition-colors flex items-center gap-2"
              >
                <svg
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                >
                  <rect x="4" y="14" width="4" height="8" rx="1"></rect>
                  <rect x="10" y="6" width="4" height="16" rx="1"></rect>
                  <rect x="16" y="10" width="4" height="12" rx="1"></rect>
                </svg>
                Leaderboard
              </Link>
            )}
          </div>
          <div className="text-text-muted text-sm">
            <span className="text-accent-danger font-medium">
              ⚠️ Navigation to other pages is blocked
            </span>
          </div>
        </div>
      </nav>
    );
  }

  const handleLogout = () => {
    setShowProfileMenu(false);
    logout();
    navigate('/');
  };

  const isActive = (path: string): boolean => location.pathname === path;

  return (
    <>
      <nav
        className="sticky top-0 z-50 bg-background-secondary border-b border-background-border"
        role="navigation"
        aria-label="Main navigation"
      >
        <div className="flex items-center justify-between px-6 py-3">
          <div className="flex items-center gap-6">
            <Link
              to={
                isSuperAdmin()
                  ? '/super-admin'
                  : isAdmin()
                    ? '/admin'
                    : isHOD()
                      ? '/hod'
                      : isPrincipal()
                        ? '/principal'
                        : isFaculty()
                          ? '/faculty'
                          : '/'
              }
              className="flex items-center gap-2.5 text-accent-secondary hover:text-accent-secondary/80 transition-colors"
              aria-label="Home"
            >
              <svg
                className="w-6 h-6"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
              >
                <path d="M16 18l6-6-6-6M8 6l-6 6 6 6" />
              </svg>
              <span className="text-lg font-semibold text-text-primary">Smart Academy</span>
            </Link>
            {/* ... navigation links kept identical ... */}
            {isSuperAdmin() ? (
              <nav aria-label="Super Admin navigation">
                <div className="flex items-center gap-1">
                  <Link
                    to="/super-admin"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/super-admin') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                  >
                    Colleges
                  </Link>
                  <Link
                    to="/super-admin/problems"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${location.pathname.startsWith('/super-admin/problems') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                  >
                    Problems
                  </Link>
                  <Link
                    to="/super-admin/topics"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${location.pathname.startsWith('/super-admin/topics') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                  >
                    Topics
                  </Link>
                </div>
              </nav>
            ) : isAdmin() ? (
              <nav aria-label="Admin navigation">
                <div className="flex items-center gap-1">
                  <Link
                    to="/admin"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/admin') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={isActive('/admin') ? 'page' : undefined}
                  >
                    Home
                  </Link>
                  <Link
                    to="/admin/courses"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/admin/courses') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={isActive('/admin/courses') ? 'page' : undefined}
                  >
                    Courses
                  </Link>
                  <Link
                    to="/admin/labs"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/admin/labs') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={isActive('/admin/labs') ? 'page' : undefined}
                  >
                    Labs
                  </Link>
                  <Link
                    to="/admin/theory"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/admin/theory') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={isActive('/admin/theory') ? 'page' : undefined}
                  >
                    Theory
                  </Link>
                  <Link
                    to="/admin/onboarding"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/admin/onboarding') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={isActive('/admin/onboarding') ? 'page' : undefined}
                  >
                    Onboarding
                  </Link>
                  <Link
                    to="/admin/contests"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/admin/contests') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={isActive('/admin/contests') ? 'page' : undefined}
                  >
                    Contests
                  </Link>
                </div>
              </nav>
            ) : isFaculty() ? (
              <nav aria-label="Faculty navigation">
                <div className="flex items-center gap-1">
                  <Link
                    to="/faculty"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/faculty') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={isActive('/faculty') ? 'page' : undefined}
                  >
                    Dashboard
                  </Link>
                  <Link
                    to="/faculty/quizzes"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${location.pathname.includes('/faculty/quizzes') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={
                      location.pathname.includes('/faculty/quizzes') ? 'page' : undefined
                    }
                  >
                    Quizzes
                  </Link>
                  <Link
                    to="/faculty/contests"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/faculty/contests') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={isActive('/faculty/contests') ? 'page' : undefined}
                  >
                    Contests
                  </Link>
                </div>
              </nav>
            ) : isHOD() ? (
              <nav aria-label="HOD navigation">
                <div className="flex items-center gap-1">
                  <Link
                    to="/hod"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/hod') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={isActive('/hod') ? 'page' : undefined}
                  >
                    HOD Dashboard
                  </Link>
                  <Link
                    to="/hod/analytics"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/hod/analytics') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={isActive('/hod/analytics') ? 'page' : undefined}
                  >
                    Analytics
                  </Link>
                  <Link
                    to="/hod/faculty"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/hod/faculty') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={isActive('/hod/faculty') ? 'page' : undefined}
                  >
                    Faculty
                  </Link>
                  <Link
                    to="/hod/courses"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/hod/courses') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={isActive('/hod/courses') ? 'page' : undefined}
                  >
                    Courses
                  </Link>
                  <Link
                    to="/hod/contests"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${location.pathname.startsWith('/hod/contests') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={
                      location.pathname.startsWith('/hod/contests') ? 'page' : undefined
                    }
                  >
                    Contests
                  </Link>
                </div>
              </nav>
            ) : isPrincipal() ? (
              <nav aria-label="Principal navigation">
                <div className="flex items-center gap-1">
                  <Link
                    to="/principal"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/principal') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={isActive('/principal') ? 'page' : undefined}
                  >
                    Home
                  </Link>
                  <Link
                    to="/principal/analytics/overview"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${location.pathname.startsWith('/principal/analytics') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={
                      location.pathname.startsWith('/principal/analytics') ? 'page' : undefined
                    }
                  >
                    Analytics
                  </Link>
                  <Link
                    to="/principal/hod-management"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/principal/hod-management') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={isActive('/principal/hod-management') ? 'page' : undefined}
                  >
                    HOD Management
                  </Link>
                </div>
              </nav>
            ) : (
              <nav aria-label="Student navigation">
                <div className="flex items-center gap-1">
                  <Link
                    to="/dashboard"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/dashboard') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={isActive('/dashboard') ? 'page' : undefined}
                  >
                    Dashboard
                  </Link>
                  <Link
                    to="/practice"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/practice') || location.pathname.startsWith('/practice') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={
                      isActive('/practice') || location.pathname.startsWith('/practice')
                        ? 'page'
                        : undefined
                    }
                  >
                    Practice
                  </Link>
                  <Link
                    to="/courses"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/courses') || location.pathname.startsWith('/courses') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={
                      isActive('/courses') || location.pathname.startsWith('/courses')
                        ? 'page'
                        : undefined
                    }
                  >
                    Courses
                  </Link>
                  <Link
                    to="/contests"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/contests') || location.pathname.startsWith('/contests') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={
                      isActive('/contests') || location.pathname.startsWith('/contests')
                        ? 'page'
                        : undefined
                    }
                  >
                    Contests
                  </Link>
                  <Link
                    to="/quizzes"
                    className={`px-4 py-2 text-sm font-medium transition-colors rounded-lg ${isActive('/quizzes') || location.pathname.startsWith('/quizzes') ? 'text-accent-secondary bg-background-tertiary' : 'text-text-secondary hover:text-text-primary hover:bg-background-tertiary'}`}
                    aria-current={
                      isActive('/quizzes') || location.pathname.startsWith('/quizzes')
                        ? 'page'
                        : undefined
                    }
                  >
                    Quizzes
                  </Link>
                </div>
              </nav>
            )}
          </div>

          <div className="flex items-center gap-4">
            <button
              onClick={toggleTheme}
              className="p-2 rounded-lg bg-background-tertiary hover:bg-background-elevated border border-background-border transition-all duration-200"
              aria-label={`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`}
              title={`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`}
              type="button"
            >
              {theme === 'dark' ? (
                <svg
                  className="w-5 h-5 text-text-secondary"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <circle cx="12" cy="12" r="5" />
                  <line x1="12" y1="1" x2="12" y2="3" />
                  <line x1="12" y1="21" x2="12" y2="23" />
                  <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
                  <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
                  <line x1="1" y1="12" x2="3" y2="12" />
                  <line x1="21" y1="12" x2="23" y2="12" />
                  <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
                  <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
                </svg>
              ) : (
                <svg
                  className="w-5 h-5 text-text-secondary"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
                </svg>
              )}
            </button>
            {user ? (
              <div className="flex items-center gap-4">
                {isStudent() && (
                  <div className="hidden sm:flex items-center gap-2 px-3 py-1.5 bg-background-tertiary rounded-lg border border-background-border">
                    <span className="text-sm">🔥</span>
                    <span className="text-sm font-semibold text-text-primary">{streak}</span>
                  </div>
                )}
                <div className="relative flex items-center gap-3" ref={profileMenuRef}>
                  <div className="hidden sm:block text-right">
                    <div className="text-sm font-medium text-text-primary">{user.name}</div>
                    <div className="text-xs text-text-tertiary capitalize">
                      {user.role?.replace('_', ' ')}
                    </div>
                  </div>
                  <button
                    onClick={() => setShowProfileMenu(!showProfileMenu)}
                    className="w-9 h-9 rounded-full bg-gradient-to-br from-accent-secondary to-blue-400 hover:from-blue-500 hover:to-blue-300 transition-all cursor-pointer"
                    aria-label="User menu"
                    aria-expanded={showProfileMenu}
                    aria-haspopup="menu"
                    type="button"
                  />
                  {showProfileMenu && (
                    <div
                      className="absolute right-0 top-full mt-2 w-48 bg-background-secondary border border-background-border rounded-lg shadow-lg overflow-hidden z-50"
                      role="menu"
                      aria-orientation="vertical"
                      aria-labelledby="user-menu-button"
                    >
                      <Link
                        to="/profile"
                        onClick={() => setShowProfileMenu(false)}
                        className="flex items-center gap-3 px-4 py-3 text-sm text-text-primary hover:bg-background-tertiary transition-colors"
                        role="menuitem"
                      >
                        <svg
                          className="w-4 h-4 text-text-secondary"
                          viewBox="0 0 24 24"
                          fill="none"
                          stroke="currentColor"
                          strokeWidth="2"
                        >
                          <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
                          <circle cx="12" cy="7" r="4" />
                        </svg>
                        Profile
                      </Link>
                      <button
                        onClick={handleLogout}
                        className="flex items-center gap-3 w-full px-4 py-3 text-sm text-red-400 hover:bg-background-tertiary transition-colors border-t border-background-border cursor-pointer"
                        role="menuitem"
                      >
                        <svg
                          className="w-4 h-4"
                          viewBox="0 0 24 24"
                          fill="none"
                          stroke="currentColor"
                          strokeWidth="2"
                        >
                          <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
                          <polyline points="16 17 21 12 16 7" />
                          <line x1="21" y1="12" x2="9" y2="12" />
                        </svg>
                        Logout
                      </button>
                    </div>
                  )}
                </div>
              </div>
            ) : (
              <div className="flex items-center gap-3">
                <Link to="/login" className="btn btn-primary">
                  Login
                </Link>
              </div>
            )}
          </div>
        </div>
      </nav>
    </>
  );
}

export default Navbar;
