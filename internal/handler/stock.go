package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sunny/technical-analysis/internal/service"
	"github.com/sunny/technical-analysis/pkg/indicator"
)

// StockHandler handles stock and price HTTP endpoints.
type StockHandler struct {
	svc *service.StockService
}

// NewStockHandler creates a StockHandler.
func NewStockHandler(svc *service.StockService) *StockHandler {
	return &StockHandler{svc: svc}
}

// ListStocks handles GET /api/v1/stocks
func (h *StockHandler) ListStocks(c *gin.Context) {
	stocks, err := h.svc.ListStocks(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to list stocks")
		return
	}
	resp := make([]StockResponse, len(stocks))
	for i, s := range stocks {
		resp[i] = StockResponse{Symbol: s.Symbol, Name: s.Name, Market: s.Market}
	}
	respondOK(c, resp)
}

// GetStock handles GET /api/v1/stocks/:symbol
func (h *StockHandler) GetStock(c *gin.Context) {
	symbol := c.Param("symbol")
	stock, err := h.svc.GetStock(c.Request.Context(), symbol)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}
	respondOK(c, StockResponse{Symbol: stock.Symbol, Name: stock.Name, Market: stock.Market})
}

// GetPrices handles GET /api/v1/stocks/:symbol/prices
func (h *StockHandler) GetPrices(c *gin.Context) {
	symbol := c.Param("symbol")
	from, to := parseDateRange(c)

	prices, err := h.svc.GetPrices(c.Request.Context(), symbol, from, to)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(c, toPriceResponses(prices))
}

func parseDateRange(c *gin.Context) (time.Time, time.Time) {
	const layout = "2006-01-02"
	from, _ := time.Parse(layout, c.Query("from"))
	to, _ := time.Parse(layout, c.Query("to"))
	return from, to
}

func toPriceResponses(prices []indicator.Price) []PriceResponse {
	resp := make([]PriceResponse, len(prices))
	for i, p := range prices {
		resp[i] = PriceResponse{
			Date:   p.Date.Format("2006-01-02"),
			Open:   p.Open,
			High:   p.High,
			Low:    p.Low,
			Close:  p.Close,
			Volume: p.Volume,
		}
	}
	return resp
}
