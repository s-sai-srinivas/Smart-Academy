import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { principalAPI } from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import { showError } from '../../utils/showAlert';

interface PrincipalStats {
  total_faculty: number;
  total_hods: number;
  total_departments: number;
  total_students: number;
}

const PrincipalHome = () => {
  const { user } = useAuth();
  const [loading, setLoading] = useState<boolean>(true);
  const [stats, setStats] = useState<PrincipalStats>({
    total_faculty: 0,
    total_hods: 0,
    total_departments: 0,
    total_students: 0,
  });

  useEffect(() => {
    fetchStats();
  }, []);

  const fetchStats = async () => {
    try {
      const response = (await principalAPI.getStats()) as PrincipalStats;
      setStats(response);
    } catch (error) {
      console.error('Error fetching principal stats:', error);
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
          Welcome, {user?.name || 'Principal'}
        </h1>
        <p className="text-text-secondary text-sm">College Management & HOD Assignment</p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        <div className="card-elevated flex items-center gap-4 border-l-4 border-accent-secondary">
          <div className="flex-1">
            <div className="text-3xl font-bold text-text-primary">{stats.total_departments}</div>
            <div className="text-sm text-text-secondary mt-1">Departments</div>
          </div>
        </div>

        <div className="card-elevated flex items-center gap-4 border-l-4 border-accent-success">
          <div className="flex-1">
            <div className="text-3xl font-bold text-text-primary">{stats.total_hods}</div>
            <div className="text-sm text-text-secondary mt-1">HODs Assigned</div>
          </div>
        </div>

        <div className="card-elevated flex items-center gap-4 border-l-4 border-accent-warning">
          <div className="flex-1">
            <div className="text-3xl font-bold text-text-primary">{stats.total_faculty}</div>
            <div className="text-sm text-text-secondary mt-1">Total Faculty</div>
          </div>
        </div>

        <div className="card-elevated flex items-center gap-4 border-l-4 border-accent-info">
          <div className="flex-1">
            <div className="text-3xl font-bold text-text-primary">{stats.total_students}</div>
            <div className="text-sm text-text-secondary mt-1">Total Students</div>
          </div>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="card">
        <h2 className="text-xl font-semibold text-text-primary mb-4">Quick Actions</h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <Link
            to="/principal/hod-management"
            className="flex items-center gap-3 p-4 bg-background-tertiary rounded-lg border border-background-border hover:border-accent-secondary/50 hover:bg-background-elevated transition-all"
          >
            <span className="text-2xl">🎓</span>
            <div>
              <div className="font-medium text-text-primary">HOD Management</div>
              <div className="text-sm text-text-secondary">
                Assign or update HODs for departments
              </div>
            </div>
          </Link>

          <Link
            to="/principal/analytics/overview"
            className="flex items-center gap-3 p-4 bg-background-tertiary rounded-lg border border-background-border hover:border-accent-secondary/50 hover:bg-background-elevated transition-all"
          >
            <span className="text-2xl">📊</span>
            <div>
              <div className="font-medium text-text-primary">College Analytics</div>
              <div className="text-sm text-text-secondary">
                View institution-wide analytics and insights
              </div>
            </div>
          </Link>
        </div>
      </div>

      {/* Info Card */}
      <div className="card border-l-4 border-accent-info">
        <h3 className="text-lg font-semibold text-text-primary mb-2">Important Notes</h3>
        <ul className="text-sm text-text-secondary space-y-1.5 list-disc list-inside">
          <li>Faculty must be onboarded by the admin before they can be assigned as HOD.</li>
          <li>Each department can have only one HOD at a time.</li>
          <li>A faculty member cannot be HOD of multiple departments.</li>
          <li>
            Reassigning an HOD will automatically demote the previous HOD back to faculty role.
          </li>
        </ul>
      </div>
    </div>
  );
};

export default PrincipalHome;
