import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import Editor from '@monaco-editor/react';
import { api } from '../api/lambdaApi';
import { FunctionParam, CreateFunctionRequest } from '../types';
import './FunctionCreate.css';

const DEFAULT_PYTHON_CODE = `def handler(event):
    """
    event example:
    {
        "score": 87
    }
    """
    score = int(event["score"])

    if score >= 90:
        grade = "A"
    elif score >= 80:
        grade = "B"
    elif score >= 70:
        grade = "C"
    elif score >= 60:
        grade = "D"
    else:
        grade = "F"

    return {
        "score": score,
        "grade": grade
    }
`;

const DEFAULT_JS_CODE = `function handler(event) {
    /*
    event example:
    {
        "score": 87
    }
    */
    const score = parseInt(event.score);

    let grade;
    if (score >= 90) grade = "A";
    else if (score >= 80) grade = "B";
    else if (score >= 70) grade = "C";
    else if (score >= 60) grade = "D";
    else grade = "F";

    return {
        score: score,
        grade: grade
    };
}
`;

export function FunctionCreate() {
  const navigate = useNavigate();
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [runtime, setRuntime] = useState('python3.11');
  const [code, setCode] = useState(DEFAULT_PYTHON_CODE);
  const [params, setParams] = useState<FunctionParam[]>([
    { key: 'score', type: 'int', required: true, description: '시험 점수 (0~100)' }
  ]);
  const [sampleEvent, setSampleEvent] = useState('{\n  "score": 87\n}');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleRuntimeChange = (newRuntime: string) => {
    setRuntime(newRuntime);
    if (newRuntime.includes('python')) {
      setCode(DEFAULT_PYTHON_CODE);
    } else {
      setCode(DEFAULT_JS_CODE);
    }
  };

  const addParam = () => {
    setParams([...params, { key: '', type: 'string', required: true, description: '' }]);
  };

  const updateParam = (index: number, field: keyof FunctionParam, value: unknown) => {
    const updated = [...params];
    updated[index] = { ...updated[index], [field]: value };
    setParams(updated);
  };

  const removeParam = (index: number) => {
    setParams(params.filter((_, i) => i !== index));
  };

  const handleSubmit = async () => {
    if (!name.trim()) {
      setError('Function name is required');
      return;
    }
    if (!code.trim()) {
      setError('Code is required');
      return;
    }

    let parsedSampleEvent: Record<string, unknown> = {};
    try {
      parsedSampleEvent = JSON.parse(sampleEvent);
    } catch {
      setError('Sample event must be valid JSON');
      return;
    }

    setSaving(true);
    setError(null);

    try {
      const request: CreateFunctionRequest = {
        name: name.trim(),
        description: description.trim(),
        runtime,
        params: params.filter(p => p.key.trim()),
        sample_event: parsedSampleEvent,
        code,
      };

      const created = await api.createFunction(request);
      navigate(`/functions/${created.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create function');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="function-create">
      <div className="create-header">
        <Link to="/functions" className="back-link">&larr; Back to Functions</Link>
        <h1>Create New Function</h1>
      </div>

      <div className="create-form">
        <div className="form-section">
          <h3>Basic Info</h3>
          <div className="form-group">
            <label>Function Name *</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g., 점수 등급 계산기"
            />
          </div>

          <div className="form-group">
            <label>Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Describe what this function does..."
              rows={3}
            />
          </div>

          <div className="form-group">
            <label>Runtime</label>
            <select value={runtime} onChange={(e) => handleRuntimeChange(e.target.value)}>
              <option value="python3.11">Python 3.11</option>
              <option value="nodejs18">Node.js 18</option>
            </select>
          </div>
        </div>

        <div className="form-section">
          <h3>Parameters</h3>
          <div className="params-list">
            {params.map((param, index) => (
              <div key={index} className="param-row">
                <input
                  type="text"
                  placeholder="key"
                  value={param.key}
                  onChange={(e) => updateParam(index, 'key', e.target.value)}
                />
                <select
                  value={param.type}
                  onChange={(e) => updateParam(index, 'type', e.target.value)}
                >
                  <option value="string">string</option>
                  <option value="int">int</option>
                  <option value="number">number</option>
                  <option value="boolean">boolean</option>
                </select>
                <label className="required-checkbox">
                  <input
                    type="checkbox"
                    checked={param.required}
                    onChange={(e) => updateParam(index, 'required', e.target.checked)}
                  />
                  Required
                </label>
                <input
                  type="text"
                  placeholder="description"
                  value={param.description || ''}
                  onChange={(e) => updateParam(index, 'description', e.target.value)}
                  className="desc-input"
                />
                <button className="remove-btn" onClick={() => removeParam(index)}>×</button>
              </div>
            ))}
          </div>
          <button className="add-param-btn" onClick={addParam}>+ Add Parameter</button>
        </div>

        <div className="form-section">
          <h3>Sample Event (JSON)</h3>
          <textarea
            className="sample-event-input"
            value={sampleEvent}
            onChange={(e) => setSampleEvent(e.target.value)}
            rows={5}
          />
        </div>

        <div className="form-section">
          <h3>Code</h3>
          <div className="code-editor">
            <Editor
              height="400px"
              language={runtime.includes('python') ? 'python' : 'javascript'}
              value={code}
              onChange={(value) => setCode(value || '')}
              theme="vs-dark"
              options={{
                minimap: { enabled: false },
                fontSize: 14,
                scrollBeyondLastLine: false,
              }}
            />
          </div>
        </div>

        {error && <div className="form-error">{error}</div>}

        <div className="form-actions">
          <button className="cancel-btn" onClick={() => navigate('/functions')}>
            Cancel
          </button>
          <button className="save-btn" onClick={handleSubmit} disabled={saving}>
            {saving ? 'Creating...' : 'Create Function'}
          </button>
        </div>
      </div>
    </div>
  );
}
