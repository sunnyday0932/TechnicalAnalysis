package indicator_test

import (
	"testing"
	"time"

	"github.com/sunny/technical-analysis/pkg/indicator"
)

func makeOHLCPrices(data [][4]float64) []indicator.Price {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := make([]indicator.Price, len(data))
	for i, d := range data {
		prices[i] = indicator.Price{Date: base.AddDate(0, 0, i), Open: d[0], High: d[1], Low: d[2], Close: d[3]}
	}
	return prices
}

func TestKD_InsufficientData(t *testing.T) {
	prices := makeOHLCPrices([][4]float64{{10, 12, 9, 11}, {11, 13, 10, 12}})
	result := indicator.KD(prices, 9)
	if len(result.K) != 0 || len(result.D) != 0 {
		t.Errorf("expected empty results for insufficient data")
	}
}

func TestKD_FirstValue(t *testing.T) {
	// period=3, init K=D=50
	// i=2: highest=14, lowest=10, RSV=(13-10)/(14-10)*100=75
	// K = 50*2/3 + 75*1/3 = 58.333...
	// D = 50*2/3 + K*1/3
	data := [][4]float64{
		{10, 12, 10, 11},
		{11, 13, 11, 12},
		{12, 14, 12, 13},
	}
	result := indicator.KD(makeOHLCPrices(data), 3)
	if len(result.K) != 1 {
		t.Fatalf("expected 1 K value, got %d", len(result.K))
	}
	wantK := 50.0*2/3 + 75.0*1/3
	if absFloat(result.K[0].Value-wantK) > 0.01 {
		t.Errorf("K: expected %.4f, got %.4f", wantK, result.K[0].Value)
	}
	wantD := 50.0*2/3 + wantK*1/3
	if absFloat(result.D[0].Value-wantD) > 0.01 {
		t.Errorf("D: expected %.4f, got %.4f", wantD, result.D[0].Value)
	}
}

func TestKD_Length(t *testing.T) {
	data := make([][4]float64, 20)
	for i := range data {
		f := float64(i + 10)
		data[i] = [4]float64{f, f + 1, f - 1, f}
	}
	result := indicator.KD(makeOHLCPrices(data), 9)
	if len(result.K) != 12 {
		t.Errorf("expected 12 K values, got %d", len(result.K))
	}
}
