package domain

import (
	"context"
	"time"
)

type TaskProcessor interface {
	Process(ctz context.Context) error
	ID() string
	Type() string
	RetryCount() int
	MaxRetries() int
	LastError() error
	SetError(err error)
	IncrementRetry()
	Status() TaskStatus
	CreatedAt() time.Time
	UpdatedAt() time.Time
}

type TaskResult struct {
	TaskID    string
	Type      string
	Status    TaskStatus
	Error     error
	Timestamp time.Time
}

type BaseTask struct {
	id         string
	taskType   string
	retries    int
	maxRetries int
	lastError  error
	status     TaskStatus
	createdAt  time.Time
	updatedAt  time.Time
}

func NewBaseTask(taskType string) BaseTask {
	now := time.Now()
	return BaseTask{
		id:         generateID(),
		taskType:   taskType,
		maxRetries: 3,
		status:     TaskPending,
		createdAt:  now,
		updatedAt:  now,
	}
}

func (t *BaseTask) ID() string {
	return t.id
}

func (t *BaseTask) Type() string {
	return t.taskType
}

func (t *BaseTask) RetryCount() int {
	return t.retries
}

func (t *BaseTask) MaxRetries() int {
	return t.maxRetries
}

func (t *BaseTask) LastError() error {
	return t.lastError
}

func (t *BaseTask) SetError(err error) {
	t.lastError = err
	t.updatedAt = time.Now()
}

func (t *BaseTask) IncrementRetry() {
	t.retries++
	t.updatedAt = time.Now()
}

func (t *BaseTask) Status() TaskStatus {
	return t.status
}

func (t *BaseTask) SetStatus(status TaskStatus) {
	t.status = status
	t.updatedAt = time.Now()
}

func (t *BaseTask) CreatedAt() time.Time {
	return t.createdAt
}

func (t *BaseTask) UpdatedAt() time.Time {
	return t.updatedAt
}
