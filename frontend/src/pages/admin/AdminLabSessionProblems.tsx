import { useState, useEffect, useCallback } from 'react';
import { useParams, Link } from 'react-router-dom';
import api from '../../services/api';
import { problemsAPI } from '../../services/api';
import { showError } from '../../utils/showAlert';
import { getErrorMessage } from '../../utils/error';
import CreateProblemModal from '../../components/admin/CreateProblemModal';

interface Course {
  course_code?: string;
  course_name?: string;
}

interface Session {
  title?: string;
  course?: Course;
}

interface Problem {
  id: string;
  title: string;
  description?: string;
  difficulty?: string;
  tags?: string;
  time_limit?: number;
  memory_limit?: number;
}

const AdminLabSessionProblems = () => {
  const { sessionId } = useParams<{ sessionId: string }>();
  const [loading, setLoading] = useState<boolean>(true);
  const [session, setSession] = useState<Session | null>(null);
  const [problems, setProblems] = useState<Problem[]>([]);
  const [showCreateModal, setShowCreateModal] = useState<boolean>(false);

  const loadSessionData = useCallback(async () => {
    try {
      // Load session details
      const sessionResponse = (await api.get(`/admin/lab-sessions/${sessionId}`)) as Session;
      setSession(sessionResponse);

      // Load problems for this session
      const problemsResponse = (await api.get(
        `/admin/lab-sessions/${sessionId}/problems`
      )) as Problem[];
      setProblems(problemsResponse || []);
    } catch (error) {
      console.error('Error loading session data:', error);
    } finally {
      setLoading(false);
    }
  }, [sessionId]);

  useEffect(() => {
    loadSessionData();
  }, [loadSessionData]);

  const handleDeleteProblem = async (problemId: string) => {
    if (!confirm('Are you sure you want to delete this problem?')) return;

    try {
      await problemsAPI.delete(problemId);
      loadSessionData();
    } catch (error: unknown) {
      const message = getErrorMessage(error, 'Failed to delete problem');
      console.error('Error deleting problem:', error);
      showError(`Failed to delete problem: ${message}`);
    }
  };

  const handleCreateSuccess = () => {
    loadSessionData();
  };

  const getDifficultyClass = (difficulty?: string): string => {
    switch (difficulty) {
      case 'easy':
        return 'badge-success';
      case 'medium':
        return 'badge-warning';
      case 'hard':
        return 'badge-danger';
      default:
        return 'badge-neutral';
    }
  };

  if (loading) {
    return (
      <div className="admin-labs">
        <div className="admin-header">
          <Link
            to="/admin/labs"
            className="btn-admin btn-admin-secondary"
            style={{ textDecoration: 'none' }}
          >
            Back to Lab Sessions
          </Link>
        </div>
        <div className="admin-card" style={{ textAlign: 'center', padding: '3rem' }}>
          <p>Loading...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="admin-labs">
      <div className="admin-header">
        <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
          <Link
            to="/admin/labs"
            className="btn-admin btn-admin-secondary"
            style={{ textDecoration: 'none' }}
          >
            Back
          </Link>
          <div>
            <h1 style={{ margin: 0 }}>{session?.title || 'Lab Session'}</h1>
            <p style={{ margin: 0, fontSize: '0.875rem', opacity: 0.7 }}>
              {session?.course?.course_code} - {session?.course?.course_name}
            </p>
          </div>
        </div>
        <button className="btn-admin btn-admin-primary" onClick={() => setShowCreateModal(true)}>
          Add Problem
        </button>
      </div>

      {problems.length === 0 ? (
        <div className="admin-empty">
          <div className="admin-empty-icon"></div>
          <h3>No problems yet</h3>
          <p>Add problems to this lab session to get started</p>
        </div>
      ) : (
        <div className="admin-card">
          <table className="admin-table">
            <thead>
              <tr>
                <th>Title</th>
                <th>Difficulty</th>
                <th>Tags</th>
                <th>Time Limit</th>
                <th>Memory Limit</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {problems.map((problem) => (
                <tr key={problem.id}>
                  <td>
                    <div>
                      <div style={{ fontWeight: '600' }}>{problem.title}</div>
                      {problem.description && (
                        <div style={{ fontSize: '0.8rem', opacity: 0.7, maxWidth: '300px' }}>
                          {problem.description.substring(0, 80)}...
                        </div>
                      )}
                    </div>
                  </td>
                  <td>
                    <span className={`badge ${getDifficultyClass(problem.difficulty)}`}>
                      {problem.difficulty}
                    </span>
                  </td>
                  <td>
                    {problem.tags ? (
                      <div style={{ display: 'flex', gap: '0.25rem', flexWrap: 'wrap' }}>
                        {problem.tags
                          .split(',')
                          .slice(0, 2)
                          .map((tag, i) => (
                            <span
                              key={i}
                              className="badge badge-neutral"
                              style={{ fontSize: '0.75rem' }}
                            >
                              {tag.trim()}
                            </span>
                          ))}
                        {problem.tags.split(',').length > 2 && (
                          <span className="badge badge-neutral" style={{ fontSize: '0.75rem' }}>
                            +{problem.tags.split(',').length - 2}
                          </span>
                        )}
                      </div>
                    ) : (
                      '-'
                    )}
                  </td>
                  <td>{problem.time_limit || 2000}ms</td>
                  <td>{(problem.memory_limit || 256) / 1024}MB</td>
                  <td>
                    <button
                      className="btn-admin btn-admin-danger"
                      onClick={() => handleDeleteProblem(problem.id)}
                      style={{ padding: '0.25rem 0.75rem', fontSize: '0.875rem' }}
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {showCreateModal && (
        <CreateProblemModal
          onClose={() => setShowCreateModal(false)}
          onSuccess={handleCreateSuccess}
        />
      )}
    </div>
  );
};

export default AdminLabSessionProblems;
