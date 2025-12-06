package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"lambda-runner-server/models"
)

type ScheduleService struct {
	db *DBService
}

func NewScheduleService(db *DBService) *ScheduleService {
	return &ScheduleService{
		db: db,
	}
}

// CreateSchedule registers a new one-time scheduled execution for a function
func (s *ScheduleService) CreateSchedule(ctx context.Context, functionID int64, req *models.CreateScheduleRequest) (*models.FunctionSchedule, error) {
	if req.ScheduledAt.IsZero() {
		return nil, fmt.Errorf("scheduled_at is required")
	}

	// Validate scheduled_at is in the future
	now := time.Now().UTC()
	if req.ScheduledAt.Before(now) {
		return nil, fmt.Errorf("scheduled_at must be in the future")
	}

	payload := req.Payload
	if payload == nil {
		payload = map[string]interface{}{}
	}

	return s.db.CreateSchedule(ctx, &models.FunctionSchedule{
		FunctionID:  functionID,
		ScheduledAt: req.ScheduledAt,
		Payload:     payload,
		Executed:    false,
	})
}

// ListSchedules returns the schedules for a function
func (s *ScheduleService) ListSchedules(ctx context.Context, functionID int64) ([]models.FunctionSchedule, error) {
	return s.db.ListSchedules(ctx, functionID)
}

// DeleteSchedule removes a schedule
func (s *ScheduleService) DeleteSchedule(ctx context.Context, functionID, scheduleID int64) error {
	return s.db.DeleteSchedule(ctx, functionID, scheduleID)
}

// ClaimDueSchedules locks due schedules and returns them for execution
func (s *ScheduleService) ClaimDueSchedules(ctx context.Context, limit int) ([]models.FunctionSchedule, error) {
	if limit <= 0 {
		limit = 10
	}

	tx, err := s.db.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	rows, err := tx.QueryContext(ctx, `
		SELECT id, function_id, scheduled_at, payload, executed, executed_at, status, error_message, created_at, updated_at
		FROM function_schedules
		WHERE executed = FALSE AND scheduled_at <= $1
		ORDER BY scheduled_at
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []models.FunctionSchedule
	var scheduleIDs []int64
	for rows.Next() {
		var sched models.FunctionSchedule
		var payloadJSON []byte
		var executedAt sql.NullTime
		var status, errorMsg sql.NullString
		if err := rows.Scan(&sched.ID, &sched.FunctionID, &sched.ScheduledAt, &payloadJSON, &sched.Executed, &executedAt, &status, &errorMsg, &sched.CreatedAt, &sched.UpdatedAt); err != nil {
			return nil, err
		}
		if payloadJSON != nil {
			json.Unmarshal(payloadJSON, &sched.Payload)
		}
		if executedAt.Valid {
			sched.ExecutedAt = &executedAt.Time
		}
		if status.Valid {
			sched.Status = status.String
		}
		if errorMsg.Valid {
			sched.ErrorMessage = errorMsg.String
		}
		schedules = append(schedules, sched)
		scheduleIDs = append(scheduleIDs, sched.ID)
	}

	// Mark as executed immediately to prevent duplicate execution
	if len(scheduleIDs) > 0 {
		// Create placeholder string for IN clause
		placeholders := ""
		for i := range scheduleIDs {
			if i > 0 {
				placeholders += ","
			}
			placeholders += fmt.Sprintf("$%d", i+1)
		}

		query := fmt.Sprintf(`
			UPDATE function_schedules
			SET executed = TRUE, executed_at = now(), updated_at = now()
			WHERE id IN (%s)
		`, placeholders)

		args := make([]interface{}, len(scheduleIDs))
		for i, id := range scheduleIDs {
			args[i] = id
		}

		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return schedules, nil
}

// MarkExecuted marks a schedule as executed with result
func (s *ScheduleService) MarkExecuted(ctx context.Context, scheduleID int64, status, errMsg string) {
	_ = s.db.MarkScheduleExecuted(ctx, scheduleID, status, errMsg)
}
