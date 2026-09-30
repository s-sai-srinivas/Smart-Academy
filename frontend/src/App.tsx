import { type ReactNode } from 'react';
import { BrowserRouter, Routes, Route, Navigate, useLocation, useNavigate } from 'react-router-dom';
import { useEffect, useRef } from 'react';
import { AuthProvider, useAuth } from './context/AuthContext';
import { NotificationProvider } from './context/NotificationContext';
import { ContestModeProvider, useContestMode } from './context/ContestModeContext';
import { WebSocketProvider } from './context/WebSocketContext';
import { ThemeProvider } from './context/ThemeContext';
import Navbar from './components/common/Navbar';
import LandingPage from './pages/LandingPage';
import LoginPage from './pages/LoginPage';
import PlagiarismPage from './pages/admin/PlagiarismPage';
import DashboardPage from './pages/student/DashboardPage';
import CoursesPage from './pages/student/CoursesPage';
import CourseDetailPage from './pages/student/CourseDetailPage';
import TopicProblemsPage from './pages/student/TopicProblemsPage';
import ProblemSolvingPage from './pages/student/ProblemSolvingPage';
import PracticePage from './pages/student/PracticePage';
import PracticeProblemPage from './pages/student/PracticeProblemPage';
import CourseDashboard from './pages/faculty/CourseDashboard';
import StudentAnalyticsPage from './pages/faculty/StudentAnalyticsPage';
import AdminDashboard from './pages/admin/AdminDashboard';
import AdminHome from './pages/admin/AdminHome';
import AdminCourses from './pages/admin/AdminCourses';
import AdminLabs from './pages/admin/AdminLabs';
import AdminTheory from './pages/admin/AdminTheory';
import AdminLabSessionProblems from './pages/admin/AdminLabSessionProblems';
import AdminOnboarding from './pages/admin/AdminOnboarding';
import AdminContests from './pages/admin/AdminContests';
import AdminContestProblems from './pages/admin/AdminContestProblems';
import ContestGenerationRequest from './pages/admin/ContestGenerationRequest';
import ReviewGeneratedContent from './pages/admin/ReviewGeneratedContent';
import SuperAdminDashboard from './pages/super-admin/SuperAdminDashboard';
import SuperAdminHome from './pages/super-admin/SuperAdminHome';
import SuperAdminProblems from './pages/super-admin/SuperAdminProblems';
import SuperAdminTopics from './pages/super-admin/SuperAdminTopics';
import FacultyDashboard from './pages/faculty/FacultyDashboard';
import FacultyHome from './pages/faculty/FacultyHome';
import FacultyContests from './pages/faculty/FacultyContests';
import ContestDetailView from './pages/faculty/ContestDetailView';
import ContestPlagiarismTab from './pages/faculty/ContestPlagiarismTab';
import QuizManagementPage from './pages/faculty/QuizManagementPage';
import QuizCreatePage from './pages/faculty/QuizCreatePage';
import QuestionEditorPage from './pages/faculty/QuestionEditorPage';
import QuizAnalyticsPage from './pages/faculty/QuizAnalyticsPage';
import HODDashboard from './pages/hod/HODDashboard';
import HODHome from './pages/hod/HODHome';
import HODAnalytics from './pages/hod/HODAnalytics';
import HODFaculty from './pages/hod/HODFaculty';
import HODCourses from './pages/hod/HODCourses';
import HODContests from './pages/hod/HODContests';
import PrincipalDashboard from './pages/principal/PrincipalDashboard';
import PrincipalHome from './pages/principal/PrincipalHome';
import HODManagement from './pages/principal/HODManagement';
import PrincipalOverview from './pages/principal/analytics/PrincipalOverview';
import BranchAnalytics from './pages/principal/analytics/BranchAnalytics';
import CourseAnalytics from './pages/principal/analytics/CourseAnalytics';
import LabAnalytics from './pages/principal/analytics/LabAnalytics';
import ContestAnalytics from './pages/principal/analytics/ContestAnalytics';
import StudentAnalytics from './pages/principal/analytics/StudentAnalytics';
import YearAnalytics from './pages/principal/analytics/YearAnalytics';
import Insights from './pages/principal/analytics/Insights';
import Alerts from './pages/principal/analytics/Alerts';
import Accreditation from './pages/principal/analytics/Accreditation';
import ContestsPage from './pages/student/ContestsPage';
import ContestProblemsPage from './pages/student/ContestProblemsPage';
import ContestProblemPage from './pages/student/ContestProblemPage';
import ContestLeaderboardPage from './pages/student/ContestLeaderboardPage';
import ProfilePage from './pages/ProfilePage';
import QuizListPage from './pages/student/QuizListPage';
import QuizTakingPage from './pages/student/QuizTakingPage';
import QuizResultPage from './pages/student/QuizResultPage';
import './index.css';

