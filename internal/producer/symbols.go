package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type exchangeInfo struct {
	Symbols []struct {
		Symbol string `json:"symbol"`
		Status string `json:"status"`
	} `json:"symbols"`
}

func FetchPairs(ctx context.Context) ([]string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.binance.com/api/v3/exchangeInfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch exchange info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var info exchangeInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("failed to parse exchange info: %w", err)
	}

	var pairs []string
	for _, symbol := range info.Symbols {
		if (strings.HasSuffix(symbol.Symbol, "USDT") ||
			strings.HasSuffix(symbol.Symbol, "BRL") ||
			strings.HasSuffix(symbol.Symbol, "BTC") ||
			strings.HasSuffix(symbol.Symbol, "USDC") ||
			strings.HasSuffix(symbol.Symbol, "ETH")) &&
			symbol.Status == "TRADING" {
			pairs = append(pairs, strings.ToLower(symbol.Symbol))
		}
	}

	if len(pairs) == 0 {
		return nil, fmt.Errorf("no trading pairs found for USDT or BRL")
	}

	return pairs, nil
}
