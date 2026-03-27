package indicator_test

import (
	"testing"
	"time"

	"github.com/sunny/technical-analysis/pkg/indicator"
)

func makePrices(closes []float64) []indicator.Price {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := make([]indicator.Price, len(closes))
	for i, c := range closes {
		prices[i] = indicator.Price{Date: base.AddDate(0, 0, i), Close: c}
	}
	return prices
}

func TestMA(t *testing.T) {
	prices := makePrices([]float64{10, 11, 12, 13, 14, 15})
	result := indicator.MA(prices, 5)
	if len(result) != 2 {
		t.Fatalf("expected 2 data points, got %d", len(result))
	}
	if result[0].Value != 12.0 {
		t.Errorf("expected 12.0, got %v", result[0].Value)
	}
	if result[1].Value != 13.0 {
		t.Errorf("expected 13.0, got %v", result[1].Value)
	}
}

func TestMA_InsufficientData(t *testing.T) {
	result := indicator.MA(makePrices([]float64{10, 11}), 5)
	if result != nil {
		t.Errorf("expected nil for insufficient data")
	}
}

func TestEMA(t *testing.T) {
	// k = 2/(3+1) = 0.5
	// EMA[0]=10, EMA[1]=10.5, EMA[2]=11.25, EMA[3]=12.125, EMA[4]=13.0625
	prices := makePrices([]float64{10, 11, 12, 13, 14})
	result := indicator.EMA(prices, 3)
	if len(result) != 5 {
		t.Fatalf("expected 5 data points, got %d", len(result))
	}
	if result[0].Value != 10.0 {
		t.Errorf("EMA[0]: expected 10.0, got %v", result[0].Value)
	}
	if result[1].Value != 10.5 {
		t.Errorf("EMA[1]: expected 10.5, got %v", result[1].Value)
	}
	if result[4].Value != 13.0625 {
		t.Errorf("EMA[4]: expected 13.0625, got %v", result[4].Value)
	}
}

func TestEMA_Empty(t *testing.T) {
	if indicator.EMA([]indicator.Price{}, 3) != nil {
		t.Errorf("expected nil for empty prices")
	}
}
