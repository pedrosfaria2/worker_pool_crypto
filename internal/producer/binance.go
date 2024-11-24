package producer

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pedrosfaria2/worker_pool_crypto/internal/domain"
	"github.com/pedrosfaria2/worker_pool_crypto/internal/pool"
)

type BinanceProducer struct {
	conn    *websocket.Conn
	pool    pool.Pool
	symbols []string
	mu      sync.Mutex
	done    chan struct{}
}

func NewBinanceProducer(symbols []string, pool pool.Pool) (*BinanceProducer, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("at least one symbol is required")
	}

	return &BinanceProducer{
		symbols: symbols,
		pool:    pool,
		done:    make(chan struct{}),
	}, nil
}

func (b *BinanceProducer) Connect(ctx context.Context) error {
	streams := make([]string, len(b.symbols))
	for i, symbol := range b.symbols {
		streams[i] = fmt.Sprintf("%s@trade", strings.ToLower(symbol))
	}

	url := fmt.Sprintf("wss://stream.binance.com:9443/ws/%s", strings.Join(streams, "/"))

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("websocket dial error: %w", err)
	}

	b.mu.Lock()
	b.conn = conn
	b.mu.Unlock()

	return nil
}

func (b *BinanceProducer) Start(ctx context.Context) error {
	if b.conn == nil {
		return fmt.Errorf("connection not established")
	}

	go b.read(ctx)

	return nil
}

func (b *BinanceProducer) Stop() error {
	close(b.done)

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.conn != nil {
		return b.conn.Close()
	}
	return nil
}

func (b *BinanceProducer) read(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-b.done:
			return
		default:
			var event domain.TradeEvent
			err := b.conn.ReadJSON(&event)
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return
				}
				continue
			}

			price, err := parseFloat(event.Price)
			if err != nil {
				fmt.Printf("error parsing price: %v\n", err)
				continue
			}

			quantity, err := parseFloat(event.Quantity)
			if err != nil {
				fmt.Printf("error parsing quantity: %v\n", err)
				continue
			}

			trade := domain.Trade{
				Symbol:   event.Symbol,
				ID:       event.TradeID,
				Price:    price,
				Quantity: quantity,
				Time:     parseTime(event.Time),
				IsBuyer:  event.IsBuyer,
				IsMaker:  false,
			}

			task := domain.NewTradeTask(trade)
			if err := b.pool.Submit(task); err != nil {
				fmt.Printf("failed to submit task: %v\n", err)
				continue
			}
		}
	}
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	if err != nil {
		return 0, fmt.Errorf("could not parse %q as float64: %w", s, err)
	}
	return f, nil
}

func parseTime(ms int64) time.Time {
	return time.Unix(0, ms*int64(time.Millisecond))
}
