package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

const (
	tradeKeyPrefix   = "trades"
	metricsKeyPrefix = "metrics"
)

func (r *RedisRepository) Save(ctx context.Context, trade domain.Trade) error {
	data, err := json.Marshal(trade)
	if err != nil {
		return fmt.Errorf("failed to marshal trade: %w", err)
	}

	key := fmt.Sprintf("%s:%s", tradeKeyPrefix, trade.Symbol)
	cmd := r.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(trade.Time.Unix()),
		Member: string(data),
	})

	if err := cmd.Err(); err != nil {
		return fmt.Errorf("failed to save trade: %w", err)
	}

	r.client.Expire(ctx, key, r.ttl)

	count, _ := r.client.ZCard(ctx, key).Result()
	log.Printf("[TRADES] Symbol: %s Count: %d", trade.Symbol, count)

	return nil
}

func (r *RedisRepository) FindBySymbol(ctx context.Context, symbol string, limit int) ([]domain.Trade, error) {
	key := fmt.Sprintf("%s:%s", tradeKeyPrefix, symbol)
	cmd := r.client.ZRevRange(ctx, key, 0, int64(limit-1))
	if cmd.Err() != nil {
		return nil, fmt.Errorf("failed to get trades: %w", cmd.Err())
	}

	data, err := cmd.Result()
	if err != nil {
		return nil, err
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

func (r *RedisRepository) FindByID(ctx context.Context, id int64) (*domain.Trade, error) {
	return nil, fmt.Errorf("operation not supported")
}

func (r *RedisRepository) SaveMetrics(ctx context.Context, metrics *domain.TradeMetrics) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	key := fmt.Sprintf("%s:%s", metricsKeyPrefix, metrics.Symbol)
	if err := r.client.Set(ctx, key, data, r.ttl).Err(); err != nil {
		return fmt.Errorf("failed to save metrics: %w", err)
	}

	log.Printf("[METRICS] Saved for %s: %s", metrics.Symbol, string(data))
	return nil
}

func (r *RedisRepository) GetLatestMetrics(ctx context.Context, symbol string) (*domain.TradeMetrics, error) {
	key := fmt.Sprintf("%s:%s", metricsKeyPrefix, symbol)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("metrics not found")
		}
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	var metrics domain.TradeMetrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	return &metrics, nil
}
