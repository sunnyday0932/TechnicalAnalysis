package service

import (
	"context"
	"fmt"
	"time"

	"github.com/sunny/technical-analysis/internal/repository"
	"github.com/sunny/technical-analysis/pkg/indicator"
)

type StockService struct {
	q repository.Querier
}

func NewStockService(q repository.Querier) *StockService {
	return &StockService{q: q}
}

// ListStocks returns all stocks.
func (s *StockService) ListStocks(ctx context.Context) ([]repository.Stock, error) {
	stocks, err := s.q.ListStocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("list stocks: %w", err)
	}
	return stocks, nil
}

// GetStock returns a single stock by symbol.
func (s *StockService) GetStock(ctx context.Context, symbol string) (repository.Stock, error) {
	stock, err := s.q.GetStock(ctx, symbol)
	if err != nil {
		return repository.Stock{}, fmt.Errorf("get stock %s: %w", symbol, err)
	}
	return stock, nil
}

// GetPrices returns OHLCV price data as []indicator.Price.
// from and to are optional (zero time = no filter).
// Rows with NULL Close are skipped (invalid/incomplete data).
func (s *StockService) GetPrices(ctx context.Context, symbol string, from, to time.Time) ([]indicator.Price, error) {
	if !from.IsZero() && !to.IsZero() {
		rows, err := s.q.GetDailyPricesBySymbolAndDateRange(ctx, repository.GetDailyPricesBySymbolAndDateRangeParams{
			Symbol:    symbol,
			StartDate: from,
			EndDate:   to,
		})
		if err != nil {
			return nil, fmt.Errorf("get prices for %s: %w", symbol, err)
		}
		return convertDateRangeRows(rows), nil
	}

	rows, err := s.q.GetDailyPricesBySymbol(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("get prices for %s: %w", symbol, err)
	}
	return convertRows(rows), nil
}

func convertRows(rows []repository.GetDailyPricesBySymbolRow) []indicator.Price {
	prices := make([]indicator.Price, 0, len(rows))
	for _, row := range rows {
		if !row.Close.Valid {
			continue
		}
		var open, high, low float64
		if row.Open.Valid {
			open = row.Open.Float64
		}
		if row.High.Valid {
			high = row.High.Float64
		}
		if row.Low.Valid {
			low = row.Low.Float64
		}
		var volume int64
		if row.Volume.Valid {
			volume = row.Volume.Int64
		}
		prices = append(prices, indicator.Price{
			Date:   row.Date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  row.Close.Float64,
			Volume: volume,
		})
	}
	return prices
}

func convertDateRangeRows(rows []repository.GetDailyPricesBySymbolAndDateRangeRow) []indicator.Price {
	prices := make([]indicator.Price, 0, len(rows))
	for _, row := range rows {
		if !row.Close.Valid {
			continue
		}
		var open, high, low float64
		if row.Open.Valid {
			open = row.Open.Float64
		}
		if row.High.Valid {
			high = row.High.Float64
		}
		if row.Low.Valid {
			low = row.Low.Float64
		}
		var volume int64
		if row.Volume.Valid {
			volume = row.Volume.Int64
		}
		prices = append(prices, indicator.Price{
			Date:   row.Date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  row.Close.Float64,
			Volume: volume,
		})
	}
	return prices
}
