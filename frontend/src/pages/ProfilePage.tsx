import { useState, useEffect, useCallback } from 'react';
import { useAuth } from '../context/AuthContext';
import { Link } from 'react-router-dom';
import { profileAPI } from '../services/api';
import { showError } from '../utils/showAlert';

interface StudentProfile {
  branch_name?: string;
  cohort_year?: number;
  admission_year?: number;
  section_name?: string;
  progress_index?: number;
  status?: string;
}

interface FacultyProfile {
  branch_name?: string;
  designation?: string;
  joining_date?: string;
  is_active?: boolean;
}

interface Stats {
  total_solved?: number;
  current_streak?: number;
  accuracy?: number;
  college_rank?: number;
}

interface ProfileUser {
  id?: number;
  name?: string;
  email?: string;
  role?: string;
  regdno?: string;
  phone?: string;
  is_active?: boolean;
  created_at?: string;
}

interface ProfileData {
  user?: ProfileUser;
  student?: StudentProfile;
  faculty?: FacultyProfile;
  stats?: Stats;
}

function ProfilePage() {
  const { user } = useAuth();
  const [profileData, setProfileData] = useState<ProfileData | null>(null);
  const [loading, setLoading] = useState<boolean>(true);

  const loadProfileData = useCallback(async () => {
    if (!user) {
      setLoading(false);
      return;
    }

    try {
      const data = (await profileAPI.getProfile()) as ProfileData;
      setProfileData(data);
    } catch (err) {
      console.error('Failed to load profile:', err);
      showError('Failed to load profile');
    } finally {
      setLoading(false);
    }
  }, [user]);

  useEffect(() => {
    loadProfileData();
  }, [loadProfileData]);

  const getInitials = (name?: string): string => {
    if (!name) return 'U';
    return name
      .split(' ')
      .map((word) => word[0])
      .join('')
      .toUpperCase()
      .slice(0, 2);
  };

  const getAvatarGradient = (name?: string): string => {
    const colors = [
      'from-purple-500 to-blue-500',
      'from-green-500 to-teal-500',
      'from-orange-500 to-red-500',
      'from-pink-500 to-purple-500',
      'from-blue-500 to-cyan-500',
      'from-yellow-500 to-orange-500',
    ];
    const index = name ? name.charCodeAt(0) % colors.length : 0;
    return colors[index];
  };

  const formatDate = (dateString?: string): string => {
    if (!dateString) return 'N/A';
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-background-primary flex items-center justify-center">
        <div className="text-text-muted">Loading profile...</div>
      </div>
    );
  }

  const userData: ProfileUser | null = profileData?.user || (user as ProfileUser | null);

  return (
    <div className="min-h-screen bg-background-primary">
      {/* Header */}
      <div className="bg-background-secondary border-b border-border">
        <div className="max-w-4xl mx-auto px-6 py-8">
          <Link
            to={
              userData?.role === 'student'
                ? '/dashboard'
                : userData?.role === 'faculty'
                  ? '/faculty'
                  : userData?.role === 'admin' || userData?.role === 'college_admin'
                    ? '/admin'
                    : userData?.role === 'super_admin'
                      ? '/super-admin'
                      : userData?.role === 'hod'
                        ? '/hod'
                        : userData?.role === 'principal'
                          ? '/principal'
                          : '/'
            }
            className="inline-flex items-center gap-2 text-text-secondary hover:text-text-primary mb-6 transition-colors"
          >
            <svg
              className="w-4 h-4"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
            >
              <path d="M19 12H5M12 19l-7-7 7-7" />
            </svg>
            Back to Dashboard
          </Link>

          {/* Profile Header */}
          <div className="flex flex-col sm:flex-row items-center sm:items-start gap-6">
            {/* Avatar */}
            <div
              className={`w-24 h-24 rounded-full bg-gradient-to-br ${getAvatarGradient(userData?.name)} flex items-center justify-center text-white text-3xl font-bold flex-shrink-0`}
            >
              {getInitials(userData?.name)}
            </div>

            {/* User Info */}
            <div className="text-center sm:text-left flex-1">
              <h1 className="text-2xl font-bold text-text-primary">{userData?.name || 'User'}</h1>
              <p className="text-text-secondary capitalize">
                {userData?.role?.replace('_', ' ') || 'Student'}
              </p>
              <p className="text-text-tertiary text-sm mt-1">
                Member since {formatDate(userData?.created_at)}
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Profile Details */}
      <div className="max-w-4xl mx-auto px-6 py-8">
        <div className="grid gap-6">
          {/* Personal Information */}
          <div className="card">
            <h2 className="text-lg font-semibold text-text-primary mb-4">Personal Information</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <InfoRow label="Registration Number" value={userData?.regdno || 'N/A'} />
              <InfoRow label="Email" value={userData?.email || 'N/A'} />
              <InfoRow label="Phone" value={userData?.phone || 'Not provided'} />
              <InfoRow label="Account Status" value={userData?.is_active ? 'Active' : 'Inactive'} />
            </div>
          </div>

          {/* Academic Information (for students) */}
          {userData?.role === 'student' && profileData?.student && (
            <div className="card">
              <h2 className="text-lg font-semibold text-text-primary mb-4">Academic Information</h2>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <InfoRow label="Branch" value={profileData.student.branch_name || 'N/A'} />
                <InfoRow label="Cohort Year" value={profileData.student.cohort_year || 'N/A'} />
                <InfoRow
                  label="Admission Year"
                  value={profileData.student.admission_year || 'N/A'}
                />
                <InfoRow
                  label="Section"
                  value={profileData.student.section_name || 'Not assigned'}
                />
                <InfoRow
                  label="Current Year"
                  value={
                    profileData.student.progress_index
                      ? `Year ${profileData.student.progress_index}`
                      : 'N/A'
                  }
                />
                <InfoRow label="Status" value={profileData.student.status || 'N/A'} />
              </div>
            </div>
          )}

          {/* Faculty Information (for faculty) */}
          {userData?.role === 'faculty' && profileData?.faculty && (
            <div className="card">
              <h2 className="text-lg font-semibold text-text-primary mb-4">
                Employment Information
              </h2>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <InfoRow
                  label="Department/Branch"
                  value={profileData.faculty.branch_name || 'N/A'}
                />
                <InfoRow label="Designation" value={profileData.faculty.designation || 'N/A'} />
                <InfoRow
                  label="Joining Date"
                  value={formatDate(profileData.faculty.joining_date)}
                />
                <InfoRow
                  label="Status"
                  value={profileData.faculty.is_active ? 'Active' : 'Inactive'}
                />
              </div>
            </div>
          )}

          {/* Statistics (for students) */}
          {userData?.role === 'student' && profileData?.stats && (
            <div className="card">
              <h2 className="text-lg font-semibold text-text-primary mb-4">Your Statistics</h2>
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
                <StatCard label="Problems Solved" value={profileData.stats.total_solved || 0} />
                <StatCard
                  label="Current Streak"
                  value={`${profileData.stats.current_streak || 0} days`}
                />
                <StatCard
                  label="Accuracy"
                  value={`${(profileData.stats.accuracy || 0).toFixed(1)}%`}
                />
                <StatCard label="College Rank" value={`#${profileData.stats.college_rank || 1}`} />
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// Helper Components
interface InfoRowProps {
  label: string;
  value: string | number;
}

function InfoRow({ label, value }: InfoRowProps) {
  return (
    <div>
      <p className="text-xs text-text-tertiary uppercase tracking-wide mb-1">{label}</p>
      <p className="text-sm font-medium text-text-primary">{value}</p>
    </div>
  );
}

interface StatCardProps {
  label: string;
  value: string | number;
}

function StatCard({ label, value }: StatCardProps) {
  return (
    <div className="text-center p-4 bg-background-tertiary rounded-lg">
      <p className="text-2xl font-bold text-text-primary">{value}</p>
      <p className="text-xs text-text-tertiary mt-1">{label}</p>
    </div>
  );
}

export default ProfilePage;
