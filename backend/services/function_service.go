package services

import (
	"context"
	"fmt"

	"lambda-runner-server/models"
)

type FunctionService struct {
	db      *DBService
	storage StorageService
	redis   *RedisService
}

func NewFunctionService(db *DBService, storage StorageService, redis *RedisService) *FunctionService {
	return &FunctionService{
		db:      db,
		storage: storage,
		redis:   redis,
	}
}

// CreateFunction creates a new function with code stored in storage
func (s *FunctionService) CreateFunction(ctx context.Context, req *models.CreateFunctionRequest) (*models.Function, error) {
	fn := &models.Function{
		Name:        req.Name,
		Description: req.Description,
		Runtime:     req.Runtime,
		SampleEvent: req.SampleEvent,
		Params:      req.Params,
	}

	// Create function in DB first to get ID
	fn.CodeS3Key = "temp" // Will be updated after we have the ID
	created, err := s.db.CreateFunction(ctx, fn)
	if err != nil {
		return nil, err
	}

	// Generate storage key and save code
	codeKey := GenerateCodeKey(created.ID, created.Runtime)
	if err := s.storage.SaveCode(ctx, codeKey, req.Code); err != nil {
		return nil, err
	}

	// Update the code_s3_key in DB
	if err := s.db.UpdateCodeKey(ctx, created.ID, codeKey); err != nil {
		return nil, err
	}
	created.CodeS3Key = codeKey
	created.Code = req.Code

	return created, nil
}

// GetFunction retrieves a function by ID
func (s *FunctionService) GetFunction(ctx context.Context, id int64) (*models.Function, error) {
	fn, err := s.db.GetFunction(ctx, id)
	if err != nil {
		return nil, err
	}
	if fn == nil {
		return nil, fmt.Errorf("function not found: %d", id)
	}

	// Load code from storage
	code, err := s.storage.GetCode(ctx, fn.CodeS3Key)
	if err != nil {
		return nil, err
	}
	fn.Code = code

	return fn, nil
}

// ListFunctions returns all functions
func (s *FunctionService) ListFunctions(ctx context.Context) ([]models.FunctionListItem, error) {
	return s.db.ListFunctions(ctx)
}

// InvokeFunction executes a function and returns invocation ID
func (s *FunctionService) InvokeFunction(ctx context.Context, functionID int64, params map[string]interface{}, invokedBy string) (*models.Invocation, error) {
	// Get function
	fn, err := s.GetFunction(ctx, functionID)
	if err != nil {
		return nil, err
	}

	// Create invocation record
	inv := &models.Invocation{
		FunctionID: functionID,
		InputEvent: params,
		InvokedBy:  invokedBy,
		Status:     models.StatusPending,
	}

	created, err := s.db.CreateInvocation(ctx, inv)
	if err != nil {
		return nil, err
	}

	// Push to Redis queue
	execReq := &models.ExecutionRequest{
		InvocationID: created.ID,
		FunctionID:   functionID,
		Code:         fn.Code,
		Input:        params,
		Runtime:      fn.Runtime,
	}

	queueName := getQueueName(fn.Runtime)
	if err := s.redis.PushExecutionRequest(ctx, queueName, execReq); err != nil {
		return nil, err
	}

	return created, nil
}

// GetInvocation retrieves an invocation by ID
func (s *FunctionService) GetInvocation(ctx context.Context, id int64) (*models.Invocation, error) {
	return s.db.GetInvocation(ctx, id)
}

// GetInvocationResult polls Redis for result and updates DB
func (s *FunctionService) GetInvocationResult(ctx context.Context, invocationID int64) (*models.Invocation, error) {
	// First check if result is already in DB
	inv, err := s.db.GetInvocation(ctx, invocationID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, fmt.Errorf("invocation not found: %d", invocationID)
	}

	// If already completed, return from DB
	if inv.Status != models.StatusPending {
		return inv, nil
	}

	// Check Redis for result
	result, err := s.redis.GetResult(ctx, invocationID)
	if err != nil {
		return nil, err
	}

	if result != nil {
		// Update DB with result
		status := result.Status
		if status == "SUCCESS" {
			status = models.StatusSuccess
		} else if status == "ERROR" {
			status = models.StatusFail
		} else if status == "TIMEOUT" {
			status = models.StatusTimeout
		}

		err = s.db.UpdateInvocationResult(ctx, invocationID, status, result.Output, result.ErrorMessage, result.DurationMs)
		if err != nil {
			return nil, err
		}

		// Return updated invocation
		return s.db.GetInvocation(ctx, invocationID)
	}

	// Still pending
	return inv, nil
}

// ListInvocations returns invocations for a function
func (s *FunctionService) ListInvocations(ctx context.Context, functionID int64, limit int) ([]models.InvocationListItem, error) {
	return s.db.ListInvocations(ctx, functionID, limit)
}

// getQueueName returns the Redis queue name based on runtime
func getQueueName(runtime string) string {
	if runtime == "python3.11" || runtime == "python" {
		return "execution_queue:python"
	}
	return "execution_queue:javascript"
}
