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

const twseDefaultURL = "https://openapi.twse.com.tw/v1/exchangeReport/STOCK_DAY_ALL"

// StockRecord is the parsed result from either TWSE or TPEx.
type StockRecord struct {
	Symbol string
	Name   string
	Market string
	Date   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

// TWSEFetcher fetches listed-stock data from TWSE Open API.
type TWSEFetcher struct {
	url    string
	client *http.Client
}

// NewTWSEFetcher creates a TWSEFetcher. Pass an empty baseURL to use the default.
func NewTWSEFetcher(baseURL string) *TWSEFetcher {
	if baseURL == "" {
		baseURL = twseDefaultURL
	}
	return &TWSEFetcher{
		url:    baseURL,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

type twseRecord struct {
	Date         string `json:"Date"`
	Code         string `json:"Code"`
	Name         string `json:"Name"`
	TradeVolume  string `json:"TradeVolume"`
	OpeningPrice string `json:"OpeningPrice"`
	HighestPrice string `json:"HighestPrice"`
	LowestPrice  string `json:"LowestPrice"`
	ClosingPrice string `json:"ClosingPrice"`
}

// parseTWSEDate converts "YYYMMDD" (ROC year, e.g. "1150326") to time.Time.
func parseTWSEDate(s string) (time.Time, error) {
	if len(s) != 7 {
		return time.Time{}, fmt.Errorf("invalid TWSE date: %s", s)
	}
	rocYear, err := strconv.Atoi(s[:3])
	if err != nil {
		return time.Time{}, err
	}
	western := fmt.Sprintf("%d%s", rocYear+1911, s[3:])
	return time.Parse("20060102", western)
}

// FetchAll fetches all listed stocks' latest daily data from TWSE.
func (f *TWSEFetcher) FetchAll() ([]StockRecord, error) {
	resp, err := f.client.Get(f.url)
	if err != nil {
		return nil, fmt.Errorf("twse fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("twse read: %w", err)
	}

	var raw []twseRecord
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("twse parse: %w", err)
	}

	records := make([]StockRecord, 0, len(raw))
	for _, r := range raw {
		date, err := parseTWSEDate(r.Date)
		if err != nil {
			continue
		}
		open, _ := strconv.ParseFloat(r.OpeningPrice, 64)
		high, _ := strconv.ParseFloat(r.HighestPrice, 64)
		low, _ := strconv.ParseFloat(r.LowestPrice, 64)
		close, _ := strconv.ParseFloat(r.ClosingPrice, 64)
		volStr := strings.ReplaceAll(r.TradeVolume, ",", "")
		vol, _ := strconv.ParseInt(volStr, 10, 64)

		records = append(records, StockRecord{
			Symbol: r.Code,
			Name:   r.Name,
			Market: "TWSE",
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
