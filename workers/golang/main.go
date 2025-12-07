package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	QueueKey        = "execution_queue:golang"
	ResultKeyPrefix = "result:"
	ResultTTL       = 10 * time.Minute
)

type ExecutionRequest struct {
	InvocationID int64                  `json:"invocationId"`
	Code         string                 `json:"code"`
	Input        map[string]interface{} `json:"input"`
}

type ExecutionResult struct {
	InvocationID int64       `json:"invocationId"`
	Status       string      `json:"status"`
	Output       interface{} `json:"output"`
	OutputRaw    string      `json:"outputRaw"`
	ErrorMessage string      `json:"errorMessage"`
	Logs         string      `json:"logs"`
	DurationMs   int64       `json:"durationMs"`
}

func main() {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort)
	log.Printf("Go Worker started. Connecting to Redis at %s", redisAddr)

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   0,
	})

	ctx := context.Background()

	// Test connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis successfully")

	for {
		// Block and wait for job from queue
		result, err := rdb.BRPop(ctx, 5*time.Second, QueueKey).Result()
		if err != nil {
			if err == redis.Nil {
				continue // Timeout, no job available
			}
			log.Printf("Error reading from queue: %v", err)
			continue
		}

		// result[0] is the queue key, result[1] is the data
		rawData := result[1]

		var req ExecutionRequest
		if err := json.Unmarshal([]byte(rawData), &req); err != nil {
			log.Printf("Error parsing request JSON: %v", err)
			continue
		}

		log.Printf("Processing invocation: %d", req.InvocationID)

		startTime := time.Now()
		status, output, logs := RunCode(req.Code, req.Input)
		duration := time.Since(startTime).Milliseconds()

		var outputParsed interface{}
		errorMessage := ""

		if status == "SUCCESS" {
			if output != "" {
				if err := json.Unmarshal([]byte(output), &outputParsed); err != nil {
					outputParsed = map[string]string{"result": output}
				}
			}
		} else {
			errorMessage = output
		}

		execResult := ExecutionResult{
			InvocationID: req.InvocationID,
			Status:       status,
			Output:       outputParsed,
			OutputRaw:    output,
			ErrorMessage: errorMessage,
			Logs:         logs,
			DurationMs:   duration,
		}

		resultJSON, err := json.Marshal(execResult)
		if err != nil {
			log.Printf("Error marshaling result: %v", err)
			continue
		}

		resultKey := ResultKeyPrefix + strconv.FormatInt(req.InvocationID, 10)
		if err := rdb.Set(ctx, resultKey, resultJSON, ResultTTL).Err(); err != nil {
			log.Printf("Error storing result: %v", err)
			continue
		}

		log.Printf("Finished invocation: %d - %s", req.InvocationID, status)
	}
}
