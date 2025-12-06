package services

import (
	"context"
	"database/sql"
	"encoding/json"

	"lambda-runner-server/models"
)

// CreateSchedule inserts a new scheduled execution
func (s *DBService) CreateSchedule(ctx context.Context, sched *models.FunctionSchedule) (*models.FunctionSchedule, error) {
	payloadJSON, _ := json.Marshal(sched.Payload)
	var created models.FunctionSchedule
	var executedAt sql.NullTime
	var status, errorMsg sql.NullString
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO function_schedules (function_id, scheduled_at, payload, executed)
		VALUES ($1, $2, $3, FALSE)
		RETURNING id, function_id, scheduled_at, payload, executed, executed_at, status, error_message, created_at, updated_at
	`, sched.FunctionID, sched.ScheduledAt, payloadJSON).
		Scan(&created.ID, &created.FunctionID, &created.ScheduledAt, &payloadJSON, &created.Executed, &executedAt, &status, &errorMsg, &created.CreatedAt, &created.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if executedAt.Valid {
		created.ExecutedAt = &executedAt.Time
	}
	if status.Valid {
		created.Status = status.String
	}
	if errorMsg.Valid {
		created.ErrorMessage = errorMsg.String
	}
	if payloadJSON != nil {
		json.Unmarshal(payloadJSON, &created.Payload)
	}

	return &created, nil
}

// ListSchedules returns schedules for a function
func (s *DBService) ListSchedules(ctx context.Context, functionID int64) ([]models.FunctionSchedule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, function_id, scheduled_at, payload, executed, executed_at, status, error_message, created_at, updated_at
		FROM function_schedules
		WHERE function_id = $1
		ORDER BY scheduled_at DESC
	`, functionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schedules := []models.FunctionSchedule{}
	for rows.Next() {
		var sched models.FunctionSchedule
		var payloadJSON []byte
		var executedAt sql.NullTime
		var status, errorMsg sql.NullString
		if err := rows.Scan(&sched.ID, &sched.FunctionID, &sched.ScheduledAt, &payloadJSON, &sched.Executed, &executedAt, &status, &errorMsg, &sched.CreatedAt, &sched.UpdatedAt); err != nil {
			return nil, err
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
		if payloadJSON != nil {
			json.Unmarshal(payloadJSON, &sched.Payload)
		}
		schedules = append(schedules, sched)
	}

	return schedules, nil
}

// DeleteSchedule removes a schedule
func (s *DBService) DeleteSchedule(ctx context.Context, functionID, scheduleID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM function_schedules WHERE id = $1 AND function_id = $2
	`, scheduleID, functionID)
	return err
}

// MarkScheduleExecuted marks a schedule as executed with result
func (s *DBService) MarkScheduleExecuted(ctx context.Context, scheduleID int64, status, errMsg string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE function_schedules
		SET executed = TRUE, executed_at = now(), status = $2, error_message = $3, updated_at = now()
		WHERE id = $1
	`, scheduleID, status, errMsg)
	return err
}