interface RouteGuardProps {
  children: ReactNode;
}

interface RoleChecks {
  isSuperAdmin: () => boolean;
  isPrincipal: () => boolean;
  isHOD: () => boolean;
  isAdmin: () => boolean;
  isFaculty: () => boolean;
}

function ProtectedRoute({ children }: RouteGuardProps) {
  const { isAuthenticated } = useAuth();
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }
  return children;
}

const getPrimaryRole = (checks: RoleChecks): string => {
  if (checks.isSuperAdmin()) return 'super_admin';
  if (checks.isPrincipal()) return 'principal';
  if (checks.isHOD()) return 'hod';
  if (checks.isAdmin()) return 'admin';
  if (checks.isFaculty()) return 'faculty';
  return 'student';
};

function StudentRoute({ children }: RouteGuardProps) {
  const { isSuperAdmin, isHOD, isPrincipal, isAdmin, isFaculty, isAuthenticated } = useAuth();
  const checks = { isSuperAdmin, isHOD, isPrincipal, isAdmin, isFaculty };

  if (!isAuthenticated) {
    return <Navigate to="/login?message=Please login to access this page" replace />;
  }

  const primaryRole = getPrimaryRole(checks);
  if (primaryRole !== 'student') {
    return <Navigate to={`/${primaryRole === 'admin' ? 'admin' : primaryRole}`} replace />;
  }

  return children;
}

function AdminRoute({ children }: RouteGuardProps) {
  const { isSuperAdmin, isHOD, isPrincipal, isAdmin, isFaculty, isAuthenticated } = useAuth();

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  const checks = { isSuperAdmin, isHOD, isPrincipal, isAdmin, isFaculty };
  const primaryRole = getPrimaryRole(checks);

  if (primaryRole !== 'admin') {
    return (
      <Navigate
        to={`/${primaryRole === 'super_admin' ? 'super-admin' : primaryRole === 'admin' ? 'admin' : primaryRole}`}
        replace
      />
    );
  }

  return children;
}

function SuperAdminRoute({ children }: RouteGuardProps) {
  const { isAuthenticated, isSuperAdmin } = useAuth();

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  if (!isSuperAdmin()) {
    return <Navigate to="/dashboard" replace />;
  }

  return children;
}

function HODRoute({ children }: RouteGuardProps) {
  const { isAuthenticated, isHOD } = useAuth();

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  if (!isHOD()) {
    return <Navigate to="/dashboard" replace />;
  }

  return children;
}

function FacultyRoute({ children }: RouteGuardProps) {
  const { isAuthenticated, isFaculty } = useAuth();

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  if (!isFaculty()) {
    return <Navigate to="/dashboard" replace />;
  }

  return children;
}

function PrincipalRoute({ children }: RouteGuardProps) {
  const { isAuthenticated, isPrincipal } = useAuth();

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  if (!isPrincipal()) {
    return <Navigate to="/dashboard" replace />;
  }

  return children;
}

function ContestNavigationGuard() {
  const { isInContestMode, isInContestModeRef, isAllowedPath, getRedirectPath } = useContestMode();
  const location = useLocation();
  const navigate = useNavigate();
  const prevPathRef = useRef<string>(location.pathname);

  useEffect(() => {
    if (!isInContestModeRef.current) return;

    const handlePopState = (e: PopStateEvent) => {
      const currentPath = window.location.pathname;
      if (!isAllowedPath(currentPath)) {
        e.preventDefault();
        window.history.pushState(null, '', prevPathRef.current || getRedirectPath());
        alert(
          'You cannot leave the contest while it is active. Please click "Finish Contest" to exit.'
        );
      }
    };

    window.history.pushState(null, '', window.location.href);
    window.addEventListener('popstate', handlePopState);

    return () => {
      window.removeEventListener('popstate', handlePopState);
    };
  }, [isInContestModeRef, isAllowedPath, getRedirectPath]);

  useEffect(() => {
    if (!isInContestModeRef.current) return;

    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      e.returnValue =
        'You are in an active contest. Leaving will count as a violation. Are you sure you want to leave?';
      return e.returnValue;
    };

    window.addEventListener('beforeunload', handleBeforeUnload);

    return () => {
      window.removeEventListener('beforeunload', handleBeforeUnload);
    };
  }, [isInContestModeRef]);

  useEffect(() => {
    if (isInContestModeRef.current && isAllowedPath(location.pathname)) {
      prevPathRef.current = location.pathname;
    }
  }, [location.pathname, isInContestModeRef, isAllowedPath]);

  useEffect(() => {
    if (isInContestMode && !isAllowedPath(location.pathname)) {
      navigate(getRedirectPath(), { replace: true });
    }
  }, [isInContestMode, location.pathname, isAllowedPath, getRedirectPath, navigate]);

  return null;
}

