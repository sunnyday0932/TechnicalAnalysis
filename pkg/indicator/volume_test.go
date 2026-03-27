package indicator_test

import (
	"testing"
	"time"

	"github.com/sunny/technical-analysis/pkg/indicator"
)

func TestVolume(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := []indicator.Price{
		{Date: base, Volume: 1000},
		{Date: base.AddDate(0, 0, 1), Volume: 2000},
		{Date: base.AddDate(0, 0, 2), Volume: 3000},
	}
	result := indicator.Volume(prices)
	if len(result) != 3 {
		t.Fatalf("expected 3 data points, got %d", len(result))
	}
	if result[0].Value != 1000 {
		t.Errorf("expected 1000, got %v", result[0].Value)
	}
	if result[2].Value != 3000 {
		t.Errorf("expected 3000, got %v", result[2].Value)
	}
}

func TestVolume_Empty(t *testing.T) {
	if len(indicator.Volume([]indicator.Price{})) != 0 {
		t.Errorf("expected empty result for empty input")
	}
}
