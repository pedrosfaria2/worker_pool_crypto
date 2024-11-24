package domain

import "time"

type Trade struct {
	Symbol   string
	ID       int64
	Price    float64
	Quantity float64
	Time     time.Time
	IsBuyer  bool
	IsMaker  bool
}

type TradeEvent struct {
	EventType string
	Time      int64
	Symbol    string
	TradeID   int64
	Price     string
	Quantity  string
	BuyerID   int64
	SellerID  int64
	TradeTime int64
	IsBuyer   bool
	Ignored   bool
}
