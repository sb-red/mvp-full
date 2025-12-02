import {
  FunctionListItem,
  FunctionDetail,
  CreateFunctionRequest,
  InvokeResponse,
  InvocationListItem,
} from '../types';

const API_BASE = '/api';

export const api = {
  // List all functions
  async listFunctions(): Promise<FunctionListItem[]> {
    const res = await fetch(`${API_BASE}/functions`);
    if (!res.ok) throw new Error('Failed to fetch functions');
    return res.json();
  },

  // Get function detail
  async getFunction(id: number): Promise<FunctionDetail> {
    const res = await fetch(`${API_BASE}/functions/${id}`);
    if (!res.ok) throw new Error('Function not found');
    return res.json();
  },

  // Create a new function
  async createFunction(func: CreateFunctionRequest): Promise<FunctionDetail> {
    const res = await fetch(`${API_BASE}/functions`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(func),
    });
    if (!res.ok) throw new Error('Failed to create function');
    return res.json();
  },

  // Invoke a function
  async invokeFunction(functionId: number, params: Record<string, unknown>): Promise<InvokeResponse> {
    const res = await fetch(`${API_BASE}/functions/${functionId}/invoke`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ params }),
    });
    if (!res.ok) throw new Error('Failed to invoke function');
    return res.json();
  },

  // Get invocation result (for polling)
  async getInvocationResult(functionId: number, invocationId: number): Promise<InvokeResponse> {
    const res = await fetch(`${API_BASE}/functions/${functionId}/invocations/${invocationId}`);
    if (!res.ok) throw new Error('Failed to fetch result');
    return res.json();
  },

  // List invocations for a function
  async listInvocations(functionId: number, limit: number = 20): Promise<InvocationListItem[]> {
    const res = await fetch(`${API_BASE}/functions/${functionId}/invocations?limit=${limit}`);
    if (!res.ok) throw new Error('Failed to fetch invocations');
    return res.json();
  },
};
