package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"lambda-runner-server/models"

	_ "github.com/lib/pq"
)

type DBService struct {
	db *sql.DB
}

func NewDBService(host string, port int, user, password, dbname string) (*DBService, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &DBService{db: db}, nil
}

func (s *DBService) Close() error {
	return s.db.Close()
}

// InitSchema creates tables if they don't exist
func (s *DBService) InitSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS functions (
		id BIGSERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		description TEXT NOT NULL,
		runtime VARCHAR(50) NOT NULL,
		code_s3_key TEXT NOT NULL,
		sample_event JSONB,
		is_public BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);

	CREATE TABLE IF NOT EXISTS function_params (
		id BIGSERIAL PRIMARY KEY,
		function_id BIGINT NOT NULL REFERENCES functions(id) ON DELETE CASCADE,
		param_key VARCHAR(100) NOT NULL,
		param_type VARCHAR(50) NOT NULL,
		is_required BOOLEAN NOT NULL DEFAULT TRUE,
		description TEXT,
		default_value JSONB
	);

	CREATE TABLE IF NOT EXISTS function_invocations (
		id BIGSERIAL PRIMARY KEY,
		function_id BIGINT NOT NULL REFERENCES functions(id) ON DELETE CASCADE,
		invoked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		invoked_by VARCHAR(255),
		input_event JSONB NOT NULL,
		status VARCHAR(20) NOT NULL,
		output_result JSONB,
		error_message TEXT,
		duration_ms INTEGER,
		container_id VARCHAR(255),
		created_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);

	CREATE INDEX IF NOT EXISTS idx_function_invocations_function_id ON function_invocations(function_id);
	CREATE INDEX IF NOT EXISTS idx_function_invocations_invoked_at ON function_invocations(invoked_at DESC);
	`

	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// CreateFunction inserts a new function and its params
func (s *DBService) CreateFunction(ctx context.Context, fn *models.Function) (*models.Function, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	sampleEventJSON, _ := json.Marshal(fn.SampleEvent)

	var id int64
	var createdAt, updatedAt time.Time
	err = tx.QueryRowContext(ctx, `
		INSERT INTO functions (name, description, runtime, code_s3_key, sample_event, is_public)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`, fn.Name, fn.Description, fn.Runtime, fn.CodeS3Key, sampleEventJSON, true).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	fn.ID = id
	fn.CreatedAt = createdAt
	fn.UpdatedAt = updatedAt
	fn.IsPublic = true

	// Insert params
	for i := range fn.Params {
		param := &fn.Params[i]
		defaultValueJSON, _ := json.Marshal(param.DefaultValue)

		var paramID int64
		err = tx.QueryRowContext(ctx, `
			INSERT INTO function_params (function_id, param_key, param_type, is_required, description, default_value)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`, id, param.ParamKey, param.ParamType, param.IsRequired, param.Description, defaultValueJSON).Scan(&paramID)
		if err != nil {
			return nil, err
		}
		param.ID = paramID
		param.FunctionID = id
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return fn, nil
}

// GetFunction retrieves a function by ID with its params
func (s *DBService) GetFunction(ctx context.Context, id int64) (*models.Function, error) {
	fn := &models.Function{}
	var sampleEventJSON []byte

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, runtime, code_s3_key, sample_event, is_public, created_at, updated_at
		FROM functions WHERE id = $1
	`, id).Scan(&fn.ID, &fn.Name, &fn.Description, &fn.Runtime, &fn.CodeS3Key, &sampleEventJSON, &fn.IsPublic, &fn.CreatedAt, &fn.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if sampleEventJSON != nil {
		json.Unmarshal(sampleEventJSON, &fn.SampleEvent)
	}

	// Get params
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, function_id, param_key, param_type, is_required, description, default_value
		FROM function_params WHERE function_id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var param models.FunctionParam
		var defaultValueJSON []byte
		var desc sql.NullString
		err := rows.Scan(&param.ID, &param.FunctionID, &param.ParamKey, &param.ParamType, &param.IsRequired, &desc, &defaultValueJSON)
		if err != nil {
			return nil, err
		}
		if desc.Valid {
			param.Description = desc.String
		}
		if defaultValueJSON != nil {
			json.Unmarshal(defaultValueJSON, &param.DefaultValue)
		}
		fn.Params = append(fn.Params, param)
	}

	return fn, nil
}

