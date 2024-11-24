package storage

import (
	"context"

	"github.com/pedrosfaria2/worker_pool_crypto/internal/domain"
)

type TradeRepository interface {
	Save(ctx context.Context, trade domain.Trade) error
	FindByID(ctx context.Context, id int64) (*domain.Trade, error)
	FindBySymbol(ctx context.Context, symbol string, limit int) ([]domain.Trade, error)
}

type TaskRepository interface {
	Save(ctx context.Context, task domain.Task) error
	UpdateStatus(ctx context.Context, taskID string, status domain.TaskStatus) error
	FindByID(ctx context.Context, id string) (domain.Task, error)
	FindByStatus(ctx context.Context, status domain.TaskStatus) ([]domain.Task, error)
}
