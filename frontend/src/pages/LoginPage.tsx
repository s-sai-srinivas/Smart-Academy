import { useState, useEffect, useRef, type ChangeEvent, type FormEvent } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { authAPI } from '../services/api';
import { showError } from '../utils/showAlert';
import { Code2, ChevronLeft, ArrowRight, Search, ChevronDown, AlertCircle } from 'lucide-react';

interface College {
  college_id: number;
  college_name: string;
  short_name?: string;
}

export default function LoginPage() {
  const {
    user: _user,
    login,
    isAdmin,
    isSuperAdmin,
    isHOD,
    isPrincipal,
    isFaculty,
    isAuthenticated,
  } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const [step, setStep] = useState<'college' | 'login' | 'admin-login'>('college');
  const [colleges, setColleges] = useState<College[]>([]);
  const [selectedCollege, setSelectedCollege] = useState<College | null>(null);
  const [formData, setFormData] = useState<{ regdno: string; password: string }>({
    regdno: '',
    password: '',
  });
  const [loading, setLoading] = useState<boolean>(false);
  const [loadingColleges, setLoadingColleges] = useState<boolean>(true);
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [dropdownOpen, setDropdownOpen] = useState<boolean>(false);
  const dropdownRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);

  // Get message from URL params
  const redirectMessage = searchParams.get('message');

  // Redirect already-authenticated users
  useEffect(() => {
    if (isAuthenticated) {
      if (isSuperAdmin()) navigate('/super-admin', { replace: true });
      else if (isAdmin()) navigate('/admin', { replace: true });
      else if (isHOD()) navigate('/hod', { replace: true });
      else if (isPrincipal()) navigate('/principal', { replace: true });
      else if (isFaculty()) navigate('/faculty', { replace: true });
      else navigate('/dashboard', { replace: true });
    }
  }, [isAuthenticated, isAdmin, isFaculty, isHOD, isPrincipal, isSuperAdmin, navigate]);

  useEffect(() => {
    loadColleges();
  }, []);

  const loadColleges = async () => {
    try {
      const data = (await authAPI.getColleges()) as College[];
      setColleges(data || []);
    } catch {
      showError('Failed to load colleges');
    } finally {
      setLoadingColleges(false);
    }
  };

  // Close dropdown on outside click
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setDropdownOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const filteredColleges = colleges.filter((c) => {
    if (!searchQuery.trim()) return true;
    const q = searchQuery.toLowerCase();
    return c.college_name?.toLowerCase().includes(q) || c.short_name?.toLowerCase().includes(q);
  });

  const handleCollegeSelect = (college: College) => {
    setSelectedCollege(college);
    setSearchQuery('');
    setDropdownOpen(false);
    setStep('login');
  };

  const handleBack = () => {
    setStep('college');
    setFormData({ regdno: '', password: '' });
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      const collegeId = step === 'admin-login' ? null : selectedCollege?.college_id;
      if (collegeId === undefined && step !== 'admin-login') {
        showError('Please select a college');
        return;
      }
      await login(formData.regdno, formData.password, collegeId ?? 0);
      // useEffect redirect will handle navigation
    } catch (err) {
      showError((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (e: ChangeEvent<HTMLInputElement>) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  return (
    <div className="min-h-screen bg-background-primary flex flex-col">
      {/* ── Minimal top bar ── */}
      <header className="border-b border-background-border bg-background-secondary/60 backdrop-blur-md">
        <div className="max-w-7xl mx-auto flex items-center justify-between px-6 py-3.5">
          <Link to="/" className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-accent-secondary/10 flex items-center justify-center">
              <Code2 className="w-5 h-5 text-accent-secondary" />
            </div>
            <span className="text-lg font-semibold text-text-primary tracking-tight">
              Smart Academy
            </span>
          </Link>
          <Link
            to="/"
            className="text-sm text-text-secondary hover:text-text-primary transition-colors flex items-center gap-1.5"
          >
            <ChevronLeft className="w-4 h-4" />
            Back to home
          </Link>
        </div>
      </header>

      {/* ── Centered card ── */}
      <main className="flex-1 flex items-center justify-center px-6 py-16">
        {/* Subtle glow */}
        <div className="absolute top-24 left-1/2 -translate-x-1/2 w-[600px] h-[350px] bg-accent-secondary/[0.04] rounded-full blur-[100px] pointer-events-none" />

        <div className="relative w-full max-w-md">
          {/* Card */}
          <div className="rounded-xl bg-background-secondary border border-background-border p-8 shadow-card">
            {/* Header */}
            <div className="text-center mb-8">
              <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-accent-secondary/10 text-accent-secondary mb-4">
                <Code2 className="w-6 h-6" />
              </div>
              <h1 className="text-2xl font-bold text-text-primary mb-2">
                {step === 'college'
                  ? 'Select Your College'
                  : step === 'admin-login'
                    ? 'Platform Admin'
                    : 'Sign In'}
              </h1>
              <p className="text-sm text-text-secondary">
                {step === 'college'
                  ? 'Choose your institution to continue'
                  : step === 'admin-login'
                    ? 'Sign in with your super admin credentials'
                    : `Signing in to ${selectedCollege?.college_name}`}
              </p>
            </div>

            {/* Redirect message banner */}
            {redirectMessage && (
              <div className="mb-6 p-3 rounded-lg bg-amber-500/10 border border-amber-500/20 flex items-center gap-2.5">
                <AlertCircle className="w-4 h-4 text-amber-500 shrink-0" />
                <p className="text-sm text-amber-600 dark:text-amber-400">{redirectMessage}</p>
              </div>
            )}

            {/* ── Step: College select ── */}
            {step === 'college' ? (
              <div ref={dropdownRef} className="relative">
                {/* Search input */}
                <div
                  className={`flex items-center gap-2.5 px-4 py-3 bg-background-tertiary border rounded-lg transition-colors cursor-text ${
                    dropdownOpen
                      ? 'border-accent-secondary'
                      : 'border-background-border hover:border-background-elevated'
                  }`}
                  onClick={() => {
                    setDropdownOpen(true);
                    inputRef.current?.focus();
                  }}
                >
                  <Search className="w-4 h-4 text-text-muted shrink-0" />
                  <input
                    ref={inputRef}
                    type="text"
                    placeholder="Search for your college..."
                    value={searchQuery}
                    onChange={(e) => {
                      setSearchQuery(e.target.value);
                      setDropdownOpen(true);
                    }}
                    onFocus={() => setDropdownOpen(true)}
                    className="flex-1 bg-transparent border-none outline-none text-sm text-text-primary placeholder-text-muted"
                  />
                  <ChevronDown
                    className={`w-4 h-4 text-text-muted shrink-0 transition-transform ${dropdownOpen ? 'rotate-180' : ''}`}
                  />
                </div>

                {/* Dropdown list */}
                {dropdownOpen && (
                  <div className="absolute z-20 left-0 right-0 mt-2 rounded-lg bg-background-secondary border border-background-border shadow-elevated overflow-hidden">
                    <div className="max-h-64 overflow-y-auto">
                      {loadingColleges ? (
                        <div className="flex items-center justify-center py-8">
                          <svg
                            className="animate-spin h-5 w-5 text-accent-secondary"
                            viewBox="0 0 24 24"
                          >
                            <circle
                              className="opacity-25"
                              cx="12"
                              cy="12"
                              r="10"
                              stroke="currentColor"
                              strokeWidth="4"
                              fill="none"
                            />
                            <path
                              className="opacity-75"
                              fill="currentColor"
                              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                            />
                          </svg>
                        </div>
                      ) : filteredColleges.length === 0 ? (
                        <div className="px-4 py-6 text-center">
                          <p className="text-sm text-text-muted">
                            No colleges match &quot;{searchQuery}&quot;
                          </p>
                        </div>
                      ) : (
                        filteredColleges.map((college) => (
                          <button
                            key={college.college_id}
                            onClick={() => handleCollegeSelect(college)}
                            className="w-full flex items-center gap-3 px-4 py-3 hover:bg-background-tertiary transition-colors text-left group"
                          >
                            <div className="w-9 h-9 rounded-lg bg-accent-secondary/10 flex items-center justify-center text-accent-secondary font-bold text-xs shrink-0">
                              {college.short_name?.substring(0, 3) || '?'}
                            </div>
                            <div className="min-w-0 flex-1">
                              <div className="font-medium text-text-primary text-sm truncate">
                                {college.college_name}
                              </div>
                              <div className="text-xs text-text-tertiary truncate">
                                {college.short_name}
                              </div>
                            </div>
                            <ArrowRight className="w-4 h-4 text-text-muted opacity-0 group-hover:opacity-100 group-hover:text-accent-secondary shrink-0 transition-all" />
                          </button>
                        ))
                      )}
                    </div>
                  </div>
                )}

                {/* Platform admin link */}
                <div className="mt-6 pt-5 border-t border-background-border text-center">
                  <button
                    onClick={() => {
                      setStep('admin-login');
                      setFormData({ regdno: '', password: '' });
                    }}
                    className="text-xs text-text-muted hover:text-text-secondary transition-colors"
                  >
                    Sign in as Platform Admin
                  </button>
                </div>
              </div>
            ) : (
              /* ── Step: Credentials (college or admin login) ── */
              <>
                <button
                  onClick={handleBack}
                  className="flex items-center gap-1.5 text-sm text-text-secondary hover:text-text-primary mb-5 transition-colors"
                >
                  <ChevronLeft className="w-4 h-4" />
                  {step === 'admin-login' ? 'Back to college select' : 'Change college'}
                </button>

                <form onSubmit={handleSubmit} className="space-y-5">
                  <div>
                    <label
                      htmlFor="regdno"
                      className="block text-sm font-medium text-text-secondary mb-1.5"
                    >
                      Regd No / Email
                    </label>
                    <input
                      type="text"
                      id="regdno"
                      name="regdno"
                      placeholder="Enter regd no or email"
                      value={formData.regdno}
                      onChange={handleChange}
                      required
                      autoFocus
                      className="w-full px-4 py-2.5 text-sm bg-background-tertiary border border-background-border rounded-lg text-text-primary placeholder-text-muted focus:border-accent-secondary focus:outline-none transition-colors"
                    />
                  </div>

                  <div>
                    <label
                      htmlFor="password"
                      className="block text-sm font-medium text-text-secondary mb-1.5"
                    >
                      Password
                    </label>
                    <input
                      type="password"
                      id="password"
                      name="password"
                      placeholder="••••••••"
                      value={formData.password}
                      onChange={handleChange}
                      required
                      className="w-full px-4 py-2.5 text-sm bg-background-tertiary border border-background-border rounded-lg text-text-primary placeholder-text-muted focus:border-accent-secondary focus:outline-none transition-colors"
                    />
                  </div>

                  <button
                    type="submit"
                    disabled={loading}
                    className="w-full py-3 text-sm font-semibold rounded-lg bg-accent-secondary hover:bg-blue-500 text-white transition-colors disabled:opacity-60"
                  >
                    {loading ? (
                      <span className="inline-flex items-center gap-2">
                        <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24">
                          <circle
                            className="opacity-25"
                            cx="12"
                            cy="12"
                            r="10"
                            stroke="currentColor"
                            strokeWidth="4"
                            fill="none"
                          />
                          <path
                            className="opacity-75"
                            fill="currentColor"
                            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                          />
                        </svg>
                        Signing in…
                      </span>
                    ) : (
                      'Sign In'
                    )}
                  </button>
                </form>
              </>
            )}
          </div>

          {/* Sub-text */}
          <p className="text-center text-xs text-text-muted mt-6">
            Don&apos;t have an account? Contact your institution&apos;s administrator.
          </p>
        </div>
      </main>
    </div>
  );
}