function AppContent() {
  const location = useLocation();
  const { isInContestMode, isAllowedPath } = useContestMode();
  const isLandingPage = location.pathname === '/' || location.pathname === '/login';

  const isContestProblemPage = /^\/contests\/[^/]+\/problems(?:\/[^/]+)?$/.test(location.pathname);
  const hideNavbar =
    isLandingPage || (isInContestMode && !isAllowedPath(location.pathname)) || isContestProblemPage;

  // Prevent Backspace from navigating back in browser history (SPA behavior)
  // Some browsers/extensions still map Backspace to history.back(). We intercept
  // it globally and manually handle text deletion so inputs keep working.
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key !== 'Backspace') return;

      const target = e.target as HTMLElement;
      const tagName = target.tagName.toLowerCase();
      const isEditable =
        target.isContentEditable ||
        tagName === 'input' ||
        tagName === 'textarea' ||
        tagName === 'select';

      if (!isEditable) {
        e.preventDefault();
        return;
      }

      // For inputs/textareas, prevent the browser's default (which may include
      // history navigation in some configurations) and perform deletion ourselves.
      if (tagName === 'input' || tagName === 'textarea') {
        const el = target as HTMLInputElement | HTMLTextAreaElement;
        const type = el.type;
        // Skip non-textual inputs where Backspace has no text-editing meaning
        if (
          type === 'checkbox' ||
          type === 'radio' ||
          type === 'file' ||
          type === 'submit' ||
          type === 'button' ||
          type === 'image' ||
          type === 'reset' ||
          type === 'range' ||
          type === 'color'
        ) {
          e.preventDefault();
          return;
        }

        e.preventDefault();

        const start = el.selectionStart ?? 0;
        const end = el.selectionEnd ?? 0;
        const value = el.value;

        let newValue: string;
        let newCursor: number;

        if (start !== end) {
          // Delete selected text
          newValue = value.slice(0, start) + value.slice(end);
          newCursor = start;
        } else if (start > 0) {
          // Delete one character before cursor
          newValue = value.slice(0, start - 1) + value.slice(start);
          newCursor = start - 1;
        } else {
          // Cursor at beginning, nothing to delete
          return;
        }

        el.value = newValue;
        el.setSelectionRange(newCursor, newCursor);

        // Dispatch input event so React controlled components stay in sync
        el.dispatchEvent(new Event('input', { bubbles: true }));
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, []);

  return (
    <>
      <ContestNavigationGuard />
      {!hideNavbar && <Navbar />}
      <Routes>
        <Route path="/" element={<LandingPage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/profile"
          element={
            <ProtectedRoute>
              <ProfilePage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/dashboard"
          element={
            <StudentRoute>
              <DashboardPage />
            </StudentRoute>
          }
        />
        <Route
          path="/courses"
          element={
            <StudentRoute>
              <CoursesPage />
            </StudentRoute>
          }
        />
        <Route
          path="/courses/:courseId"
          element={
            <StudentRoute>
              <CourseDetailPage />
            </StudentRoute>
          }
        />
        <Route
          path="/courses/:courseId/lab/:topicId"
          element={
            <StudentRoute>
              <TopicProblemsPage />
            </StudentRoute>
          }
        />
        <Route
          path="/lab/:topicId/problem/:problemId"
          element={
            <StudentRoute>
              <ProblemSolvingPage />
            </StudentRoute>
          }
        />
        <Route
          path="/practice"
          element={
            <StudentRoute>
              <PracticePage />
            </StudentRoute>
          }
        />
        <Route
          path="/practice/:problemId"
          element={
            <StudentRoute>
              <PracticeProblemPage />
            </StudentRoute>
          }
        />
        <Route
          path="/faculty/course/:id/dashboard"
          element={
            <FacultyRoute>
              <CourseDashboard />
            </FacultyRoute>
          }
        />
        <Route
          path="/faculty/student/:id/analytics"
          element={
            <FacultyRoute>
              <StudentAnalyticsPage />
            </FacultyRoute>
          }
        />
        <Route
          path="/contests"
          element={
            <StudentRoute>
              <ContestsPage />
            </StudentRoute>
          }
        />
        <Route
          path="/contests/:id"
          element={
            <StudentRoute>
              <ContestsPage />
            </StudentRoute>
          }
        />
        <Route
          path="/contests/:id/problems"
          element={
            <StudentRoute>
              <ContestProblemsPage />
            </StudentRoute>
          }
        />
        <Route
          path="/contests/:id/problems/:problemId"
          element={
            <StudentRoute>
              <ContestProblemPage />
            </StudentRoute>
          }
        />
        <Route
          path="/contests/:id/leaderboard"
          element={
            <StudentRoute>
              <ContestLeaderboardPage />
            </StudentRoute>
          }
        />
        <Route path="/plagiarism" element={<PlagiarismPage />} />
        <Route
          path="/admin"
          element={
            <AdminRoute>
              <AdminDashboard />
            </AdminRoute>
          }
        >
          <Route index element={<AdminHome />} />
          <Route path="courses" element={<AdminCourses />} />
          <Route path="labs" element={<AdminLabs />} />
          <Route path="lab-sessions/:sessionId/problems" element={<AdminLabSessionProblems />} />
          <Route path="theory" element={<AdminTheory />} />
          <Route path="onboarding" element={<AdminOnboarding />} />
          <Route path="contests" element={<AdminContests />} />
          <Route path="contests/:contestId/problems" element={<AdminContestProblems />} />
          <Route path="contests/generate" element={<ContestGenerationRequest />} />
          <Route path="review-generated-content" element={<ReviewGeneratedContent />} />
        </Route>
        <Route
          path="/super-admin"
          element={
            <SuperAdminRoute>
              <SuperAdminDashboard />
            </SuperAdminRoute>
          }
        >
          <Route index element={<SuperAdminHome />} />
          <Route path="problems" element={<SuperAdminProblems />} />
          <Route path="topics" element={<SuperAdminTopics />} />
        </Route>
        <Route
          path="/faculty"
          element={
            <FacultyRoute>
              <FacultyDashboard />
            </FacultyRoute>
          }
        >
          <Route index element={<FacultyHome />} />
          <Route path="contests" element={<FacultyContests />} />
          <Route path="contests/:id" element={<ContestDetailView />} />
          <Route path="contests/:id/plagiarism" element={<ContestPlagiarismTab />} />
          <Route path="quizzes" element={<QuizManagementPage />} />
          <Route path="quizzes/create" element={<QuizCreatePage />} />
          <Route path="quizzes/create/manual" element={<QuestionEditorPage />} />
          <Route path="quizzes/:quizId/edit" element={<QuizCreatePage />} />
          <Route path="quizzes/:quizId/questions" element={<QuestionEditorPage />} />
          <Route path="quizzes/:quizId/analytics" element={<QuizAnalyticsPage />} />
          <Route path="course/:id/dashboard" element={<CourseDashboard />} />
        </Route>
        <Route
          path="/hod"
          element={
            <HODRoute>
              <HODDashboard />
            </HODRoute>
          }
        >
          <Route index element={<HODHome />} />
          <Route path="analytics" element={<HODAnalytics />} />
          <Route path="faculty" element={<HODFaculty />} />
          <Route path="courses" element={<HODCourses />} />
          <Route path="contests" element={<HODContests />} />
          <Route path="contests/:id" element={<ContestDetailView />} />
          <Route path="contests/:id/plagiarism" element={<ContestPlagiarismTab />} />
        </Route>
        <Route
          path="/principal"
          element={
            <PrincipalRoute>
              <PrincipalDashboard />
            </PrincipalRoute>
          }
        >
          <Route index element={<PrincipalHome />} />
          <Route path="hod-management" element={<HODManagement />} />
          <Route path="analytics/overview" element={<PrincipalOverview />} />
          <Route path="analytics/branches" element={<BranchAnalytics />} />
          <Route path="analytics/courses" element={<CourseAnalytics />} />
          <Route path="analytics/labs" element={<LabAnalytics />} />
          <Route path="analytics/contests" element={<ContestAnalytics />} />
          <Route path="analytics/students" element={<StudentAnalytics />} />
          <Route path="analytics/years" element={<YearAnalytics />} />
          <Route path="analytics/insights" element={<Insights />} />
          <Route path="analytics/alerts" element={<Alerts />} />
          <Route path="analytics/accreditation" element={<Accreditation />} />
        </Route>
        <Route
          path="/quizzes"
          element={
            <StudentRoute>
              <QuizListPage />
            </StudentRoute>
          }
        />
        <Route
          path="/quizzes/:id"
          element={
            <StudentRoute>
              <QuizTakingPage />
            </StudentRoute>
          }
        />
        <Route
          path="/quizzes/:id/result"
          element={
            <StudentRoute>
              <QuizResultPage />
            </StudentRoute>
          }
        />
      </Routes>
    </>
  );
}

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <NotificationProvider>
          <WebSocketProvider>
            <ContestModeProvider>
              <BrowserRouter>
                <AppContent />
              </BrowserRouter>
            </ContestModeProvider>
          </WebSocketProvider>
        </NotificationProvider>
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App;
