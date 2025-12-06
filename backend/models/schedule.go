package models

import "time"

// FunctionSchedule represents a one-time scheduled execution for a function
type FunctionSchedule struct {
	ID           int64                  `json:"id"`
	FunctionID   int64                  `json:"function_id"`
	ScheduledAt  time.Time              `json:"scheduled_at"`
	Payload      map[string]interface{} `json:"payload"`
	Executed     bool                   `json:"executed"`
	ExecutedAt   *time.Time             `json:"executed_at,omitempty"`
	Status       string                 `json:"status,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// CreateScheduleRequest is used to register a new schedule
type CreateScheduleRequest struct {
	ScheduledAt time.Time              `json:"scheduled_at"`
	Payload     map[string]interface{} `json:"payload"`
}
