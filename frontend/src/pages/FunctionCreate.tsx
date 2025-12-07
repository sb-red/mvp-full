import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import Editor from '@monaco-editor/react';
import { api } from '../api/lambdaApi';
import { FunctionParam, CreateFunctionRequest } from '../types';
import './FunctionCreate.css';

// Default code templates
const DEFAULT_PYTHON_CODE = `def handler(event):
    score = int(event.get("score", 0))

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

const DEFAULT_PYPY3_CODE = `def handler(event):
    score = int(event.get("score", 0))

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
  const score = parseInt(event.score ?? 0, 10);
  let grade;
  if (score >= 90) grade = "A";
  else if (score >= 80) grade = "B";
  else if (score >= 70) grade = "C";
  else if (score >= 60) grade = "D";
  else grade = "F";

  return {
    score,
    grade,
  };
}
`;

const DEFAULT_RUBY_CODE = `def handler(event)
  # event example: { "score" => 87 }
  score = event["score"].to_i

  grade = case score
          when 90..100 then "A"
          when 80..89 then "B"
          when 70..79 then "C"
          when 60..69 then "D"
          else "F"
          end

  { "score" => score, "grade" => grade }
end
`;

const DEFAULT_CPP_CODE = `#include <string>

json handler(const json& event) {
    int score = event["score"].get<int>();
    std::string grade;

    if (score >= 90) grade = "A";
    else if (score >= 80) grade = "B";
    else if (score >= 70) grade = "C";
    else if (score >= 60) grade = "D";
    else grade = "F";

    return {{"score", score}, {"grade", grade}};
}
`;

const DEFAULT_C99_CODE = `cJSON* handler(cJSON* event) {
    int score = cJSON_GetObjectItem(event, "score")->valueint;
    const char* grade;

    if (score >= 90) grade = "A";
    else if (score >= 80) grade = "B";
    else if (score >= 70) grade = "C";
    else if (score >= 60) grade = "D";
    else grade = "F";

    cJSON* result = cJSON_CreateObject();
    cJSON_AddNumberToObject(result, "score", score);
    cJSON_AddStringToObject(result, "grade", grade);
    return result;
}
`;

const DEFAULT_CSHARP_CODE = `using System;
using System.Text.Json;

public class Handler {
    public static object Run(JsonElement evt) {
        int score = evt.GetProperty("score").GetInt32();
        string grade;

        if (score >= 90) grade = "A";
        else if (score >= 80) grade = "B";
        else if (score >= 70) grade = "C";
        else if (score >= 60) grade = "D";
        else grade = "F";

