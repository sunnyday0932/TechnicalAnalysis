package syncer_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunny/technical-analysis/internal/syncer"
)

func TestTWSEFetcher_ParsesResponse(t *testing.T) {
	sample := []map[string]string{
		{
			"Date":         "1140326", // ROC 114 = 2025
			"Code":         "2330",
			"Name":         "台積電",
			"TradeVolume":  "23,456,789",
			"OpeningPrice": "1000.0",
			"HighestPrice": "1010.0",
			"LowestPrice":  "995.0",
			"ClosingPrice": "1005.0",
		},
	}
	body, _ := json.Marshal(sample)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer server.Close()

	fetcher := syncer.NewTWSEFetcher(server.URL)
	records, err := fetcher.FetchAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	r := records[0]
	if r.Symbol != "2330" {
		t.Errorf("Symbol: expected 2330, got %s", r.Symbol)
	}
	if r.Name != "台積電" {
		t.Errorf("Name: expected 台積電, got %s", r.Name)
	}
	if r.Close != 1005.0 {
		t.Errorf("Close: expected 1005.0, got %v", r.Close)
	}
	if r.Volume != 23456789 {
		t.Errorf("Volume: expected 23456789, got %v", r.Volume)
	}
}

func TestTWSEFetcher_HandlesEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	fetcher := syncer.NewTWSEFetcher(server.URL)
	records, err := fetcher.FetchAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}
