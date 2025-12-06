// Function parameter definition
export interface FunctionParam {
  id?: number;
  function_id?: number;
  key: string;
  type: string;
  required: boolean;
  description?: string;
  default_value?: unknown;
}

// Function for list view
export interface FunctionListItem {
  id: number;
  name: string;
  description: string;
  runtime: string;
  created_at: string;
}

// Full function detail
export interface FunctionDetail {
  id: number;
  name: string;
  description: string;
  runtime: string;
  code: string;
  params: FunctionParam[];
  sample_event?: Record<string, unknown>;
  is_public: boolean;
  created_at: string;
  updated_at: string;
}

// Create function request
export interface CreateFunctionRequest {
  name: string;
  description: string;
  runtime: string;
  params: FunctionParam[];
  sample_event?: Record<string, unknown>;
  code: string;
}

// Invoke request
export interface InvokeRequest {
  params: Record<string, unknown>;
}

// Invocation response (from invoke API)
export interface InvokeResponse {
  status: string;
  function_id: number;
  invocation_id: number;
  input_event: Record<string, unknown>;
  result?: Record<string, unknown>;
  error_message?: string;
  duration_ms: number;
  logged_at: string;
}

// Invocation list item
export interface InvocationListItem {
  id: number;
  function_id: number;
  invoked_at: string;
  input_event: Record<string, unknown>;
  status: string;
  output_result?: Record<string, unknown>;
  error_message?: string;
  duration_ms: number;
}

// Execution state for UI
export interface ExecutionState {
  isExecuting: boolean;
  invocationId: number | null;
  result: InvokeResponse | null;
  error: string | null;
}

export interface FunctionSchedule {
  id: number;
  function_id: number;
  scheduled_at: string;
  payload: Record<string, unknown>;
  executed: boolean;
  executed_at?: string;
  status?: string;
  error_message?: string;
}

export interface CreateScheduleRequest {
  scheduled_at: string;
  payload?: Record<string, unknown>;
}
