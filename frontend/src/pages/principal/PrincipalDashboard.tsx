import { Outlet, Link, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  GitBranch,
  BookOpen,
  FlaskConical,
  Trophy,
  GraduationCap,
  Calendar,
  Lightbulb,
  Bell,
  Award,
  Users,
} from 'lucide-react';

const navItems = [
  { path: '/principal', label: 'Overview', icon: LayoutDashboard },
  { path: '/principal/analytics/branches', label: 'Branch Performance', icon: GitBranch },
  { path: '/principal/analytics/courses', label: 'Course Analytics', icon: BookOpen },
  { path: '/principal/analytics/labs', label: 'Lab Analytics', icon: FlaskConical },
  { path: '/principal/analytics/contests', label: 'Contest Analytics', icon: Trophy },
  { path: '/principal/analytics/students', label: 'Student Insights', icon: GraduationCap },
  { path: '/principal/analytics/years', label: 'Year Trends', icon: Calendar },
  { path: '/principal/analytics/insights', label: 'AI Insights', icon: Lightbulb },
  { path: '/principal/analytics/alerts', label: 'Risk Alerts', icon: Bell },
  { path: '/principal/analytics/accreditation', label: 'Accreditation', icon: Award },
  { path: '/principal/hod-management', label: 'HOD Management', icon: Users },
];

const PrincipalDashboard = () => {
  const location = useLocation();

  const isActive = (path: string) => {
    if (path === '/principal') {
      return location.pathname === '/principal';
    }
    return location.pathname.startsWith(path);
  };

  return (
    <div className="min-h-[calc(100vh-64px)] bg-background-primary flex">
      {/* Sidebar */}
      <aside className="w-64 bg-background-elevated border-r border-background-border hidden lg:block">
        <div className="p-4">
          <h2 className="text-sm font-semibold text-text-muted uppercase tracking-wider mb-4 px-2">
            Analytics
          </h2>
          <nav className="space-y-1">
            {navItems.map((item) => {
              const Icon = item.icon;
              const active = isActive(item.path);
              return (
                <Link
                  key={item.path}
                  to={item.path}
                  className={`flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all ${
                    active
                      ? 'bg-accent-primary/10 text-accent-primary border-l-2 border-accent-primary'
                      : 'text-text-secondary hover:bg-background-tertiary hover:text-text-primary'
                  }`}
                >
                  <Icon className="w-4 h-4" />
                  {item.label}
                </Link>
              );
            })}
          </nav>
        </div>
      </aside>

      {/* Mobile Navigation */}
      <div className="lg:hidden fixed bottom-0 left-0 right-0 bg-background-elevated border-t border-background-border z-50">
        <nav className="flex overflow-x-auto px-2 py-2 gap-1">
          {navItems.map((item) => {
            const Icon = item.icon;
            const active = isActive(item.path);
            return (
              <Link
                key={item.path}
                to={item.path}
                className={`flex flex-col items-center gap-1 px-3 py-2 rounded-lg text-xs font-medium whitespace-nowrap transition-all ${
                  active ? 'bg-accent-primary/10 text-accent-primary' : 'text-text-secondary'
                }`}
              >
                <Icon className="w-5 h-5" />
                {item.label}
              </Link>
            );
          })}
        </nav>
      </div>

      {/* Main Content */}
      <main className="flex-1 p-6 pb-24 lg:pb-6">
        <Outlet />
      </main>
    </div>
  );
};

export default PrincipalDashboard;
