package indicator_test

import (
	"testing"

	"github.com/sunny/technical-analysis/pkg/indicator"
)

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func TestMACD_Length(t *testing.T) {
	closes := make([]float64, 30)
	for i := range closes {
		closes[i] = float64(i + 10)
	}
	result := indicator.MACD(makePrices(closes))
	if len(result.DIF) != 30 {
		t.Errorf("DIF: expected 30, got %d", len(result.DIF))
	}
	if len(result.Signal) != 30 {
		t.Errorf("Signal: expected 30, got %d", len(result.Signal))
	}
	if len(result.Histogram) != 30 {
		t.Errorf("Histogram: expected 30, got %d", len(result.Histogram))
	}
}

func TestMACD_HistogramEquality(t *testing.T) {
	closes := make([]float64, 30)
	for i := range closes {
		closes[i] = float64(i + 10)
	}
	result := indicator.MACD(makePrices(closes))
	for i := range result.Histogram {
		expected := result.DIF[i].Value - result.Signal[i].Value
		if absFloat(result.Histogram[i].Value-expected) > 1e-9 {
			t.Errorf("Histogram[%d]: expected %v, got %v", i, expected, result.Histogram[i].Value)
		}
	}
}
