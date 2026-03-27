package indicator_test

import (
	"testing"

	"github.com/sunny/technical-analysis/pkg/indicator"
)

func TestRSI_InsufficientData(t *testing.T) {
	if indicator.RSI(makePrices([]float64{10, 11, 12}), 14) != nil {
		t.Errorf("expected nil for insufficient data")
	}
}

func TestRSI_AllGains(t *testing.T) {
	closes := make([]float64, 15)
	for i := range closes {
		closes[i] = float64(i + 10)
	}
	result := indicator.RSI(makePrices(closes), 14)
	if len(result) == 0 {
		t.Fatal("expected at least one result")
	}
	if result[0].Value != 100.0 {
		t.Errorf("all-gain RSI: expected 100.0, got %v", result[0].Value)
	}
}

func TestRSI_AllLosses(t *testing.T) {
	closes := make([]float64, 15)
	for i := range closes {
		closes[i] = float64(24 - i)
	}
	result := indicator.RSI(makePrices(closes), 14)
	if len(result) == 0 {
		t.Fatal("expected at least one result")
	}
	if result[0].Value != 0.0 {
		t.Errorf("all-loss RSI: expected 0.0, got %v", result[0].Value)
	}
}

func TestRSI_Length(t *testing.T) {
	closes := make([]float64, 20)
	for i := range closes {
		closes[i] = float64(i + 1)
	}
	result := indicator.RSI(makePrices(closes), 14)
	if len(result) != 6 {
		t.Errorf("expected 6 results, got %d", len(result))
	}
}
