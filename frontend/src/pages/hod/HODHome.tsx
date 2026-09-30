import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import api from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import { showError } from '../../utils/showAlert';

interface DashboardStats {
  total_students: number;
  total_faculty: number;
  total_courses: number;
  total_submissions: number;
  active_students_24h: number;
}

const HODHome = () => {
  const { user } = useAuth();
  const [loading, setLoading] = useState<boolean>(true);
  const [stats, setStats] = useState<DashboardStats>({
    total_students: 0,
    total_faculty: 0,
    total_courses: 0,
    total_submissions: 0,
    active_students_24h: 0,
  });

  useEffect(() => {
    fetchDashboardStats();
  }, []);

  const fetchDashboardStats = async () => {
    try {
      const response = (await api.get('/hod/stats')) as DashboardStats;
      setStats(response);
    } catch (error) {
      console.error('Error fetching HOD stats:', error);
      showError('Failed to load dashboard stats');
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="flex items-center gap-3">
          <svg
            className="animate-spin h-5 w-5 text-accent-primary"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              className="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="4"
            ></circle>
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            ></path>
          </svg>
          <span className="text-text-secondary">Loading...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-text-primary mb-1">
          Welcome, {user?.name || 'HOD'}
        </h1>
        <p className="text-text-secondary text-sm">Branch Analytics & Performance Overview</p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
        <div className="card-elevated flex items-center gap-4 border-l-4 border-accent-secondary">
          <div className="flex-1">
            <div className="text-3xl font-bold text-text-primary">{stats.total_students}</div>
            <div className="text-sm text-text-secondary mt-1">Total Students</div>
          </div>
        </div>

        <div className="card-elevated flex items-center gap-4 border-l-4 border-accent-success">
          <div className="flex-1">
            <div className="text-3xl font-bold text-text-primary">{stats.total_faculty}</div>
            <div className="text-sm text-text-secondary mt-1">Faculty Members</div>
          </div>
        </div>

        <div className="card-elevated flex items-center gap-4 border-l-4 border-accent-warning">
          <div className="flex-1">
            <div className="text-3xl font-bold text-text-primary">{stats.total_courses}</div>
            <div className="text-sm text-text-secondary mt-1">Courses</div>
          </div>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4">Quick Actions</h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          <Link
            to="/hod/analytics"
            className="flex items-center gap-3 p-4 bg-background-tertiary rounded-lg border border-background-border hover:border-accent-secondary/50 hover:bg-background-elevated transition-all"
          >
            <div>
              <div className="font-medium text-text-primary">View Analytics</div>
              <div className="text-sm text-text-secondary">Detailed branch metrics</div>
            </div>
          </Link>

          <Link
            to="/hod/faculty"
            className="flex items-center gap-3 p-4 bg-background-tertiary rounded-lg border border-background-border hover:border-accent-secondary/50 hover:bg-background-elevated transition-all"
          >
            <span className="text-2xl">👨‍🏫</span>
            <div>
              <div className="font-medium text-text-primary">Manage Faculty</div>
              <div className="text-sm text-text-secondary">Assign courses & roles</div>
            </div>
          </Link>

          <Link
            to="/hod/courses"
            className="flex items-center gap-3 p-4 bg-background-tertiary rounded-lg border border-background-border hover:border-accent-secondary/50 hover:bg-background-elevated transition-all"
          >
            <div>
              <div className="font-medium text-text-primary">View Courses</div>
              <div className="text-sm text-text-secondary">All branch courses</div>
            </div>
          </Link>
        </div>
      </div>
    </div>
  );
};

export default HODHome;
