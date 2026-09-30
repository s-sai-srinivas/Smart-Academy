import { useState, useEffect, type ChangeEvent, type FormEvent } from 'react';
import { useAuth } from '../../context/AuthContext';
import { authAPI } from '../../services/api';
import Modal from '../common/Modal';

export interface College {
  college_id: number;
  college_name: string;
  short_name?: string;
}

export interface AuthModalProps {
  onClose: () => void;
}

function AuthModal({ onClose }: AuthModalProps) {
  const { login } = useAuth();
  const [step, setStep] = useState<'college' | 'login'>('college');
  const [colleges, setColleges] = useState<College[]>([]);
  const [selectedCollege, setSelectedCollege] = useState<College | null>(null);
  const [formData, setFormData] = useState({ regdno: '', password: '' });
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [loadingColleges, setLoadingColleges] = useState(true);

  useEffect(() => {
    loadColleges();
  }, []);

  const loadColleges = async () => {
    try {
      const data = await authAPI.getColleges();
      setColleges((data as College[]) || []);
    } catch {
      setError('Failed to load colleges');
    } finally {
      setLoadingColleges(false);
    }
  };

  const handleCollegeSelect = (college: College) => {
    setSelectedCollege(college);
    setError('');
    setStep('login');
  };

  const handleBack = () => {
    setStep('college');
    setError('');
    setFormData({ regdno: '', password: '' });
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      if (!selectedCollege) return;
      await login(formData.regdno, formData.password, selectedCollege.college_id);
      onClose();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (e: ChangeEvent<HTMLInputElement>) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  return (
    <Modal onClose={onClose}>
      <div className="text-center mb-6">
        <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-accent-primary/10 text-accent-primary mb-4">
          <svg
            className="w-6 h-6"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
          >
            <path d="M16 18l6-6-6-6M8 6l-6 6 6 6" />
          </svg>
        </div>
        <h2 className="text-2xl font-bold text-text-primary mb-2">
          {step === 'college' ? 'Select Your College' : 'Sign In'}
        </h2>
        <p className="text-text-secondary text-sm">
          {step === 'college'
            ? 'Choose your college to continue'
            : `Logging in to ${selectedCollege?.college_name}`}
        </p>
      </div>

      {step === 'college' ? (
        <div className="space-y-2 max-h-80 overflow-y-auto">
          {loadingColleges ? (
            <div className="flex items-center justify-center py-8">
              <svg className="animate-spin h-6 w-6 text-accent-primary" viewBox="0 0 24 24">
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
          ) : colleges.length === 0 ? (
            <p className="text-text-muted text-sm text-center py-4">No colleges available</p>
          ) : (
            colleges.map((college) => (
              <button
                key={college.college_id}
                onClick={() => handleCollegeSelect(college)}
                className="w-full flex items-center gap-3 p-3 rounded-lg bg-background-tertiary hover:bg-background-border border border-transparent hover:border-accent-primary/30 transition-all text-left"
              >
                <div className="w-10 h-10 rounded-lg bg-accent-primary/10 flex items-center justify-center text-accent-primary font-bold text-sm shrink-0">
                  {college.short_name?.substring(0, 3) || '?'}
                </div>
                <div className="min-w-0">
                  <div className="font-medium text-text-primary text-sm truncate">
                    {college.college_name}
                  </div>
                  <div className="text-xs text-text-tertiary truncate">{college.short_name}</div>
                </div>
                <svg
                  className="w-4 h-4 text-text-muted ml-auto shrink-0"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                >
                  <path d="M9 18l6-6-6-6" />
                </svg>
              </button>
            ))
          )}
        </div>
      ) : (
        <>
          <button
            onClick={handleBack}
            className="flex items-center gap-1 text-sm text-text-secondary hover:text-text-primary mb-4 transition-colors"
          >
            <svg
              className="w-4 h-4"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
            >
              <path d="M15 18l-6-6 6-6" />
            </svg>
            Change college
          </button>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="form-group">
              <label htmlFor="regdno" className="form-label">
                Regd No / Email
              </label>
              <input
                type="text"
                id="regdno"
                name="regdno"
                className="input"
                placeholder="Enter regd no or email"
                value={formData.regdno}
                onChange={handleChange}
                required
                autoFocus
              />
            </div>
            <div className="form-group">
              <label htmlFor="password" className="form-label">
                Password
              </label>
              <input
                type="password"
                id="password"
                name="password"
                className="input"
                placeholder="••••••••"
                value={formData.password}
                onChange={handleChange}
                required
              />
            </div>
            {error && (
              <div className="p-3 rounded-lg bg-accent-danger/10 border border-accent-danger/20 text-accent-danger text-sm">
                {error}
              </div>
            )}
            <button
              type="submit"
              className="w-full btn btn-primary justify-center mt-6"
              disabled={loading}
            >
              {loading ? (
                <span className="flex items-center gap-2">
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
                  Signing in...
                </span>
              ) : (
                'Sign In'
              )}
            </button>
          </form>
        </>
      )}
      {step === 'college' && error && (
        <div className="mt-4 p-3 rounded-lg bg-accent-danger/10 border border-accent-danger/20 text-accent-danger text-sm">
          {error}
        </div>
      )}
    </Modal>
  );
}

export default AuthModal;
