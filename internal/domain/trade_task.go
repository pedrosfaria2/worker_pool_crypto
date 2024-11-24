package domain

import (
	"context"
	"fmt"
	"log"
	"time"
)

type TradeMetrics struct {
	Symbol      string    `json:"symbol"`
	Price       float64   `json:"price"`
	Open24h     float64   `json:"open_24h"`
	High24h     float64   `json:"high_24h"`
	Low24h      float64   `json:"low_24h"`
	Volume24h   float64   `json:"volume_24h"`
	QuoteVolume float64   `json:"quote_volume"`
	Count       int64     `json:"count"`
	BidPrice    float64   `json:"bid_price"`
	AskPrice    float64   `json:"ask_price"`
	Change24h   float64   `json:"change_24h"`
	Change24hP  float64   `json:"change_24h_percent"`
	VWAP        float64   `json:"vwap"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TradeTask struct {
	BaseTask
	trade Trade
	repo  TradeRepository
}

func NewTradeTask(trade Trade, repo TradeRepository) *TradeTask {
	return &TradeTask{
		BaseTask: NewBaseTask("trade_processing"),
		trade:    trade,
		repo:     repo,
	}
}

func (t *TradeTask) Process(ctx context.Context) error {
	t.SetStatus(TaskRunning)
	log.Printf("[TRADE] Processing %s: Price %.2f, Quantity %.8f", t.trade.Symbol, t.trade.Price, t.trade.Quantity)

	if err := t.repo.Save(ctx, t.trade); err != nil {
		t.SetStatus(TaskFailed)
		return fmt.Errorf("failed to save trade: %w", err)
	}

	trades, err := t.repo.FindBySymbol(ctx, t.trade.Symbol, 2000)
	if err != nil {
		t.SetStatus(TaskFailed)
		return fmt.Errorf("failed to fetch historical trades: %w", err)
	}

	log.Printf("[TRADE] Found %d historical trades", len(trades))
	metrics := t.calculateMetrics(trades)

	if err := t.repo.SaveMetrics(ctx, metrics); err != nil {
		t.SetStatus(TaskFailed)
		return fmt.Errorf("failed to save metrics: %w", err)
	}

	t.SetStatus(TaskCompleted)
	return nil
}

func (t *TradeTask) calculateMetrics(trades []Trade) *TradeMetrics {
	metrics := &TradeMetrics{
		Symbol:    t.trade.Symbol,
		Price:     t.trade.Price,
		UpdatedAt: time.Now(),
	}

	if len(trades) == 0 {
		return metrics
	}

	metrics.High24h = trades[0].Price
	metrics.Low24h = trades[0].Price
	metrics.Open24h = trades[len(trades)-1].Price
	metrics.Count = int64(len(trades))

	var sumVolume, sumQuoteVolume float64

	for _, trade := range trades {
		sumVolume += trade.Quantity
		sumQuoteVolume += trade.Quantity * trade.Price

		if trade.Price > metrics.High24h {
			metrics.High24h = trade.Price
		}
		if trade.Price < metrics.Low24h {
			metrics.Low24h = trade.Price
		}

		if trade.IsMaker {
			if trade.IsBuyer {
				if metrics.BidPrice == 0 || trade.Price > metrics.BidPrice {
					metrics.BidPrice = trade.Price
				}
			} else {
				if metrics.AskPrice == 0 || trade.Price < metrics.AskPrice {
					metrics.AskPrice = trade.Price
				}
			}
		}
	}

	metrics.VWAP = sumQuoteVolume / sumVolume
	metrics.Change24h = metrics.Price - metrics.Open24h
	metrics.Change24hP = (metrics.Change24h / metrics.Open24h) * 100
	metrics.Volume24h = sumVolume
	metrics.QuoteVolume = sumQuoteVolume

	log.Printf("[METRICS] %s: Price: %.2f Open: %.2f High: %.2f Low: %.2f Volume: %.8f QuoteVol: %.2f Change: %.2f%%",
		metrics.Symbol,
		metrics.Price,
		metrics.Open24h,
		metrics.High24h,
		metrics.Low24h,
		metrics.Volume24h,
		metrics.QuoteVolume,
		metrics.Change24hP,
	)

	return metrics
}

func (t *TradeTask) Trade() Trade {
	return t.trade
}
