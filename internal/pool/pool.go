package pool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/pedrosfaria2/worker_pool_crypto/internal/domain"
)

var ErrPoolFull = errors.New("pool queue is full")

type pool struct {
	workers     []*worker
	tasks       chan domain.Task
	wg          sync.WaitGroup
	size        int
	metrics     Metrics
	busyWorkers int32
}

func NewPool(size, queueSize int, metrics Metrics) Pool {
	p := &pool{
		workers: make([]*worker, size),
		tasks:   make(chan domain.Task, queueSize),
		size:    size,
		metrics: metrics,
	}

	for i := 0; i < size; i++ {
		p.workers[i] = newWorker(i, p.tasks, &p.wg, p.workerStarted, p.workerFinished, metrics)
	}

	return p
}

func (p *pool) Start(ctx context.Context) {
	for _, worker := range p.workers {
		worker.Start(ctx)
	}
}

func (p *pool) Stop() {
	for _, worker := range p.workers {
		worker.Stop()
	}
	p.wg.Wait()
}

func (p *pool) Submit(task domain.Task) error {
	select {
	case p.tasks <- task:
		p.metrics.TaskSubmitted(task.Type())
		p.metrics.QueueSize(len(p.tasks))
		return nil
	default:
		return ErrPoolFull
	}
}

func (p *pool) Size() int {
	return p.size
}

func (p *pool) Len() int {
	return len(p.tasks)
}

func (p *pool) workerStarted() {
	busy := atomic.AddInt32(&p.busyWorkers, 1)
	p.metrics.WorkersBusy(int(busy))
}

func (p *pool) workerFinished() {
	busy := atomic.AddInt32(&p.busyWorkers, -1)
	p.metrics.WorkersBusy(int(busy))
}
