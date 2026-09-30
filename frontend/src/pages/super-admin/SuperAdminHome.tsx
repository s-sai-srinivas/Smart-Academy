import { useState, useEffect, useCallback, FormEvent, ChangeEvent } from 'react';
import { useAuth } from '../../context/AuthContext';
import { superAdminAPI } from '../../services/api';
import { showError, showSuccess } from '../../utils/showAlert';
import {
  Building2,
  Users,
  Shield,
  Plus,
  Power,
  Eye,
  Search,
  X,
  BookOpen,
  ChevronRight,
  Tag,
} from 'lucide-react';
import { Link } from 'react-router-dom';

interface User {
  name?: string;
}

interface College {
  college_id: string;
  college_name: string;
  short_name?: string;
  is_active: boolean;
  user_count?: number;
  address?: string;
}

interface CollegeDetail {
  college?: College;
  admin_count?: number;
  student_count?: number;
  faculty_count?: number;
}

interface FormState {
  college_id: string;
  college_name: string;
  short_name: string;
  address: string;
  admin_regdno: string;
  admin_email: string;
  admin_password: string;
  admin_name: string;
}

const SuperAdminHome = () => {
  const { user } = useAuth();
  const [colleges, setColleges] = useState<College[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [showCreate, setShowCreate] = useState<boolean>(false);
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [creating, setCreating] = useState<boolean>(false);
  const [selectedCollege, setSelectedCollege] = useState<College | null>(null);
  const [collegeDetail, setCollegeDetail] = useState<CollegeDetail | null>(null);
  const [loadingDetail, setLoadingDetail] = useState<boolean>(false);

  const [form, setForm] = useState<FormState>({
    college_id: '',
    college_name: '',
    short_name: '',
    address: '',
    admin_regdno: '',
    admin_email: '',
    admin_password: '',
    admin_name: '',
  });

  const fetchColleges = useCallback(async () => {
    try {
      setLoading(true);
      const active = statusFilter === 'all' ? undefined : statusFilter === 'active';
      const data = (await superAdminAPI.listColleges(active)) as College[];
      setColleges(data || []);
    } catch {
      showError('Failed to load colleges');
    } finally {
      setLoading(false);
    }
  }, [statusFilter]);

  useEffect(() => {
    fetchColleges();
  }, [fetchColleges]);

  const handleCreate = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setCreating(true);
    try {
      await superAdminAPI.createCollege(form);
      showSuccess('College created successfully!');
      setShowCreate(false);
      setForm({
        college_id: '',
        college_name: '',
        short_name: '',
        address: '',
        admin_regdno: '',
        admin_email: '',
        admin_password: '',
        admin_name: '',
      });
      fetchColleges();
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to create college');
    } finally {
      setCreating(false);
    }
  };

  const handleToggleStatus = async (college: College) => {
    try {
      await superAdminAPI.updateCollegeStatus(college.college_id, !college.is_active);
      showSuccess(`College ${college.is_active ? 'suspended' : 'activated'} successfully`);
      fetchColleges();
      if (selectedCollege?.college_id === college.college_id) {
        viewCollegeDetail(college.college_id);
      }
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to update college status');
    }
  };

  const viewCollegeDetail = async (id: string) => {
    setLoadingDetail(true);
    try {
      const data = (await superAdminAPI.getCollege(id)) as CollegeDetail;
      setCollegeDetail(data);
      setSelectedCollege(data.college || null);
    } catch {
      showError('Failed to load college details');
    } finally {
      setLoadingDetail(false);
    }
  };

  const filteredColleges = colleges.filter((c) => {
    if (!searchQuery.trim()) return true;
    const q = searchQuery.toLowerCase();
    return c.college_name?.toLowerCase().includes(q) || c.short_name?.toLowerCase().includes(q);
  });

  const totalColleges = colleges.length;
  const activeColleges = colleges.filter((c) => c.is_active).length;
  const totalUsers = colleges.reduce((sum, c) => sum + (c.user_count || 0), 0);

  return (
    <div className="space-y-6 max-w-7xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-text-primary mb-1">Platform Administration</h1>
          <p className="text-text-secondary text-sm">
            Welcome, {(user as User)?.name || 'Super Admin'} — manage colleges and platform settings
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="btn btn-primary flex items-center gap-2"
        >
          <Plus className="w-4 h-4" />
          Add College
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-5">
        <div className="card-elevated flex items-center gap-4 border-l-4 border-accent-secondary">
          <Building2 className="w-8 h-8 text-accent-secondary" />
          <div>
            <div className="text-3xl font-bold text-text-primary">
              {loading ? '...' : totalColleges}
            </div>
            <div className="text-sm text-text-secondary">Total Colleges</div>
          </div>
        </div>
        <div className="card-elevated flex items-center gap-4 border-l-4 border-accent-success">
          <Shield className="w-8 h-8 text-accent-success" />
          <div>
            <div className="text-3xl font-bold text-text-primary">
              {loading ? '...' : activeColleges}
            </div>
            <div className="text-sm text-text-secondary">Active Colleges</div>
          </div>
        </div>
        <div className="card-elevated flex items-center gap-4 border-l-4 border-accent-primary">
          <Users className="w-8 h-8 text-accent-primary" />
          <div>
            <div className="text-3xl font-bold text-text-primary">
              {loading ? '...' : totalUsers}
            </div>
            <div className="text-sm text-text-secondary">Total Platform Users</div>
          </div>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <Link
          to="/super-admin/problems"
          className="card-elevated flex items-center gap-4 hover:border-accent-secondary/30 transition-colors cursor-pointer group"
        >
          <div className="w-12 h-12 rounded-lg bg-accent-secondary/20 flex items-center justify-center">
            <BookOpen className="w-6 h-6 text-accent-secondary" />
          </div>
          <div className="flex-1">
            <div className="font-medium text-text-primary group-hover:text-accent-secondary transition-colors">
              Practice Problems
            </div>
            <div className="text-sm text-text-secondary">Manage global problems</div>
          </div>
          <ChevronRight className="w-5 h-5 text-text-muted group-hover:text-accent-secondary transition-colors" />
        </Link>
        <Link
          to="/super-admin/topics"
          className="card-elevated flex items-center gap-4 hover:border-accent-secondary/30 transition-colors cursor-pointer group"
        >
          <div className="w-12 h-12 rounded-lg bg-accent-primary/20 flex items-center justify-center">
            <Tag className="w-6 h-6 text-accent-primary" />
          </div>
          <div className="flex-1">
            <div className="font-medium text-text-primary group-hover:text-accent-secondary transition-colors">
              Subjects & Topics
            </div>
            <div className="text-sm text-text-secondary">Manage topic hierarchy</div>
          </div>
          <ChevronRight className="w-5 h-5 text-text-muted group-hover:text-accent-secondary transition-colors" />
        </Link>
        <button
          onClick={() => setShowCreate(true)}
          className="card-elevated flex items-center gap-4 hover:border-accent-secondary/30 transition-colors cursor-pointer group text-left"
        >
          <div className="w-12 h-12 rounded-lg bg-green-500/20 flex items-center justify-center">
            <Building2 className="w-6 h-6 text-green-400" />
          </div>
          <div className="flex-1">
            <div className="font-medium text-text-primary group-hover:text-accent-secondary transition-colors">
              Add College
            </div>
            <div className="text-sm text-text-secondary">Create new institution</div>
          </div>
          <ChevronRight className="w-5 h-5 text-text-muted group-hover:text-accent-secondary transition-colors" />
        </button>
      </div>

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" />
          <input
            type="text"
            placeholder="Search colleges..."
            value={searchQuery}
            onChange={(e: ChangeEvent<HTMLInputElement>) => setSearchQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-2.5 bg-background-secondary border border-background-border rounded-lg text-text-primary placeholder-text-tertiary text-sm focus:outline-none focus:border-accent-secondary"
          />
        </div>
        <div className="flex gap-2">
          {['all', 'active', 'inactive'].map((f) => (
            <button
              key={f}
              onClick={() => setStatusFilter(f)}
              className={`px-4 py-2 text-sm font-medium rounded-lg transition-colors capitalize ${
                statusFilter === f
                  ? 'bg-accent-secondary text-white'
                  : 'bg-background-secondary text-text-secondary hover:text-text-primary border border-background-border'
              }`}
            >
              {f}
            </button>
          ))}
        </div>
      </div>

      {/* College List + Detail Side Panel */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* College List */}
        <div className={`space-y-3 ${selectedCollege ? 'lg:col-span-2' : 'lg:col-span-3'}`}>
          {loading ? (
            <div className="text-center py-12 text-text-secondary">Loading colleges...</div>
          ) : filteredColleges.length === 0 ? (
            <div className="text-center py-12">
              <Building2 className="w-12 h-12 mx-auto text-text-muted mb-3" />
              <p className="text-text-secondary">No colleges found</p>
              <button
                onClick={() => setShowCreate(true)}
                className="mt-3 text-accent-secondary hover:underline text-sm"
              >
                Create your first college
              </button>
            </div>
          ) : (
            filteredColleges.map((college) => (
              <div
                key={college.college_id}
                className={`card-elevated flex items-center justify-between cursor-pointer transition-colors hover:border-accent-secondary/30 ${
                  selectedCollege?.college_id === college.college_id
                    ? 'border-accent-secondary/50 bg-background-tertiary'
                    : ''
                }`}
                onClick={() => viewCollegeDetail(college.college_id)}
              >
                <div className="flex items-center gap-4 min-w-0">
                  <div
                    className={`w-10 h-10 rounded-lg flex items-center justify-center text-sm font-bold ${
                      college.is_active
                        ? 'bg-accent-secondary/20 text-accent-secondary'
                        : 'bg-red-500/20 text-red-400'
                    }`}
                  >
                    {college.short_name?.substring(0, 2).toUpperCase()}
                  </div>
                  <div className="min-w-0">
                    <div className="text-text-primary font-medium truncate">
                      {college.college_name}
                    </div>
                    <div className="text-text-tertiary text-xs flex items-center gap-3">
                      <span>{college.short_name}</span>
                      <span>{college.user_count || 0} users</span>
                      <span
                        className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs ${
                          college.is_active
                            ? 'bg-green-500/10 text-green-400'
                            : 'bg-red-500/10 text-red-400'
                        }`}
                      >
                        <span
                          className={`w-1.5 h-1.5 rounded-full ${college.is_active ? 'bg-green-400' : 'bg-red-400'}`}
                        />
                        {college.is_active ? 'Active' : 'Suspended'}
                      </span>
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      handleToggleStatus(college);
                    }}
                    className={`p-2 rounded-lg transition-colors ${
                      college.is_active
                        ? 'text-red-400 hover:bg-red-500/10'
                        : 'text-green-400 hover:bg-green-500/10'
                    }`}
                    title={college.is_active ? 'Suspend' : 'Activate'}
                  >
                    <Power className="w-4 h-4" />
                  </button>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      viewCollegeDetail(college.college_id);
                    }}
                    className="p-2 rounded-lg text-text-secondary hover:bg-background-tertiary transition-colors"
                    title="View details"
                  >
                    <Eye className="w-4 h-4" />
                  </button>
                </div>
              </div>
            ))
          )}
        </div>

        {/* Detail Panel */}
        {selectedCollege && (
          <div className="lg:col-span-1">
            <div className="card-elevated sticky top-20">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-text-primary">College Details</h3>
                <button
                  onClick={() => {
                    setSelectedCollege(null);
                    setCollegeDetail(null);
                  }}
                  className="text-text-muted hover:text-text-primary"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
              {loadingDetail ? (
                <div className="text-text-secondary text-sm py-4">Loading...</div>
              ) : collegeDetail ? (
                <div className="space-y-4">
                  <div>
                    <div className="text-text-tertiary text-xs uppercase mb-1">Name</div>
                    <div className="text-text-primary font-medium">
                      {collegeDetail.college?.college_name}
                    </div>
                  </div>
                  <div>
                    <div className="text-text-tertiary text-xs uppercase mb-1">Short Name</div>
                    <div className="text-text-primary">{collegeDetail.college?.short_name}</div>
                  </div>
                  <div>
                    <div className="text-text-tertiary text-xs uppercase mb-1">Status</div>
                    <span
                      className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${
                        collegeDetail.college?.is_active
                          ? 'bg-green-500/10 text-green-400'
                          : 'bg-red-500/10 text-red-400'
                      }`}
                    >
                      <span
                        className={`w-1.5 h-1.5 rounded-full ${collegeDetail.college?.is_active ? 'bg-green-400' : 'bg-red-400'}`}
                      />
                      {collegeDetail.college?.is_active ? 'Active' : 'Suspended'}
                    </span>
                  </div>
                  {collegeDetail.college?.address && (
                    <div>
                      <div className="text-text-tertiary text-xs uppercase mb-1">Address</div>
                      <div className="text-text-primary text-sm">
                        {collegeDetail.college.address}
                      </div>
                    </div>
                  )}
                  <hr className="border-background-border" />
                  <div className="grid grid-cols-3 gap-3 text-center">
                    <div>
                      <div className="text-xl font-bold text-accent-secondary">
                        {collegeDetail.admin_count ?? 0}
                      </div>
                      <div className="text-text-tertiary text-xs">Admins</div>
                    </div>
                    <div>
                      <div className="text-xl font-bold text-accent-primary">
                        {collegeDetail.student_count ?? 0}
                      </div>
                      <div className="text-text-tertiary text-xs">Students</div>
                    </div>
                    <div>
                      <div className="text-xl font-bold text-accent-success">
                        {collegeDetail.faculty_count ?? 0}
                      </div>
                      <div className="text-text-tertiary text-xs">Faculty</div>
                    </div>
                  </div>
                  <button
                    onClick={() => {
                      if (collegeDetail.college) {
                        handleToggleStatus(collegeDetail.college);
                      }
                    }}
                    className={`w-full mt-2 py-2 rounded-lg text-sm font-medium transition-colors ${
                      collegeDetail.college?.is_active
                        ? 'bg-red-500/10 text-red-400 hover:bg-red-500/20 border border-red-500/30'
                        : 'bg-green-500/10 text-green-400 hover:bg-green-500/20 border border-green-500/30'
                    }`}
                  >
                    {collegeDetail.college?.is_active ? 'Suspend College' : 'Activate College'}
                  </button>
                </div>
              ) : null}
            </div>
          </div>
        )}
      </div>

      {/* Create College Modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="w-full max-w-lg bg-background-secondary border border-background-border rounded-xl p-6">
            <div className="flex items-center justify-between mb-5">
              <h2 className="text-xl font-bold text-text-primary">Add New College</h2>
              <button
                onClick={() => setShowCreate(false)}
                className="text-text-muted hover:text-text-primary"
              >
                <X className="w-5 h-5" />
              </button>
            </div>
            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="block text-xs text-text-secondary mb-1">College ID *</label>
                <input
                  type="text"
                  required
                  value={form.college_id}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    setForm({
                      ...form,
                      college_id: e.target.value.toUpperCase().replace(/\s/g, '_'),
                    })
                  }
                  placeholder="e.g. VIT-AP, NITK, IIIT-H"
                  className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary font-mono"
                />
                <span className="text-xs text-text-muted mt-0.5 block">
                  Unique identifier for this college (uppercase, no spaces)
                </span>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs text-text-secondary mb-1">College Name *</label>
                  <input
                    type="text"
                    required
                    value={form.college_name}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      setForm({ ...form, college_name: e.target.value })
                    }
                    className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary"
                  />
                </div>
                <div>
                  <label className="block text-xs text-text-secondary mb-1">Short Name *</label>
                  <input
                    type="text"
                    required
                    value={form.short_name}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      setForm({ ...form, short_name: e.target.value })
                    }
                    placeholder="e.g. MIT"
                    className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary"
                  />
                </div>
              </div>
              <div>
                <label className="block text-xs text-text-secondary mb-1">Address</label>
                <input
                  type="text"
                  value={form.address}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    setForm({ ...form, address: e.target.value })
                  }
                  className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary"
                />
              </div>
              <hr className="border-background-border" />
              <p className="text-xs text-text-tertiary">
                College Admin Account — this user will manage the college
              </p>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs text-text-secondary mb-1">Admin Name *</label>
                  <input
                    type="text"
                    required
                    value={form.admin_name}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      setForm({ ...form, admin_name: e.target.value })
                    }
                    className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary"
                  />
                </div>
                <div>
                  <label className="block text-xs text-text-secondary mb-1">Admin Reg. No *</label>
                  <input
                    type="text"
                    required
                    value={form.admin_regdno}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      setForm({ ...form, admin_regdno: e.target.value })
                    }
                    className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary"
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs text-text-secondary mb-1">Admin Email *</label>
                  <input
                    type="email"
                    required
                    value={form.admin_email}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      setForm({ ...form, admin_email: e.target.value })
                    }
                    className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary"
                  />
                </div>
                <div>
                  <label className="block text-xs text-text-secondary mb-1">Admin Password *</label>
                  <input
                    type="password"
                    required
                    minLength={6}
                    value={form.admin_password}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      setForm({ ...form, admin_password: e.target.value })
                    }
                    className="w-full px-3 py-2 bg-background-tertiary border border-background-border rounded-lg text-text-primary text-sm focus:outline-none focus:border-accent-secondary"
                  />
                </div>
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowCreate(false)}
                  className="btn btn-secondary"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={creating}
                  className="btn btn-primary flex items-center gap-2"
                >
                  {creating ? 'Creating...' : 'Create College'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default SuperAdminHome;
