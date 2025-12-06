package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-xray-sdk-go/xray"
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
	var err error
	xray.Capture(ctx, "Redis.LPush", func(ctx1 context.Context) error {
		jsonData, marshalErr := json.Marshal(req)
		if marshalErr != nil {
			err = marshalErr
			return marshalErr
		}
		err = r.client.LPush(ctx, queueKey, string(jsonData)).Err()

		// Add metadata to subsegment
		if seg := xray.GetSegment(ctx1); seg != nil {
			seg.AddMetadata("redis.queue_key", queueKey)
			seg.AddMetadata("redis.operation", "LPUSH")
		}

		return err
	})
	return err
}

// GetResult retrieves execution result for an invocation ID
func (r *RedisService) GetResult(ctx context.Context, invocationID int64) (*models.ExecutionResult, error) {
	var result *models.ExecutionResult
	var finalErr error

	xray.Capture(ctx, "Redis.Get", func(ctx1 context.Context) error {
		key := fmt.Sprintf("%s%d", ResultKeyPrefix, invocationID)
		jsonData, err := r.client.Get(ctx, key).Result()
		if err == redis.Nil {
			result = nil
			finalErr = nil
			return nil
		}
		if err != nil {
			finalErr = err
			return err
		}

		var execResult models.ExecutionResult
		if err := json.Unmarshal([]byte(jsonData), &execResult); err != nil {
			finalErr = err
			return err
		}
		result = &execResult
		finalErr = nil

		// Add metadata to subsegment
		if seg := xray.GetSegment(ctx1); seg != nil {
			seg.AddMetadata("redis.key", key)
			seg.AddMetadata("redis.operation", "GET")
			seg.AddMetadata("redis.invocation_id", invocationID)
		}

		return nil
	})

	return result, finalErr
}

// Ping checks Redis connection
func (r *RedisService) Ping(ctx context.Context) error {
	var err error
	xray.Capture(ctx, "Redis.Ping", func(ctx1 context.Context) error {
		err = r.client.Ping(ctx).Err()

		// Add metadata to subsegment
		if seg := xray.GetSegment(ctx1); seg != nil {
			seg.AddMetadata("redis.operation", "PING")
		}

		return err
	})
	return err
}