// UpdateCodeKey updates the code_s3_key for a function
func (s *DBService) UpdateCodeKey(ctx context.Context, id int64, codeKey string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE functions SET code_s3_key = $2, updated_at = now() WHERE id = $1
	`, id, codeKey)
	return err
}

// DeleteFunction removes a function record (cascades to params/invocations)
func (s *DBService) DeleteFunction(ctx context.Context, id int64) (*models.Function, error) {
	fn, err := s.GetFunction(ctx, id)
	if err != nil {
		return nil, err
	}
	if fn == nil {
		return nil, nil
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM functions WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}

	return fn, nil
}

// ListFunctions returns all functions (without code)
func (s *DBService) ListFunctions(ctx context.Context) ([]models.FunctionListItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, runtime, created_at
		FROM functions ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var functions []models.FunctionListItem
	for rows.Next() {
		var fn models.FunctionListItem
		err := rows.Scan(&fn.ID, &fn.Name, &fn.Description, &fn.Runtime, &fn.CreatedAt)
		if err != nil {
			return nil, err
		}
		functions = append(functions, fn)
	}

	return functions, nil
}

// CreateInvocation creates a new invocation record
func (s *DBService) CreateInvocation(ctx context.Context, inv *models.Invocation) (*models.Invocation, error) {
	inputEventJSON, _ := json.Marshal(inv.InputEvent)

	var id int64
	var invokedAt, createdAt time.Time
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO function_invocations (function_id, invoked_by, input_event, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, invoked_at, created_at
	`, inv.FunctionID, inv.InvokedBy, inputEventJSON, inv.Status).Scan(&id, &invokedAt, &createdAt)
	if err != nil {
		return nil, err
	}

	inv.ID = id
	inv.InvokedAt = invokedAt
	inv.CreatedAt = createdAt

	return inv, nil
}

// UpdateInvocationResult updates the invocation with execution result
func (s *DBService) UpdateInvocationResult(ctx context.Context, id int64, status string, outputResult map[string]interface{}, errorMessage string, durationMs int) error {
	outputJSON, _ := json.Marshal(outputResult)

	_, err := s.db.ExecContext(ctx, `
		UPDATE function_invocations
		SET status = $2, output_result = $3, error_message = $4, duration_ms = $5
		WHERE id = $1
	`, id, status, outputJSON, errorMessage, durationMs)

	return err
}

// GetInvocation retrieves an invocation by ID
func (s *DBService) GetInvocation(ctx context.Context, id int64) (*models.Invocation, error) {
	inv := &models.Invocation{}
	var inputEventJSON, outputResultJSON []byte
	var errorMessage, invokedBy, containerID sql.NullString
	var durationMs sql.NullInt32

	err := s.db.QueryRowContext(ctx, `
		SELECT id, function_id, invoked_at, invoked_by, input_event, status, output_result, error_message, duration_ms, container_id, created_at
		FROM function_invocations WHERE id = $1
	`, id).Scan(&inv.ID, &inv.FunctionID, &inv.InvokedAt, &invokedBy, &inputEventJSON, &inv.Status, &outputResultJSON, &errorMessage, &durationMs, &containerID, &inv.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if inputEventJSON != nil {
		json.Unmarshal(inputEventJSON, &inv.InputEvent)
	}
	if outputResultJSON != nil {
		json.Unmarshal(outputResultJSON, &inv.OutputResult)
	}
	if errorMessage.Valid {
		inv.ErrorMessage = errorMessage.String
	}
	if invokedBy.Valid {
		inv.InvokedBy = invokedBy.String
	}
	if containerID.Valid {
		inv.ContainerID = containerID.String
	}
	if durationMs.Valid {
		inv.DurationMs = int(durationMs.Int32)
	}

	return inv, nil
}

// ListInvocations returns invocations for a function
func (s *DBService) ListInvocations(ctx context.Context, functionID int64, limit int) ([]models.InvocationListItem, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, function_id, invoked_at, input_event, status, output_result, error_message, duration_ms
		FROM function_invocations
		WHERE function_id = $1
		ORDER BY invoked_at DESC
		LIMIT $2
	`, functionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invocations []models.InvocationListItem
	for rows.Next() {
		var inv models.InvocationListItem
		var inputEventJSON, outputResultJSON []byte
		var errorMessage sql.NullString
		var durationMs sql.NullInt32

		err := rows.Scan(&inv.ID, &inv.FunctionID, &inv.InvokedAt, &inputEventJSON, &inv.Status, &outputResultJSON, &errorMessage, &durationMs)
		if err != nil {
			return nil, err
		}

		if inputEventJSON != nil {
			json.Unmarshal(inputEventJSON, &inv.InputEvent)
		}
		if outputResultJSON != nil {
			json.Unmarshal(outputResultJSON, &inv.OutputResult)
		}
		if errorMessage.Valid {
			inv.ErrorMessage = errorMessage.String
		}
		if durationMs.Valid {
			inv.DurationMs = int(durationMs.Int32)
		}

		invocations = append(invocations, inv)
	}

	return invocations, nil
}
