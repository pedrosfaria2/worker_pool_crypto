package producer

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pedrosfaria2/worker_pool_crypto/internal/domain"
	"github.com/pedrosfaria2/worker_pool_crypto/internal/pool"
)

type BinanceProducer struct {
	conn    *websocket.Conn
	pool    pool.Pool
	repo    domain.TradeRepository
	symbols []string
	mu      sync.Mutex
	done    chan struct{}
}

type BinanceTradeEvent struct {
	EventType    string `json:"e"`
	EventTime    int64  `json:"E"`
	Symbol       string `json:"s"`
	TradeID      int64  `json:"t"`
	Price        string `json:"p"`
	Quantity     string `json:"q"`
	BuyerID      int64  `json:"b"`
	SellerID     int64  `json:"a"`
	TradeTime    int64  `json:"T"`
	IsBuyerMaker bool   `json:"m"`
}

func NewBinanceProducer(symbols []string, pool pool.Pool, repo domain.TradeRepository) (Producer, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("at least one symbol is required")
	}

	return &BinanceProducer{
		symbols: symbols,
		pool:    pool,
		repo:    repo,
		done:    make(chan struct{}),
	}, nil
}

func (b *BinanceProducer) Connect(ctx context.Context) error {
	streams := make([]string, len(b.symbols))
	for i, symbol := range b.symbols {
		streams[i] = fmt.Sprintf("%s@trade", symbol)
	}

	url := fmt.Sprintf("wss://stream.binance.com:9443/ws/%s", streams[0])
	log.Printf("Connecting to Binance WebSocket: %s", url)

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
			var event BinanceTradeEvent
			err := b.conn.ReadJSON(&event)
			if err != nil {
				log.Printf("Error reading from websocket: %v", err)
				continue
			}

			log.Printf("Received trade event: %+v", event)

			price, err := parseFloat(event.Price)
			if err != nil {
				log.Printf("Error parsing price: %v", err)
				continue
			}

			quantity, err := parseFloat(event.Quantity)
			if err != nil {
				log.Printf("Error parsing quantity: %v", err)
				continue
			}

			trade := domain.Trade{
				Symbol:   event.Symbol,
				ID:       event.TradeID,
				Price:    price,
				Quantity: quantity,
				Time:     time.Unix(0, event.TradeTime*int64(time.Millisecond)),
				IsBuyer:  !event.IsBuyerMaker,
				IsMaker:  event.IsBuyerMaker,
			}

			task := domain.NewTradeTask(trade, b.repo)
			if err := b.pool.Submit(task); err != nil {
				log.Printf("Error submitting task: %v", err)
				continue
			}
		}
	}
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	if err != nil {
		return 0, fmt.Errorf("failed to parse float: %w", err)
	}
	return f, nil
}
