import { useState } from 'react';

export interface TestCase {
  input: string;
  expected_output: string;
  is_sample?: boolean;
}

export interface TestCasePanelProps {
  testCases: TestCase[];
}

function TestCasePanel({ testCases }: TestCasePanelProps) {
  const [selectedIndex, setSelectedIndex] = useState(0);

  const sampleCases = testCases.filter((tc) => tc.is_sample);

  if (sampleCases.length === 0) {
    return <p className="text-muted">No sample test cases available</p>;
  }

  const currentCase = sampleCases[selectedIndex];

  return (
    <div>
      <div className="testcase-selector" style={{ marginBottom: '1rem' }}>
        {sampleCases.map((_, index) => (
          <button
            key={index}
            className={`testcase-btn ${selectedIndex === index ? 'active' : ''}`}
            onClick={() => setSelectedIndex(index)}
          >
            Case {index + 1}
          </button>
        ))}
      </div>
      <div className="testcase-display">
        <div className="testcase-section">
          <label>Input:</label>
          <pre className="testcase-pre">{currentCase?.input || ''}</pre>
        </div>
        <div className="testcase-section">
          <label>Expected Output:</label>
          <pre className="testcase-pre">{currentCase?.expected_output || ''}</pre>
        </div>
      </div>
    </div>
  );
}

export default TestCasePanel;
