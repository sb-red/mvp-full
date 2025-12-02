package models

import (
	"time"
)

// Function represents a serverless function metadata
type Function struct {
	ID          int64                  `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Runtime     string                 `json:"runtime"`
	CodeS3Key   string                 `json:"code_s3_key,omitempty"`
	Code        string                 `json:"code,omitempty"`
	SampleEvent map[string]interface{} `json:"sample_event,omitempty"`
	IsPublic    bool                   `json:"is_public"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Params      []FunctionParam        `json:"params,omitempty"`
}

// FunctionParam represents a parameter definition for a function
type FunctionParam struct {
	ID           int64                  `json:"id,omitempty"`
	FunctionID   int64                  `json:"function_id,omitempty"`
	ParamKey     string                 `json:"key"`
	ParamType    string                 `json:"type"`
	IsRequired   bool                   `json:"required"`
	Description  string                 `json:"description,omitempty"`
	DefaultValue map[string]interface{} `json:"default_value,omitempty"`
}

// FunctionListItem represents a function in list view (without code)
type FunctionListItem struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Runtime     string    `json:"runtime"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateFunctionRequest represents the request body for creating a function
type CreateFunctionRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Runtime     string                 `json:"runtime"`
	Params      []FunctionParam        `json:"params"`
	SampleEvent map[string]interface{} `json:"sample_event"`
	Code        string                 `json:"code"`
}

// InvokeRequest represents the request body for invoking a function
type InvokeRequest struct {
	Params map[string]interface{} `json:"params"`
}
