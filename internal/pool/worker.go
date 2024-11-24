package pool

import (
	"context"
	"github.com/pedrosfaria2/worker_pool_crypto/internal/domain"
	"sync"
	"time"
)

type worker struct {
	id              int
	tasks           chan domain.Task
	quit            chan bool
	wg              *sync.WaitGroup
	onWorkerStarted func()
	onWorkerDone    func()
	metrics         Metrics
}

func newWorker(id int, tasks chan domain.Task, wg *sync.WaitGroup, onStarted, onDone func(), metrics Metrics) *worker {
	return &worker{
		id:              id,
		tasks:           tasks,
		quit:            make(chan bool),
		wg:              wg,
		onWorkerStarted: onStarted,
		onWorkerDone:    onDone,
		metrics:         metrics,
	}
}

func (w *worker) Start(ctx context.Context) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		for {
			select {
			case task := <-w.tasks:
				w.processTask(ctx, task)
			case <-w.quit:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (w *worker) processTask(ctx context.Context, task domain.Task) {
	w.onWorkerStarted()
	start := time.Now()

	err := task.Process(ctx)
	duration := float64(time.Since(start).Milliseconds())

	if err != nil {
		task.SetError(err)
		task.SetStatus(domain.TaskFailed)
		w.metrics.TaskFailed(task.Type())
		if task.RetryCount() < task.MaxRetries() {
			task.IncrementRetry()
			w.tasks <- task
		}
	} else {
		task.SetStatus(domain.TaskCompleted)
		w.metrics.TaskCompleted(task.Type(), duration)
	}

	w.onWorkerDone()
}

func (w *worker) Stop() {
	w.quit <- true
}
