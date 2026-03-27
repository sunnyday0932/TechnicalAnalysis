package syncer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const tpexDefaultURL = "https://www.tpex.org.tw/openapi/v1/tpex_mainboard_daily_close_quotes"

// TPExFetcher fetches OTC stock data from TPEx Open API.
type TPExFetcher struct {
	url    string
	client *http.Client
}

// NewTPExFetcher creates a TPExFetcher. Pass empty baseURL to use the default.
func NewTPExFetcher(baseURL string) *TPExFetcher {
	if baseURL == "" {
		baseURL = tpexDefaultURL
	}
	return &TPExFetcher{
		url:    baseURL,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

type tpexRecord struct {
	Date   string `json:"Date"`   // e.g. "114/03/26" (ROC year)
	Code   string `json:"SecuritiesCompanyCode"`
	Name   string `json:"CompanyName"`
	Open   string `json:"Open"`
	High   string `json:"High"`
	Low    string `json:"Low"`
	Close  string `json:"Close"`
	Volume string `json:"Volumn"` // Note: TPEx API typo — "Volumn" not "Volume"
}

// FetchAll fetches all OTC stocks' latest daily data from TPEx.
func (f *TPExFetcher) FetchAll() ([]StockRecord, error) {
	resp, err := f.client.Get(f.url)
	if err != nil {
		return nil, fmt.Errorf("tpex fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tpex read: %w", err)
	}

	var raw []tpexRecord
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("tpex parse: %w", err)
	}

	records := make([]StockRecord, 0, len(raw))
	for _, r := range raw {
		date, err := parseROCDate(r.Date)
		if err != nil {
			continue
		}
		open, _ := strconv.ParseFloat(r.Open, 64)
		high, _ := strconv.ParseFloat(r.High, 64)
		low, _ := strconv.ParseFloat(r.Low, 64)
		close, _ := strconv.ParseFloat(r.Close, 64)
		volStr := strings.ReplaceAll(r.Volume, ",", "")
		vol, _ := strconv.ParseInt(volStr, 10, 64)

		records = append(records, StockRecord{
			Symbol: r.Code,
			Name:   r.Name,
			Market: "TPEx",
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: vol,
		})
	}
	return records, nil
}

// parseROCDate converts "114/03/26" (ROC year) to time.Time.
func parseROCDate(s string) (time.Time, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid ROC date: %s", s)
	}
	rocYear, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, err
	}
	western := fmt.Sprintf("%d/%s/%s", rocYear+1911, parts[1], parts[2])
	return time.Parse("2006/01/02", western)
}
