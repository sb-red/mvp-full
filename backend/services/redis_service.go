package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"lambda-runner-server/models"
)

const (
	ResultKeyPrefix = "result:"
	ResultTTL       = 10 * time.Minute
)

type RedisService struct {
	client *redis.Client
}

func NewRedisService(host string, port int) *RedisService {
	client := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", host, port),
	})
	return &RedisService{client: client}
}

// PushExecutionRequest pushes an execution request to the specified queue
func (r *RedisService) PushExecutionRequest(ctx context.Context, queueKey string, req *models.ExecutionRequest) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return r.client.LPush(ctx, queueKey, string(jsonData)).Err()
}

// GetResult retrieves execution result for an invocation ID
func (r *RedisService) GetResult(ctx context.Context, invocationID int64) (*models.ExecutionResult, error) {
	key := fmt.Sprintf("%s%d", ResultKeyPrefix, invocationID)
	jsonData, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var result models.ExecutionResult
	if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Ping checks Redis connection
func (r *RedisService) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
