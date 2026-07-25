package core

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"

	servicejobs "github.com/a-novel/service-jobs/pkg/go"
	jobworker "github.com/a-novel/service-jobs/pkg/go/worker"

	"github.com/a-novel-kit/golib/logging"
	golibworker "github.com/a-novel-kit/golib/worker"

	"github.com/a-novel/service-narrative-engine/internal/config"
)

const (
	jobClaimLimit   = 1
	maximumJobLease = time.Hour
)

// ErrInvalidWorkerConfig is returned when the process cannot safely run the configured claim loop.
var ErrInvalidWorkerConfig = errors.New("invalid worker config")

// Worker claims narrative jobs and delegates their execution to service-jobs.
type Worker struct {
	client   servicejobs.Client
	executor *jobworker.Executor
	kinds    []string
	config   config.Worker
	logger   logging.Log
	id       string
}

// NewWorker builds the narrative-owned claim loop around the service-jobs executor.
func NewWorker(
	client servicejobs.Client,
	executor *jobworker.Executor,
	kinds []string,
	workerConfig config.Worker,
	logger logging.Log,
) (*Worker, error) {
	err := validateWorkerConfig(workerConfig)
	if err != nil {
		return nil, err
	}

	return &Worker{
		client: client, executor: executor, kinds: slices.Clone(kinds),
		config: workerConfig, logger: logger, id: uuid.NewString(),
	}, nil
}

// Run polls until ctx is cancelled, then waits for already-claimed jobs to finish within their
// drain budgets.
func (worker *Worker) Run(ctx context.Context) {
	var pollers sync.WaitGroup

	for pollerIndex := range worker.config.Concurrency {
		pollers.Go(func() {
			stagger := time.Duration(pollerIndex) * worker.config.PollInterval /
				time.Duration(worker.config.Concurrency)

			golibworker.Poll(
				ctx,
				worker.logger,
				fmt.Sprintf("narrative-jobs-%d", pollerIndex+1),
				worker.config.PollInterval,
				stagger,
				worker.runOnce,
			)
		})
	}

	pollers.Wait()
}

func (worker *Worker) runOnce(ctx context.Context) (bool, error) {
	response, err := worker.client.JobClaim(ctx, &servicejobs.JobClaimRequest{
		Kinds:        worker.kinds,
		WorkerId:     worker.id,
		Limit:        jobClaimLimit,
		LeaseSeconds: durationSecondsCeil(worker.config.Lease),
	})
	if err != nil {
		return false, fmt.Errorf("claim narrative job: %w", err)
	}

	jobs := response.GetJobs()
	for _, job := range jobs {
		executionCtx, cancelExecution := context.WithTimeout(
			context.WithoutCancel(ctx), worker.config.DrainBudget,
		)
		_, err = worker.executor.Execute(executionCtx, &jobworker.ExecuteRequest{
			Job: job, WorkerID: worker.id, Deadline: worker.config.JobDeadline,
		})

		cancelExecution()

		if errors.Is(err, jobworker.ErrClaimLost) {
			worker.logger.Warn(executionCtx, fmt.Sprintf("execute narrative job %s: claim lost", job.GetId()))

			continue
		}

		if err != nil {
			return false, fmt.Errorf("execute narrative job %s: %w", job.GetId(), err)
		}
	}

	return len(jobs) > 0, nil
}

func validateWorkerConfig(workerConfig config.Worker) error {
	switch {
	case workerConfig.Concurrency <= 0:
		return fmt.Errorf("%w: concurrency must be positive", ErrInvalidWorkerConfig)
	case workerConfig.PollInterval <= 0:
		return fmt.Errorf("%w: poll interval must be positive", ErrInvalidWorkerConfig)
	case workerConfig.JobDeadline <= 0:
		return fmt.Errorf("%w: job deadline must be positive", ErrInvalidWorkerConfig)
	case workerConfig.Lease <= workerConfig.JobDeadline:
		return fmt.Errorf("%w: lease must exceed job deadline", ErrInvalidWorkerConfig)
	case workerConfig.Lease > maximumJobLease:
		return fmt.Errorf("%w: lease must not exceed %s", ErrInvalidWorkerConfig, maximumJobLease)
	case workerConfig.DrainBudget <= workerConfig.JobDeadline:
		return fmt.Errorf("%w: drain budget must exceed job deadline", ErrInvalidWorkerConfig)
	}

	return nil
}

func durationSecondsCeil(duration time.Duration) int32 {
	// validateWorkerConfig caps this value at one hour before the worker starts.
	return int32((duration + time.Second - 1) / time.Second) //nolint:gosec
}
