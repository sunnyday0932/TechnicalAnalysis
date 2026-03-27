package indicator_test

import (
	"math"
	"testing"

	"github.com/sunny/technical-analysis/pkg/indicator"
)

func TestBollingerBands_Length(t *testing.T) {
	result := indicator.BollingerBands(makePrices([]float64{10, 11, 12, 13, 14, 15}), 3)
	if len(result.Mid) != 4 || len(result.Upper) != 4 || len(result.Lower) != 4 {
		t.Errorf("expected 4 values each, got Mid=%d Upper=%d Lower=%d",
			len(result.Mid), len(result.Upper), len(result.Lower))
	}
}

func TestBollingerBands_FirstValues(t *testing.T) {
	// [10,11,12], period=3: Mid=11.0, std=sqrt(2/3)
	result := indicator.BollingerBands(makePrices([]float64{10, 11, 12, 13}), 3)
	wantMid := 11.0
	wantStd := math.Sqrt(2.0 / 3.0)
	if absFloat(result.Mid[0].Value-wantMid) > 1e-9 {
		t.Errorf("Mid: expected %v, got %v", wantMid, result.Mid[0].Value)
	}
	if absFloat(result.Upper[0].Value-(wantMid+2*wantStd)) > 1e-9 {
		t.Errorf("Upper: expected %v, got %v", wantMid+2*wantStd, result.Upper[0].Value)
	}
	if absFloat(result.Lower[0].Value-(wantMid-2*wantStd)) > 1e-9 {
		t.Errorf("Lower: expected %v, got %v", wantMid-2*wantStd, result.Lower[0].Value)
	}
}

func TestBollingerBands_Symmetry(t *testing.T) {
	result := indicator.BollingerBands(makePrices([]float64{10, 11, 12, 13, 14}), 3)
	for i := range result.Mid {
		upperDiff := result.Upper[i].Value - result.Mid[i].Value
		lowerDiff := result.Mid[i].Value - result.Lower[i].Value
		if absFloat(upperDiff-lowerDiff) > 1e-9 {
			t.Errorf("band[%d] not symmetric", i)
		}
	}
}
