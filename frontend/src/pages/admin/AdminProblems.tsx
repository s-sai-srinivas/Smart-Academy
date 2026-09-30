import { useState, FormEvent, ChangeEvent } from 'react';
import api from '../../services/api';
import { showSuccess, showError } from '../../utils/showAlert';
import { getErrorMessage } from '../../utils/error';
import './AdminDashboard.css';

interface TestCase {
  input: string;
  expected_output: string;
  is_sample: boolean;
  points: number;
}

interface FormData {
  title: string;
  description: string;
  difficulty: string;
  time_limit: number;
  memory_limit: number;
  tags: string;
  test_cases: TestCase[];
}

const AdminProblems = () => {
  const [showModal, setShowModal] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(false);
  const [formData, setFormData] = useState<FormData>({
    title: '',
    description: '',
    difficulty: 'easy',
    time_limit: 2000,
    memory_limit: 256000,
    tags: '',
    test_cases: [{ input: '', expected_output: '', is_sample: true, points: 10 }],
  });

  const difficulties: string[] = ['easy', 'medium', 'hard'];

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setLoading(true);

    try {
      await api.post('/admin/problems', formData);
      setShowModal(false);
      setFormData({
        title: '',
        description: '',
        difficulty: 'easy',
        time_limit: 2000,
        memory_limit: 256000,
        tags: '',
        test_cases: [{ input: '', expected_output: '', is_sample: true, points: 10 }],
      });
      showSuccess('Problem created successfully!');
    } catch (error: unknown) {
      const message = getErrorMessage(error, 'Failed to create problem');
      console.error('Error creating problem:', error);
      showError(`Failed to create problem: ${message}`);
    } finally {
      setLoading(false);
    }
  };

  const addTestCase = () => {
    setFormData({
      ...formData,
      test_cases: [
        ...formData.test_cases,
        { input: '', expected_output: '', is_sample: false, points: 10 },
      ],
    });
  };

  const removeTestCase = (index: number) => {
    if (formData.test_cases.length > 1) {
      const newTestCases = formData.test_cases.filter((_, i) => i !== index);
      setFormData({ ...formData, test_cases: newTestCases });
    }
  };

  const updateTestCase = (
    index: number,
    field: keyof TestCase,
    value: string | boolean | number
  ) => {
    const newTestCases = [...formData.test_cases];
    newTestCases[index] = { ...newTestCases[index], [field]: value } as TestCase;
    setFormData({ ...formData, test_cases: newTestCases });
  };

  return (
    <div className="admin-problems">
      <div className="admin-header">
        <h1>Problems</h1>
        <button className="btn-admin btn-admin-primary" onClick={() => setShowModal(true)}>
          Create Problem
        </button>
      </div>

      <div className="admin-card">
        <h2>Create New Coding Problem</h2>
        <button className="btn-admin btn-admin-primary" onClick={() => setShowModal(true)}>
          Create New Problem
        </button>
      </div>

      {showModal && (
        <div
          className="modal-overlay"
          onClick={(e) => e.target === e.currentTarget && setShowModal(false)}
        >
          <div className="modal-content" style={{ maxWidth: '800px' }}>
            <div className="modal-header">
              <h2>Create New Problem</h2>
              <button className="modal-close" onClick={() => setShowModal(false)}>
                Close
              </button>
            </div>
            <form onSubmit={handleSubmit} className="admin-form">
              <div className="form-group">
                <label>Problem Title *</label>
                <input
                  type="text"
                  value={formData.title}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    setFormData({ ...formData, title: e.target.value })
                  }
                  placeholder="e.g., Two Sum"
                  required
                />
              </div>

              <div className="form-row">
                <div className="form-group">
                  <label>Difficulty *</label>
                  <select
                    value={formData.difficulty}
                    onChange={(e: ChangeEvent<HTMLSelectElement>) =>
                      setFormData({ ...formData, difficulty: e.target.value })
                    }
                    required
                  >
                    {difficulties.map((d) => (
                      <option key={d} value={d}>
                        {d.charAt(0).toUpperCase() + d.slice(1)}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="form-group">
                  <label>Time Limit (ms) *</label>
                  <input
                    type="number"
                    value={formData.time_limit}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      setFormData({ ...formData, time_limit: parseInt(e.target.value) })
                    }
                    min="100"
                    max="10000"
                    required
                  />
                </div>
                <div className="form-group">
                  <label>Memory Limit (KB) *</label>
                  <input
                    type="number"
                    value={formData.memory_limit}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      setFormData({ ...formData, memory_limit: parseInt(e.target.value) })
                    }
                    min="64"
                    max="1048576"
                    required
                  />
                </div>
              </div>

              <div className="form-group">
                <label>Description *</label>
                <textarea
                  value={formData.description}
                  onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                    setFormData({ ...formData, description: e.target.value })
                  }
                  placeholder="Describe the problem statement..."
                  rows={6}
                  required
                />
              </div>

              <div className="form-group">
                <label>Tags</label>
                <input
                  type="text"
                  value={formData.tags}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    setFormData({ ...formData, tags: e.target.value })
                  }
                  placeholder="e.g., arrays, two-pointers, hashing"
                />
              </div>

              <div className="test-cases-section">
                <div className="test-cases-header">
                  <h3>Test Cases</h3>
                  <button
                    type="button"
                    className="btn-admin btn-admin-secondary"
                    onClick={addTestCase}
                  >
                    Add Test Case
                  </button>
                </div>

                {formData.test_cases.map((tc, index) => (
                  <div key={index} className="test-case-item">
                    <div className="test-case-header">
                      <span className={`test-case-type ${tc.is_sample ? 'sample' : 'hidden'}`}>
                        {tc.is_sample ? 'Sample Test Case' : 'Hidden Test Case'}
                      </span>
                      {formData.test_cases.length > 1 && (
                        <button
                          type="button"
                          className="btn-action btn-delete"
                          onClick={() => removeTestCase(index)}
                        >
                          Remove
                        </button>
                      )}
                    </div>

                    <div className="form-row">
                      <div className="form-group" style={{ flex: 1 }}>
                        <label>
                          <input
                            type="checkbox"
                            checked={tc.is_sample}
                            onChange={(e: ChangeEvent<HTMLInputElement>) =>
                              updateTestCase(index, 'is_sample', e.target.checked)
                            }
                            style={{ marginRight: '8px' }}
                          />
                          Sample (visible to students)
                        </label>
                      </div>
                      <div className="form-group" style={{ width: '120px' }}>
                        <label>Points</label>
                        <input
                          type="number"
                          value={tc.points}
                          onChange={(e: ChangeEvent<HTMLInputElement>) =>
                            updateTestCase(index, 'points', parseInt(e.target.value) || 0)
                          }
                          min="0"
                        />
                      </div>
                    </div>

                    <div className="form-group">
                      <label>Input *</label>
                      <textarea
                        value={tc.input}
                        onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                          updateTestCase(index, 'input', e.target.value)
                        }
                        placeholder="Enter input for this test case..."
                        rows={3}
                        required
                      />
                    </div>

                    <div className="form-group">
                      <label>Expected Output *</label>
                      <textarea
                        value={tc.expected_output}
                        onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                          updateTestCase(index, 'expected_output', e.target.value)
                        }
                        placeholder="Enter expected output..."
                        rows={2}
                        required
                      />
                    </div>
                  </div>
                ))}
              </div>

              <div className="form-actions">
                <button type="submit" className="btn-admin btn-admin-primary" disabled={loading}>
                  {loading ? 'Creating...' : 'Create Problem'}
                </button>
                <button
                  type="button"
                  className="btn-admin btn-admin-secondary"
                  onClick={() => setShowModal(false)}
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default AdminProblems;
