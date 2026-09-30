import { useState, useEffect, useCallback, type ChangeEvent } from 'react';
import { useParams, Link } from 'react-router-dom';
import { contestsAPI } from '../../services/api';
import { showError, showSuccess } from '../../utils/showAlert';

interface PlagiarismStudent {
  name: string;
  user_regd_no: string;
  max_plagiarism_percent: number;
  problems_with_plagiarism: number;
  status: 'PLAGIARIZED' | 'SUSPICIOUS' | 'SAFE';
}

interface PlagiarismData {
  total_students: number;
  plagiarized_count: number;
  suspicious_count: number;
  safe_count: number;
  students: PlagiarismStudent[];
}

function ContestPlagiarismTab() {
  const { id } = useParams<{ id: string }>();
  const numericId = Number(id);
  const [data, setData] = useState<PlagiarismData | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [checking, setChecking] = useState<boolean>(false);
  const [searchTerm, setSearchTerm] = useState<string>('');
  const [filterStatus, setFilterStatus] = useState<string>('all');

  const loadPlagiarismData = useCallback(async () => {
    try {
      setLoading(true);
      const result = (await contestsAPI.getPlagiarism(numericId)) as PlagiarismData;
      setData(result);
    } catch (err: unknown) {
      console.error('Failed to load plagiarism data:', err);
      const message = err instanceof Error ? err.message : 'Failed to load plagiarism data';
      showError(message);
    } finally {
      setLoading(false);
    }
  }, [numericId]);

  useEffect(() => {
    loadPlagiarismData();
  }, [id, loadPlagiarismData]);

  const runPlagiarismCheck = async () => {
    setChecking(true);
    try {
      const result = (await contestsAPI.checkPlagiarism(numericId)) as PlagiarismData;
      setData(result);
      showSuccess(
        `Plagiarism check complete: ${result.plagiarized_count || 0} plagiarized, ${result.suspicious_count || 0} suspicious, ${result.safe_count || 0} safe`
      );
    } catch (err: unknown) {
      console.error('Failed to run plagiarism check:', err);
      const message = err instanceof Error ? err.message : 'Failed to run plagiarism check';
      showError(message);
    } finally {
      setChecking(false);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'PLAGIARIZED':
        return 'text-accent-danger bg-accent-danger/10';
      case 'SUSPICIOUS':
        return 'text-accent-warning bg-accent-warning/10';
      default:
        return 'text-accent-success bg-accent-success/10';
    }
  };

  const getPlagiarismBarColor = (percent: number) => {
    if (percent > 60) return 'bg-accent-danger';
    if (percent >= 30) return 'bg-accent-warning';
    return 'bg-accent-success';
  };

  const filteredStudents = (data?.students || []).filter((student: PlagiarismStudent) => {
    const matchesSearch =
      student.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      student.user_regd_no.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesFilter = filterStatus === 'all' || student.status === filterStatus;
    return matchesSearch && matchesFilter;
  });

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <p className="text-text-muted">Loading plagiarism data...</p>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto">
      {/* Header */}
      <div className="flex items-center gap-3 mb-6">
        <Link to=".." relative="path" className="text-text-muted hover:text-text-primary">
          ← Back to Contests
        </Link>
      </div>

      <div className="card mb-6">
        <div className="flex items-start justify-between mb-4">
          <div>
            <h1 className="text-3xl font-bold text-text-primary mb-2">Plagiarism Analysis</h1>
            <p className="text-text-secondary">Contest ID: {id}</p>
          </div>
          <button onClick={runPlagiarismCheck} disabled={checking} className="btn btn-primary">
            {checking ? 'Checking...' : 'Run Plagiarism Check'}
          </button>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-4 gap-4">
          <div className="bg-bg-secondary p-4 rounded-lg text-center">
            <div className="text-3xl font-bold text-text-primary">{data?.total_students || 0}</div>
            <div className="text-sm text-text-muted">Total Students</div>
          </div>
          <div className="bg-accent-danger/10 p-4 rounded-lg text-center">
            <div className="text-3xl font-bold text-accent-danger">
              {data?.plagiarized_count || 0}
            </div>
            <div className="text-sm text-accent-danger">Plagiarized</div>
          </div>
          <div className="bg-accent-warning/10 p-4 rounded-lg text-center">
            <div className="text-3xl font-bold text-accent-warning">
              {data?.suspicious_count || 0}
            </div>
            <div className="text-sm text-accent-warning">Suspicious</div>
          </div>
          <div className="bg-accent-success/10 p-4 rounded-lg text-center">
            <div className="text-3xl font-bold text-accent-success">{data?.safe_count || 0}</div>
            <div className="text-sm text-accent-success">Safe</div>
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4 mb-4">
        <input
          type="text"
          placeholder="Search by name or registration number..."
          value={searchTerm}
          onChange={(e: ChangeEvent<HTMLInputElement>) => setSearchTerm(e.target.value)}
          className="input flex-1"
        />
        <select
          value={filterStatus}
          onChange={(e: ChangeEvent<HTMLSelectElement>) => setFilterStatus(e.target.value)}
          className="select"
        >
          <option value="all">All Status</option>
          <option value="PLAGIARIZED">Plagiarized</option>
          <option value="SUSPICIOUS">Suspicious</option>
          <option value="SAFE">Safe</option>
        </select>
      </div>

      {/* Student List */}
      <div className="card">
        <h3 className="text-lg font-semibold text-text-primary mb-4">Student Plagiarism Results</h3>

        {filteredStudents.length === 0 ? (
          <div className="text-center py-8">
            <p className="text-text-muted">No students found matching the criteria.</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border-light">
                  <th className="text-left p-3 text-text-secondary font-medium">Registration No</th>
                  <th className="text-left p-3 text-text-secondary font-medium">Name</th>
                  <th className="text-left p-3 text-text-secondary font-medium">Max Similarity</th>
                  <th className="text-left p-3 text-text-secondary font-medium">
                    Problems Flagged
                  </th>
                  <th className="text-left p-3 text-text-secondary font-medium">Status</th>
                </tr>
              </thead>
              <tbody>
                {filteredStudents.map((student, index) => (
                  <tr key={index} className="border-b border-border-light hover:bg-bg-secondary">
                    <td className="p-3 text-text-primary font-mono text-sm">
                      {student.user_regd_no}
                    </td>
                    <td className="p-3 text-text-primary">{student.name}</td>
                    <td className="p-3">
                      <div className="flex items-center gap-2">
                        <div className="w-24 h-2 bg-bg-tertiary rounded-full overflow-hidden">
                          <div
                            className={`h-full ${getPlagiarismBarColor(student.max_plagiarism_percent)}`}
                            style={{ width: `${Math.min(student.max_plagiarism_percent, 100)}%` }}
                          />
                        </div>
                        <span
                          className={`text-sm font-medium ${
                            student.max_plagiarism_percent > 60
                              ? 'text-accent-danger'
                              : student.max_plagiarism_percent >= 30
                                ? 'text-accent-warning'
                                : 'text-text-primary'
                          }`}
                        >
                          {student.max_plagiarism_percent.toFixed(1)}%
                        </span>
                      </div>
                    </td>
                    <td className="p-3 text-text-primary">{student.problems_with_plagiarism}</td>
                    <td className="p-3">
                      <span
                        className={`px-2 py-1 rounded text-sm font-medium ${getStatusColor(student.status)}`}
                      >
                        {student.status}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

export default ContestPlagiarismTab;
