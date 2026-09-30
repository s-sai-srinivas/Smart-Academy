import { useState, useEffect, type ChangeEvent, type FormEvent } from 'react';
import Modal from '../common/Modal';
import { problemsAPI, contestsAPI, superAdminPracticeAPI, referenceAPI } from '../../services/api';

export interface TestCaseForm {
  input: string;
  expected_output: string;
  is_sample: boolean;
  points: number;
}

export interface CreateProblemModalProps {
  onClose: () => void;
  onSuccess: () => void;
  labSessionId?: number;
  contestId?: number;
  contestPoints?: number;
  sectionId?: number;
  isGlobal?: boolean;
}

interface Subject {
  id: number;
  name: string;
}

interface Topic {
  id: number;
  subject_id: number;
  name: string;
}

function CreateProblemModal({
  onClose,
  onSuccess,
  labSessionId,
  contestId,
  contestPoints: defaultPoints = 100,
  sectionId,
  isGlobal = false,
}: CreateProblemModalProps) {
  const [step, setStep] = useState(1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [contestPoints, setContestPoints] = useState(defaultPoints);

  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [topics, setTopics] = useState<Topic[]>([]);
  const [selectedSubject, setSelectedSubject] = useState<number>(0);

  const [problemData, setProblemData] = useState({
    title: '',
    description: '',
    difficulty: 'easy' as 'easy' | 'medium' | 'hard',
    tags: '',
    time_limit: 2000,
    memory_limit: 256000,
  });

  const [testCases, setTestCases] = useState<TestCaseForm[]>([
    { input: '', expected_output: '', is_sample: true, points: 10 },
  ]);

  // Fetch subjects/topics only when creating global problems
  useEffect(() => {
    if (!isGlobal) return;
    referenceAPI
      .getSubjects(false)
      .then((data) => {
        setSubjects(Array.isArray(data) ? data : []);
      })
      .catch(() => {
        /* ignore */
      });
  }, [isGlobal]);

  useEffect(() => {
    if (!isGlobal) return;
    if (selectedSubject) {
      referenceAPI
        .getTopics(selectedSubject)
        .then((data) => {
          setTopics(Array.isArray(data) ? data : []);
        })
        .catch(() => {
          /* ignore */
        });
    } else {
      setTopics([]);
    }
  }, [selectedSubject, isGlobal]);

  const handleProblemChange = (
    e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>
  ) => {
    setProblemData({ ...problemData, [e.target.name]: e.target.value });
  };

  const handleNext = (e: FormEvent) => {
    e.preventDefault();
    setStep(2);
  };

  const addTestCase = () => {
    setTestCases([...testCases, { input: '', expected_output: '', is_sample: false, points: 10 }]);
  };

  const removeTestCase = (index: number) => {
    setTestCases(testCases.filter((_, i) => i !== index));
  };

  const updateTestCase = <K extends keyof TestCaseForm>(
    index: number,
    field: K,
    value: TestCaseForm[K]
  ) => {
    const updated = [...testCases];
    updated[index] = { ...updated[index], [field]: value };
    setTestCases(updated);
  };

  const handleSubmit = async () => {
    setError('');
    setLoading(true);
    try {
      const createData = {
        ...problemData,
        time_limit: parseInt(String(problemData.time_limit)),
        memory_limit: parseInt(String(problemData.memory_limit)),
        test_cases: testCases.filter((tc) => tc.input && tc.expected_output),
      };

      if (isGlobal) {
        const created = await superAdminPracticeAPI.create({
          title: createData.title,
          description: createData.description,
          difficulty: createData.difficulty,
          tags: createData.tags,
          time_limit: createData.time_limit,
          memory_limit: createData.memory_limit,
          subject_id: selectedSubject || undefined,
        });
        const res = created as { problem?: { id?: number }; id?: number };
        const problemId = res?.problem?.id || res?.id;
        if (problemId && createData.test_cases.length > 0) {
          for (const tc of createData.test_cases) {
            await superAdminPracticeAPI.createTestCase(problemId, tc);
          }
        }
      } else {
        const payload: Record<string, unknown> = { ...createData };
        // For lab problems (non-global) we don't send subject or tags
        delete payload.tags;
        if (labSessionId) payload.lab_session_id = labSessionId;
        const created = await problemsAPI.create(payload);
        if (contestId) {
          const res = created as { id?: number; problem_id?: number };
          const problemId = res?.id || res?.problem_id;
          if (problemId) {
            const addPayload: Record<string, unknown> = {
              problem_id: problemId,
              points: contestPoints,
            };
            if (sectionId) addPayload.section_id = sectionId;
            await contestsAPI.addProblem(contestId, addPayload);
          }
        }
      }
      onSuccess();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal onClose={onClose}>
      {step === 1 ? (
        <>
          <h2 style={{ fontSize: '1.25rem', fontWeight: '600', marginBottom: '1.25rem' }}>
            {isGlobal ? 'Create Global Problem' : 'Create Problem'}
          </h2>
          <form
            onSubmit={handleNext}
            style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}
          >
            <div className="form-group">
              <label className="form-label">Title</label>
              <input
                type="text"
                name="title"
                className="input"
                value={problemData.title}
                onChange={handleProblemChange}
                placeholder="e.g. Two Sum"
                required
              />
            </div>
            <div className="form-group">
              <label className="form-label">Description</label>
              <textarea
                name="description"
                rows={5}
                className="input"
                style={{ resize: 'vertical' }}
                value={problemData.description}
                onChange={handleProblemChange}
                placeholder="Describe the problem..."
                required
              />
            </div>
            <div className="form-group">
              <label className="form-label">Difficulty</label>
              <select
                name="difficulty"
                className="select"
                value={problemData.difficulty}
                onChange={handleProblemChange}
              >
                <option value="easy">Easy</option>
                <option value="medium">Medium</option>
                <option value="hard">Hard</option>
              </select>
            </div>
            {isGlobal && (
              <>
                <div className="form-group">
                  <label className="form-label">Subject</label>
                  <select
                    className="select"
                    value={selectedSubject}
                    onChange={(e) => {
                      setSelectedSubject(Number(e.target.value));
                      setProblemData({ ...problemData, tags: '' });
                    }}
                  >
                    <option value={0}>Select Subject (optional)</option>
                    {subjects.map((s) => (
                      <option key={s.id} value={s.id}>
                        {s.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="form-group">
                  <label className="form-label">Topic</label>
                  <select
                    className="select"
                    value={problemData.tags}
                    disabled={!selectedSubject}
                    onChange={(e) => setProblemData({ ...problemData, tags: e.target.value })}
                  >
                    <option value="">Select Topic (optional)</option>
                    {topics.map((t) => (
                      <option key={t.id} value={t.name}>
                        {t.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="form-group">
                  <label className="form-label">Tags (comma-separated)</label>
                  <input
                    type="text"
                    name="tags"
                    className="input"
                    value={problemData.tags}
                    onChange={handleProblemChange}
                    placeholder="e.g. Arrays, Strings"
                  />
                </div>
              </>
            )}
            {contestId && (
              <div className="form-group">
                <label className="form-label">Points for this Contest</label>
                <input
                  type="number"
                  className="input"
                  value={contestPoints}
                  onChange={(e) => setContestPoints(parseInt(e.target.value) || 100)}
                  min={1}
                  placeholder="100"
                />
              </div>
            )}

            <div className="form-row">
              <div className="form-group">
                <label className="form-label">Time Limit (ms)</label>
                <input
                  type="number"
                  name="time_limit"
                  className="input"
                  value={problemData.time_limit}
                  onChange={handleProblemChange}
                />
              </div>
              <div className="form-group">
                <label className="form-label">Memory Limit (KB)</label>
                <input
                  type="number"
                  name="memory_limit"
                  className="input"
                  value={problemData.memory_limit}
                  onChange={handleProblemChange}
                />
              </div>
            </div>
            {error && (
              <p
                style={{ color: 'var(--accent-danger, #ef4444)', fontSize: '0.875rem', margin: 0 }}
              >
                {error}
              </p>
            )}
            <button
              type="submit"
              className="btn btn-primary"
              style={{ width: '100%', justifyContent: 'center' }}
            >
              Next: Add Test Cases
            </button>
          </form>
        </>
      ) : (
        <>
          <h2 style={{ fontSize: '1.25rem', fontWeight: '600', marginBottom: '0.5rem' }}>
            Add Test Cases
          </h2>
          <p
            style={{
              fontSize: '0.875rem',
              color: 'var(--text-secondary, #a1a1aa)',
              marginBottom: '1rem',
            }}
          >
            Add test cases for <strong>{problemData.title}</strong>. Mark sample test cases visible
            to users.
          </p>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
            {testCases.map((tc, index) => (
              <div
                key={index}
                style={{
                  border: '1px solid var(--background-border, #2a2a2d)',
                  borderRadius: '0.5rem',
                  padding: '1rem',
                  background: 'var(--background-tertiary, #1c1c1e)',
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    marginBottom: '0.75rem',
                  }}
                >
                  <strong style={{ fontSize: '0.875rem' }}>Test Case {index + 1}</strong>
                  {testCases.length > 1 && (
                    <button
                      type="button"
                      className="btn btn-danger"
                      style={{ padding: '0.25rem 0.5rem', fontSize: '0.75rem' }}
                      onClick={() => removeTestCase(index)}
                    >
                      Remove
                    </button>
                  )}
                </div>
                <div className="form-group" style={{ marginBottom: '0.5rem' }}>
                  <label className="form-label">Input</label>
                  <textarea
                    rows={2}
                    className="input"
                    style={{ resize: 'vertical' }}
                    value={tc.input}
                    onChange={(e) => updateTestCase(index, 'input', e.target.value)}
                  />
                </div>
                <div className="form-group" style={{ marginBottom: '0.5rem' }}>
                  <label className="form-label">Expected Output</label>
                  <textarea
                    rows={2}
                    className="input"
                    style={{ resize: 'vertical' }}
                    value={tc.expected_output}
                    onChange={(e) => updateTestCase(index, 'expected_output', e.target.value)}
                  />
                </div>
                <div style={{ display: 'flex', gap: '1rem', alignItems: 'center' }}>
                  <label
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '0.5rem',
                      fontSize: '0.875rem',
                      cursor: 'pointer',
                    }}
                  >
                    <input
                      type="checkbox"
                      checked={tc.is_sample}
                      onChange={(e) => updateTestCase(index, 'is_sample', e.target.checked)}
                    />
                    Sample (visible to users)
                  </label>
                  <label
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '0.5rem',
                      fontSize: '0.875rem',
                    }}
                  >
                    Points:{' '}
                    <input
                      type="number"
                      className="input"
                      value={tc.points}
                      onChange={(e) => updateTestCase(index, 'points', parseInt(e.target.value))}
                      style={{ width: '70px' }}
                    />
                  </label>
                </div>
              </div>
            ))}
          </div>
          <button
            type="button"
            className="btn btn-secondary"
            onClick={addTestCase}
            style={{ marginTop: '0.75rem', width: '100%', justifyContent: 'center' }}
          >
            + Add Test Case
          </button>
          {error && (
            <p
              style={{
                color: 'var(--accent-danger, #ef4444)',
                fontSize: '0.875rem',
                margin: '0.75rem 0 0',
              }}
            >
              {error}
            </p>
          )}
          <div style={{ display: 'flex', gap: '0.5rem', marginTop: '1rem' }}>
            <button
              type="button"
              className="btn btn-secondary"
              style={{ flex: 1, justifyContent: 'center' }}
              onClick={() => setStep(1)}
            >
              Back
            </button>
            <button
              type="button"
              className="btn btn-success"
              style={{ flex: 2, justifyContent: 'center' }}
              onClick={handleSubmit}
              disabled={loading}
            >
              {loading ? 'Creating...' : 'Create Problem'}
            </button>
          </div>
        </>
      )}
    </Modal>
  );
}

export default CreateProblemModal;
