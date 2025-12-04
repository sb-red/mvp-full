import { useState, useEffect, useCallback, useRef } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import Editor from '@monaco-editor/react';
import { api } from '../api/lambdaApi';
import { FunctionDetail as FunctionDetailType, InvokeResponse, InvocationListItem } from '../types';
import './FunctionDetail.css';

export function FunctionDetail() {
  const { id } = useParams<{ id: string }>();
  const [func, setFunc] = useState<FunctionDetailType | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Execution state
  const [params, setParams] = useState<Record<string, string>>({});
  const [isExecuting, setIsExecuting] = useState(false);
  const [result, setResult] = useState<InvokeResponse | null>(null);
  const [execError, setExecError] = useState<string | null>(null);

  // Invocation history
  const [invocations, setInvocations] = useState<InvocationListItem[]>([]);

  const pollingRef = useRef<number | null>(null);
  const navigate = useNavigate();
  const [isDeleting, setIsDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  useEffect(() => {
    if (id) {
      loadFunction(parseInt(id));
      loadInvocations(parseInt(id));
    }
    return () => {
      if (pollingRef.current) clearInterval(pollingRef.current);
    };
  }, [id]);

  const loadFunction = async (functionId: number) => {
    try {
      setLoading(true);
      const data = await api.getFunction(functionId);
      setFunc(data);

      // Initialize params with sample_event or defaults
      const initialParams: Record<string, string> = {};
      data.params?.forEach((p) => {
        if (data.sample_event && data.sample_event[p.key] !== undefined) {
          initialParams[p.key] = String(data.sample_event[p.key]);
        } else if (p.default_value !== undefined) {
          initialParams[p.key] = String(p.default_value);
        } else {
          initialParams[p.key] = '';
        }
      });
      setParams(initialParams);
      setError(null);
    } catch (err) {
      setError('Function not found');
    } finally {
      setLoading(false);
    }
  };

  const loadInvocations = async (functionId: number) => {
    try {
      const data = await api.listInvocations(functionId, 10);
      setInvocations(data);
    } catch {
      // Ignore error for invocations
    }
  };

  const handleParamChange = (key: string, value: string) => {
    setParams((prev) => ({ ...prev, [key]: value }));
  };

  const parseParamValue = (value: string, type: string): unknown => {
    if (type === 'int' || type === 'number') {
      const num = parseFloat(value);
      return isNaN(num) ? value : num;
    }
    if (type === 'boolean') {
      return value.toLowerCase() === 'true';
    }
    return value;
  };

  const handleExecute = useCallback(async () => {
    if (!func || !id) return;

    setIsExecuting(true);
    setResult(null);
    setExecError(null);

    try {
      // Build params object with proper types
      const typedParams: Record<string, unknown> = {};
      func.params?.forEach((p) => {
        typedParams[p.key] = parseParamValue(params[p.key] || '', p.type);
      });

      const response = await api.invokeFunction(func.id, typedParams);

      // Start polling for result
      const invocationId = response.invocation_id;
      const functionId = func.id;

      pollingRef.current = window.setInterval(async () => {
        try {
          const pollResult = await api.getInvocationResult(functionId, invocationId);
          if (pollResult.status !== 'pending') {
            if (pollingRef.current) {
              clearInterval(pollingRef.current);
              pollingRef.current = null;
            }
            setResult(pollResult);
            setIsExecuting(false);
            loadInvocations(functionId);
          }
        } catch {
          if (pollingRef.current) {
            clearInterval(pollingRef.current);
            pollingRef.current = null;
          }
          setExecError('Failed to fetch result');
          setIsExecuting(false);
        }
      }, 500);
    } catch (err) {
      setExecError(err instanceof Error ? err.message : 'Execution failed');
      setIsExecuting(false);
    }
  }, [func, id, params]);

  const handleDelete = useCallback(async () => {
    if (!func) return;
    const confirmed = window.confirm('이 함수를 삭제할까요? 실행 기록과 코드도 함께 삭제됩니다.');
    if (!confirmed) return;

    try {
      setIsDeleting(true);
      setDeleteError(null);
      await api.deleteFunction(func.id);
      navigate('/functions');
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Failed to delete function');
    } finally {
      setIsDeleting(false);
    }
  }, [func, navigate]);

  const getEditorLanguage = (runtime: string) => {
    if (runtime.includes('python')) return 'python';
    return 'javascript';
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleString('ko-KR');
  };

  if (loading) {
    return <div className="function-detail"><div className="loading">Loading...</div></div>;
  }

  if (error || !func) {
    return <div className="function-detail"><div className="error">{error || 'Function not found'}</div></div>;
  }

  return (
    <div className="function-detail">
      <div className="detail-header">
        <Link to="/functions" className="back-link">&larr; Back to Functions</Link>
        <div className="header-row">
          <div className="header-info">
            <h1>{func.name}</h1>
            <span className={`runtime-badge ${func.runtime.includes('python') ? 'python' : 'javascript'}`}>
              {func.runtime}
            </span>
          </div>
          <button className="delete-btn" onClick={handleDelete} disabled={isDeleting}>
            {isDeleting ? 'Deleting...' : 'Delete'}
          </button>
        </div>
        {deleteError && <div className="delete-error">{deleteError}</div>}
        <p className="description">{func.description}</p>
      </div>

      <div className="detail-content">
        <div className="left-panel">
          <div className="params-section">
            <h3>Parameters</h3>
            {func.params && func.params.length > 0 ? (
              <div className="params-form">
                {func.params.map((p) => (
                  <div key={p.key} className="param-field">
                    <label>
                      <span className="param-name">{p.key}</span>
                      <span className="param-type">{p.type}</span>
                      {p.required && <span className="required">*</span>}
                    </label>
                    {p.description && <p className="param-desc">{p.description}</p>}
                    <input
                      type="text"
                      value={params[p.key] || ''}
                      onChange={(e) => handleParamChange(p.key, e.target.value)}
                      placeholder={`Enter ${p.key}`}
                    />
                  </div>
                ))}
              </div>
            ) : (
              <p className="no-params">No parameters defined</p>
            )}

            <button
              className="execute-btn"
              onClick={handleExecute}
              disabled={isExecuting}
            >
              {isExecuting ? 'Executing...' : 'Execute'}
            </button>
          </div>

          <div className="result-section">
            <h3>Result</h3>
            {isExecuting && (
              <div className="executing">
                <div className="spinner"></div>
                <span>Running...</span>
              </div>
            )}
            {execError && (
              <div className="result-error">
                <strong>Error:</strong> {execError}
              </div>
            )}
            {result && (
              <div className={`result-box ${result.status}`}>
                <div className="result-header">
                  <span className={`status-badge ${result.status}`}>{result.status}</span>
                  <span className="duration">{result.duration_ms}ms</span>
                </div>
                {result.status === 'success' && result.result && (
                  <pre className="result-output">{JSON.stringify(result.result, null, 2)}</pre>
                )}
                {(result.status === 'fail' || result.status === 'error') && result.error_message && (
                  <pre className="result-error-msg">{result.error_message}</pre>
                )}
              </div>
            )}
          </div>
        </div>

        <div className="right-panel">
          <div className="code-section">
            <h3>Code (Read-only)</h3>
            <div className="code-editor">
              <Editor
                height="400px"
                language={getEditorLanguage(func.runtime)}
                value={func.code}
                theme="vs-dark"
                options={{
                  readOnly: true,
                  minimap: { enabled: false },
                  fontSize: 14,
                  scrollBeyondLastLine: false,
                }}
              />
            </div>
          </div>
        </div>
      </div>

      <div className="history-section">
        <h3>Recent Invocations</h3>
        {invocations.length === 0 ? (
          <p className="no-history">No execution history yet</p>
        ) : (
          <table className="invocations-table">
            <thead>
              <tr>
                <th>Time</th>
                <th>Status</th>
                <th>Input</th>
                <th>Output / Error</th>
                <th>Duration</th>
              </tr>
            </thead>
            <tbody>
              {invocations.map((inv) => (
                <tr key={inv.id} className={inv.status}>
                  <td>{formatDate(inv.invoked_at)}</td>
                  <td><span className={`status-badge ${inv.status}`}>{inv.status}</span></td>
                  <td><code>{JSON.stringify(inv.input_event)}</code></td>
                  <td>
                    {inv.status === 'success' ? (
                      <code>{JSON.stringify(inv.output_result)}</code>
                    ) : (
                      <span className="error-text">{inv.error_message}</span>
                    )}
                  </td>
                  <td>{inv.duration_ms}ms</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
