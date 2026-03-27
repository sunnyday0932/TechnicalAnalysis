package service

import (
	"context"
	"fmt"
	"time"

	"github.com/sunny/technical-analysis/pkg/indicator"
)

// IndicatorResult is the unified return type for any indicator computation.
type IndicatorResult struct {
	Symbol    string
	Name      string
	Indicator string
	Period    int
	Data      any // []indicator.DataPoint | indicator.MACDResult | indicator.KDResult | indicator.BBResult
}

// IndicatorService computes technical indicators.
type IndicatorService struct {
	stock *StockService
}

// NewIndicatorService creates an IndicatorService.
func NewIndicatorService(stock *StockService) *IndicatorService {
	return &IndicatorService{stock: stock}
}

// Compute fetches prices for symbol and computes the requested indicator.
// indicatorType: ma, ema, rsi, macd, kd, bb, volume
// period: used for ma, ema, rsi, kd, bb (ignored for macd, volume)
func (s *IndicatorService) Compute(ctx context.Context, symbol, indicatorType string, period int) (IndicatorResult, error) {
	stock, err := s.stock.GetStock(ctx, symbol)
	if err != nil {
		return IndicatorResult{}, err
	}

	prices, err := s.stock.GetPrices(ctx, symbol, time.Time{}, time.Time{})
	if err != nil {
		return IndicatorResult{}, fmt.Errorf("failed to fetch prices for %s: %w", symbol, err)
	}
	if len(prices) == 0 {
		return IndicatorResult{}, fmt.Errorf("no price data for symbol %q", symbol)
	}

	var data any
	switch indicatorType {
	case "ma":
		data = indicator.MA(prices, period)
	case "ema":
		data = indicator.EMA(prices, period)
	case "rsi":
		data = indicator.RSI(prices, period)
	case "macd":
		data = indicator.MACD(prices)
	case "kd":
		data = indicator.KD(prices, period)
	case "bb":
		data = indicator.BollingerBands(prices, period)
	case "volume":
		data = indicator.Volume(prices)
	default:
		return IndicatorResult{}, fmt.Errorf("unknown indicator type %q", indicatorType)
	}

	return IndicatorResult{
		Symbol:    stock.Symbol,
		Name:      stock.Name,
		Indicator: indicatorType,
		Period:    period,
		Data:      data,
	}, nil
}
