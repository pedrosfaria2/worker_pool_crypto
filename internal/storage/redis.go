package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/pedrosfaria2/worker_pool_crypto/internal/domain"
	"github.com/redis/go-redis/v9"
)

type RedisRepository struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisRepository(client *redis.Client, ttl time.Duration) *RedisRepository {
	return &RedisRepository{
		client: client,
		ttl:    ttl,
	}
}

func (r *RedisRepository) Save(ctx context.Context, trade domain.Trade) error {
	key := fmt.Sprintf("trade:%d", trade.ID)
	data, err := json.Marshal(trade)
	if err != nil {
		return fmt.Errorf("failed to marshal trade: %w", err)
	}

	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *RedisRepository) FindByID(ctx context.Context, id int64) (*domain.Trade, error) {
	key := fmt.Sprintf("trade:%d", id)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get trade: %w", err)
	}

	var trade domain.Trade
	if err := json.Unmarshal(data, &trade); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trade: %w", err)
	}

	return &trade, nil
}

func (r *RedisRepository) FindBySymbol(ctx context.Context, symbol string, limit int) ([]domain.Trade, error) {
	key := fmt.Sprintf("trades:%s", symbol)
	data, err := r.client.LRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get trades: %w", err)
	}

	trades := make([]domain.Trade, 0, len(data))
	for _, item := range data {
		var trade domain.Trade
		if err := json.Unmarshal([]byte(item), &trade); err != nil {
			return nil, fmt.Errorf("failed to unmarshal trade: %w", err)
		}
		trades = append(trades, trade)
	}

	return trades, nil
}

func (r *RedisRepository) SaveTask(ctx context.Context, task domain.Task) error {
	key := fmt.Sprintf("task:%s", task.ID())
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	if err := r.client.Set(ctx, key, data, r.ttl).Err(); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	statusKey := fmt.Sprintf("tasks:%d", task.Status())
	return r.client.LPush(ctx, statusKey, task.ID()).Err()
}

func (r *RedisRepository) UpdateStatus(ctx context.Context, taskID string, status domain.TaskStatus) error {
	key := fmt.Sprintf("task:%s", taskID)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	var task domain.TradeTask
	if err := json.Unmarshal(data, &task); err != nil {
		return fmt.Errorf("failed to unmarshal task: %w", err)
	}

	task.SetStatus(status)

	updated, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal updated task: %w", err)
	}

	return r.client.Set(ctx, key, updated, r.ttl).Err()
}

func (r *RedisRepository) FindTaskByID(ctx context.Context, id string) (domain.Task, error) {
	key := fmt.Sprintf("task:%s", id)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	var task domain.TradeTask
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

func (r *RedisRepository) FindTasksByStatus(ctx context.Context, status domain.TaskStatus) ([]domain.Task, error) {
	key := fmt.Sprintf("tasks:%d", status)
	taskIDs, err := r.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get task ids: %w", err)
	}

	tasks := make([]domain.Task, 0, len(taskIDs))
	for _, id := range taskIDs {
		task, err := r.FindTaskByID(ctx, id)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}
