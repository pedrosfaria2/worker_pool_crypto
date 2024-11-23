package domain

import (
	"context"
)

type TradeTask struct {
	BaseTask
	trade Trade
}

func NewTradeTask(trade Trade) *TradeTask {
	return &TradeTask{
		BaseTask: NewBaseTask("trade_processing"),
		trade:    trade,
	}
}

func (t *TradeTask) Process(ctx context.Context) error {
	t.SetStatus(TaskRunning)

	// Not implemented yet

	t.SetStatus(TaskCompleted)
	return nil
}

func (t *TradeTask) Trade() Trade {
	return t.trade
}
