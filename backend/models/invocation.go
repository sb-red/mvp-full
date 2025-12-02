package models

import (
	"time"
)

// Invocation represents a function execution log (function_invocations table)
type Invocation struct {
	ID           int64                  `json:"id"`
	FunctionID   int64                  `json:"function_id"`
	InvokedAt    time.Time              `json:"invoked_at"`
	InvokedBy    string                 `json:"invoked_by,omitempty"`
	InputEvent   map[string]interface{} `json:"input_event"`
	Status       string                 `json:"status"`
	OutputResult map[string]interface{} `json:"output_result,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	DurationMs   int                    `json:"duration_ms"`
	ContainerID  string                 `json:"container_id,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// InvocationStatus constants
const (
	StatusSuccess = "success"
	StatusFail    = "fail"
	StatusTimeout = "timeout"
	StatusPending = "pending"
)

// InvokeResponse represents the response for function invocation
type InvokeResponse struct {
	Status       string                 `json:"status"`
	FunctionID   int64                  `json:"function_id"`
	InvocationID int64                  `json:"invocation_id"`
	InputEvent   map[string]interface{} `json:"input_event"`
	Result       map[string]interface{} `json:"result,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	DurationMs   int                    `json:"duration_ms"`
	LoggedAt     time.Time              `json:"logged_at"`
}

// InvocationListItem represents an invocation in list view
type InvocationListItem struct {
	ID           int64                  `json:"id"`
	FunctionID   int64                  `json:"function_id"`
	InvokedAt    time.Time              `json:"invoked_at"`
	InputEvent   map[string]interface{} `json:"input_event"`
	Status       string                 `json:"status"`
	OutputResult map[string]interface{} `json:"output_result,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	DurationMs   int                    `json:"duration_ms"`
}
