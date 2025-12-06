package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"lambda-runner-server/models"
)

type ScheduleRunner struct {
	scheduleService *ScheduleService
	functionService *FunctionService
	interval        time.Duration
	batchSize       int
	stopCh          chan struct{}
	wg              sync.WaitGroup
}

func NewScheduleRunner(scheduleService *ScheduleService, functionService *FunctionService) *ScheduleRunner {
	return &ScheduleRunner{
		scheduleService: scheduleService,
		functionService: functionService,
		interval:        time.Second,
		batchSize:       20,
		stopCh:          make(chan struct{}),
	}
}

func (r *ScheduleRunner) Start() {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				r.processDueSchedules()
			case <-r.stopCh:
				return
			}
		}
	}()
}

func (r *ScheduleRunner) Stop() {
	close(r.stopCh)
	r.wg.Wait()
}

func (r *ScheduleRunner) processDueSchedules() {
	ctx := context.Background()
	schedules, err := r.scheduleService.ClaimDueSchedules(ctx, r.batchSize)
	if err != nil {
		log.Printf("scheduler: failed to claim schedules: %v", err)
		return
	}
	for _, sched := range schedules {
		go r.executeSchedule(ctx, sched)
	}
}

func (r *ScheduleRunner) executeSchedule(ctx context.Context, sched models.FunctionSchedule) {
	payload := sched.Payload
	if payload == nil {
		payload = map[string]interface{}{}
	}
	invokedBy := fmt.Sprintf("schedule:%d", sched.ID)
	inv, err := r.functionService.InvokeFunction(ctx, sched.FunctionID, payload, invokedBy)
	if err != nil {
		r.scheduleService.MarkExecuted(ctx, sched.ID, models.StatusFail, err.Error())
		return
	}

	// Poll for result (max 60 seconds)
	maxRetries := 120 // 120 * 0.5s = 60s
	for i := 0; i < maxRetries; i++ {
		time.Sleep(500 * time.Millisecond)

		result, err := r.functionService.GetInvocationResult(ctx, inv.ID)
		if err != nil {
			log.Printf("scheduler: failed to get invocation result: %v", err)
			continue
		}

		if result.Status != models.StatusPending {
			// Execution completed
			status := result.Status
			errMsg := ""
			if result.Status == models.StatusFail || result.Status == models.StatusTimeout {
				errMsg = result.ErrorMessage
			}
			r.scheduleService.MarkExecuted(ctx, sched.ID, status, errMsg)
			return
		}
	}

	// Timeout
	r.scheduleService.MarkExecuted(ctx, sched.ID, models.StatusTimeout, "execution timed out after 60 seconds")
}
