package pool

import (
	"context"

	"github.com/pedrosfaria2/worker_pool_crypto/internal/domain"
)

type Worker interface {
	Start(ctx context.Context)
	Stop()
}

type Pool interface {
	Start(ctx context.Context)
	Stop()
	Submit(task domain.Task) error
	Size() int
	Len() int
}

type Metrics interface {
	TaskSubmitted(taskType string)
	TaskCompleted(taskType string, durationMs float64)
	TaskFailed(taskType string)
	QueueSize(size int)
	WorkersBusy(count int)
}
