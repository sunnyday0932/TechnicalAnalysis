package syncer_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunny/technical-analysis/internal/syncer"
)

func TestTPExFetcher_ParsesResponse(t *testing.T) {
	// TPEx uses ROC year (e.g. 114 = 2025) and "Volumn" (their typo)
	sample := []map[string]string{
		{
			"Date":                  "114/03/26",
			"SecuritiesCompanyCode": "6505",
			"CompanyName":           "台塑化",
			"Open":                  "80.00",
			"High":                  "81.00",
			"Low":                   "79.50",
			"Close":                 "80.50",
			"Volumn":                "1234567",
		},
	}
	body, _ := json.Marshal(sample)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer server.Close()

	fetcher := syncer.NewTPExFetcher(server.URL)
	records, err := fetcher.FetchAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	r := records[0]
	if r.Symbol != "6505" {
		t.Errorf("Symbol: expected 6505, got %s", r.Symbol)
	}
	if r.Market != "TPEx" {
		t.Errorf("Market: expected TPEx, got %s", r.Market)
	}
	if r.Close != 80.50 {
		t.Errorf("Close: expected 80.50, got %v", r.Close)
	}
	if r.Volume != 1234567 {
		t.Errorf("Volume: expected 1234567, got %v", r.Volume)
	}
	if r.Date.Year() != 2025 || r.Date.Month() != 3 || r.Date.Day() != 26 {
		t.Errorf("Date: expected 2025-03-26, got %v", r.Date)
	}
}
