import { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { api } from '../api/lambdaApi';
import { FunctionListItem } from '../types';
import './FunctionList.css';

export function FunctionList() {
  const [functions, setFunctions] = useState<FunctionListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    loadFunctions();
  }, []);

  const loadFunctions = async () => {
    try {
      setLoading(true);
      const data = await api.listFunctions();
      setFunctions(data);
      setError(null);
    } catch (err) {
      setError('Failed to load functions');
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString('ko-KR', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const getRuntimeLabel = (runtime: string) => {
    if (runtime.includes('python')) return 'Python';
    if (runtime.includes('node') || runtime.includes('javascript')) return 'JavaScript';
    return runtime;
  };

  if (loading) {
    return (
      <div className="function-list">
        <div className="loading">Loading...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="function-list">
        <div className="error">{error}</div>
      </div>
    );
  }

  return (
    <div className="function-list">
      <div className="list-header">
        <h1>Functions</h1>
        <button className="create-btn" onClick={() => navigate('/functions/new')}>
          + Create Function
        </button>
      </div>

      {functions.length === 0 ? (
        <div className="empty-state">
          <p>No functions yet.</p>
          <p>Create your first function to get started!</p>
        </div>
      ) : (
        <div className="function-grid">
          {functions.map((fn) => (
            <Link to={`/functions/${fn.id}`} key={fn.id} className="function-card">
              <div className="card-header">
                <h3 className="function-name">{fn.name}</h3>
                <span className={`runtime-badge ${fn.runtime.includes('python') ? 'python' : 'javascript'}`}>
                  {getRuntimeLabel(fn.runtime)}
                </span>
              </div>
              <p className="function-description">{fn.description}</p>
              <div className="card-footer">
                <span className="created-at">{formatDate(fn.created_at)}</span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
