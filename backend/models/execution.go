package models

// ExecutionRequest represents a request to execute code (sent to Redis queue)
type ExecutionRequest struct {
	InvocationID int64                  `json:"invocationId"`
	FunctionID   int64                  `json:"functionId"`
	Code         string                 `json:"code"`
	Input        map[string]interface{} `json:"input"`
	Runtime      string                 `json:"runtime"`
}

// ExecutionResult represents the result from worker (stored in Redis)
type ExecutionResult struct {
	InvocationID int64                  `json:"invocationId"`
	Status       string                 `json:"status"`
	Output       map[string]interface{} `json:"output,omitempty"`
	OutputRaw    string                 `json:"outputRaw,omitempty"`
	ErrorMessage string                 `json:"errorMessage,omitempty"`
	Logs         string                 `json:"logs,omitempty"`
	DurationMs   int                    `json:"durationMs"`
}
