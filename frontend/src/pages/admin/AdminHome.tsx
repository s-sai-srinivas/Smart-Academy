import { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import api from '../../services/api';

interface AdminStats {
  total_courses: number;
  total_lab_sessions: number;
  total_problems: number;
  total_students: number;
}

const AdminHome: React.FC = () => {
  const { user } = useAuth();
  const [stats, setStats] = useState<AdminStats>({
    total_courses: 0,
    total_lab_sessions: 0,
    total_problems: 0,
    total_students: 0,
  });
  const [loading, setLoading] = useState<boolean>(true);

  const fetchStats = useCallback(async (): Promise<void> => {
    try {
      const data = await api.get('/admin/stats');
      setStats(data as unknown as AdminStats);
    } catch (error) {
      console.error('Error fetching admin stats:', error);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  const statCards = [
    {
      label: 'Total Courses',
      value: stats.total_courses,
      color: 'border-l-4 border-accent-secondary',
    },
    {
      label: 'Lab Sessions',
      value: stats.total_lab_sessions,
      color: 'border-l-4 border-accent-success',
    },
    { label: 'Problems', value: stats.total_problems, color: 'border-l-4 border-accent-warning' },
    { label: 'Students', value: stats.total_students, color: 'border-l-4 border-accent-primary' },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-text-primary mb-1">
          Welcome, {user?.name || 'Admin'}
        </h1>
        <p className="text-text-secondary text-sm">
          Manage your college courses, lab sessions, and problems
        </p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        {statCards.map((stat) => (
          <div key={stat.label} className={`card-elevated flex items-center gap-4 ${stat.color}`}>
            <div className="flex-1">
              <div className="text-3xl font-bold text-text-primary">
                {loading ? '...' : stat.value}
              </div>
              <div className="text-sm text-text-secondary mt-1">{stat.label}</div>
            </div>
          </div>
        ))}
      </div>

      {/* Quick Actions */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4">Quick Actions</h2>
        <div className="flex flex-wrap gap-3">
          <Link to="/admin/courses" className="btn btn-primary">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 4v16m8-8H4"
              />
            </svg>
            Create New Course
          </Link>
          <Link to="/admin/contests" className="btn btn-primary">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
            Manage Contests
          </Link>
          <Link to="/admin/labs" className="btn btn-secondary">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
              />
            </svg>
            Create New Problem
          </Link>
          <Link to="/admin/labs" className="btn btn-secondary">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
              />
            </svg>
            Schedule Lab Session
          </Link>
        </div>
      </div>
    </div>
  );
};

export default AdminHome;