        return new { score = score, grade = grade };
    }
}
`;

const DEFAULT_GO_CODE = `func handler(event map[string]interface{}) map[string]interface{} {
    score := int(event["score"].(float64))
    var grade string

    switch {
    case score >= 90:
        grade = "A"
    case score >= 80:
        grade = "B"
    case score >= 70:
        grade = "C"
    case score >= 60:
        grade = "D"
    default:
        grade = "F"
    }

    return map[string]interface{}{
        "score": score,
        "grade": grade,
    }
}
`;

const DEFAULT_RUST_CODE = `fn handler(event: Value) -> Value {
    let score = event["score"].as_i64().unwrap_or(0);
    let grade = match score {
        90..=100 => "A",
        80..=89 => "B",
        70..=79 => "C",
        60..=69 => "D",
        _ => "F",
    };

    json!({
        "score": score,
        "grade": grade
    })
}
`;

const DEFAULT_JAVA_CODE = `class Handler {
    public static Map<String, Object> handle(Map<String, Object> event) {
        Map<String, Object> result = new HashMap<>();

        Object raw = event.get("score");
        int score = (raw instanceof Number) ? ((Number) raw).intValue() : 0;

        String grade =
            (score >= 90) ? "A" :
            (score >= 80) ? "B" :
            (score >= 70) ? "C" :
            (score >= 60) ? "D" : "F";

        result.put("score", score);
        result.put("grade", grade);
        return result;
    }
}
`;

const DEFAULT_KOTLIN_CODE = `object Handler {
    @JvmStatic
    fun handle(event: Map<String, Any?>): Map<String, Any?> {
        val score = (event["score"] as? Number)?.toInt() ?: 0
        val grade = when {
            score >= 90 -> "A"
            score >= 80 -> "B"
            score >= 70 -> "C"
            score >= 60 -> "D"
            else -> "F"
        }
        return mapOf(
            "score" to score,
            "grade" to grade
        )
    }
}
`;

const DEFAULT_SWIFT_CODE = `func handler(event: [String: Any]) -> [String: Any] {
    let score = event["score"] as? Int ?? 0
    let grade: String

    switch score {
    case 90...100: grade = "A"
    case 80..<90: grade = "B"
    case 70..<80: grade = "C"
    case 60..<70: grade = "D"
    default: grade = "F"
    }

    return [
        "score": score,
        "grade": grade
    ]
}
`;

const DEFAULT_CODES: Record<string, string> = {
  'python3.11': DEFAULT_PYTHON_CODE,
  'pypy3': DEFAULT_PYPY3_CODE,
  'nodejs18': DEFAULT_JS_CODE,
  'ruby': DEFAULT_RUBY_CODE,
  'cpp_gcc': DEFAULT_CPP_CODE,
  'cpp17_clang': DEFAULT_CPP_CODE,
  'c99': DEFAULT_C99_CODE,
  'csharp': DEFAULT_CSHARP_CODE,
  'golang': DEFAULT_GO_CODE,
  'rust': DEFAULT_RUST_CODE,
  'java11': DEFAULT_JAVA_CODE,
  'java17': DEFAULT_JAVA_CODE,
  'java21': DEFAULT_JAVA_CODE,
  'kotlin': DEFAULT_KOTLIN_CODE,
  'swift': DEFAULT_SWIFT_CODE,
};

const getEditorLanguage = (runtime: string): string => {
  const langMap: Record<string, string> = {
    'python3.11': 'python',
    'python': 'python',
    'pypy3': 'python',
    'nodejs18': 'javascript',
    'javascript': 'javascript',
    'cpp_gcc': 'cpp',
    'cpp17_clang': 'cpp',
    'c99': 'c',
    'ruby': 'ruby',
    'csharp': 'csharp',
    'golang': 'go',
    'rust': 'rust',
    'java11': 'java',
    'java17': 'java',
    'java21': 'java',
    'kotlin': 'kotlin',
    'swift': 'swift',
  };
  return langMap[runtime] || 'plaintext';
};

const DEFAULT_RUNTIME = 'python3.11';

export function FunctionCreate() {
  const navigate = useNavigate();
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [runtime, setRuntime] = useState(DEFAULT_RUNTIME);
  const [code, setCode] = useState(DEFAULT_CODES[DEFAULT_RUNTIME]);
  const [params, setParams] = useState<FunctionParam[]>([
    { key: 'score', type: 'int', required: true, description: '시험 점수 (0~100)' }
  ]);
  const [sampleEvent, setSampleEvent] = useState('{\n  "score": 87\n}');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleRuntimeChange = (newRuntime: string) => {
    setRuntime(newRuntime);
    setCode(DEFAULT_CODES[newRuntime] || DEFAULT_JS_CODE);
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
              <optgroup label="Interpreted">
                <option value="python3.11">Python 3.11</option>
                <option value="pypy3">PyPy 3</option>
                <option value="nodejs18">Node.js 18</option>
                <option value="ruby">Ruby 3.x</option>
              </optgroup>
              <optgroup label="JVM">
                <option value="java11">Java 11</option>
                <option value="java17">Java 17</option>
                <option value="java21">Java 21</option>
                <option value="kotlin">Kotlin</option>
              </optgroup>
              <optgroup label="Compiled (Native)">
                <option value="cpp_gcc">C++ (GCC)</option>
                <option value="cpp17_clang">C++17 (Clang)</option>
                <option value="c99">C99</option>
                <option value="golang">Go</option>
                <option value="rust">Rust 2018</option>
                <option value="swift">Swift</option>
              </optgroup>
              <optgroup label="Managed">
                <option value="csharp">C# (.NET)</option>
              </optgroup>
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
              language={getEditorLanguage(runtime)}
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
