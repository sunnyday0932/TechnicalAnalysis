package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse is the unified JSON error body.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

func respondError(c *gin.Context, status int, msg string) {
	c.JSON(status, ErrorResponse{Error: msg, Code: status})
}

// StockResponse is the JSON body for a single stock.
type StockResponse struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Market string `json:"market"`
}

// PriceResponse is the JSON body for one OHLCV day.
type PriceResponse struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int64   `json:"volume"`
}

// DataPointResponse is a single time-series value.
type DataPointResponse struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

// MACDDataResponse holds three MACD series.
type MACDDataResponse struct {
	DIF       []DataPointResponse `json:"dif"`
	Signal    []DataPointResponse `json:"signal"`
	Histogram []DataPointResponse `json:"histogram"`
}

// KDDataResponse holds K and D series.
type KDDataResponse struct {
	K []DataPointResponse `json:"k"`
	D []DataPointResponse `json:"d"`
}

// BBDataResponse holds Bollinger Bands series.
type BBDataResponse struct {
	Upper []DataPointResponse `json:"upper"`
	Mid   []DataPointResponse `json:"mid"`
	Lower []DataPointResponse `json:"lower"`
}

// IndicatorResponse is the JSON body for indicator results.
type IndicatorResponse struct {
	Symbol    string `json:"symbol"`
	Name      string `json:"name"`
	Indicator string `json:"indicator"`
	Period    int    `json:"period,omitempty"`
	Data      any    `json:"data"`
}

func respondOK(c *gin.Context, body any) {
	c.JSON(http.StatusOK, body)
}
