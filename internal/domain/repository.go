package domain

import (
	"context"
)

type TradeRepository interface {
	Save(ctx context.Context, trade Trade) error
	FindByID(ctx context.Context, id int64) (*Trade, error)
	FindBySymbol(ctx context.Context, symbol string, limit int) ([]Trade, error)
	SaveMetrics(ctx context.Context, metrics *TradeMetrics) error
	GetLatestMetrics(ctx context.Context, symbol string) (*TradeMetrics, error)
}

type TaskRepository interface {
	Save(ctx context.Context, task Task) error
	UpdateStatus(ctx context.Context, taskID string, status TaskStatus) error
	FindByID(ctx context.Context, id string) (Task, error)
	FindByStatus(ctx context.Context, status TaskStatus) ([]Task, error)
}
