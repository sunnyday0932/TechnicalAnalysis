package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sunny/technical-analysis/internal/service"
	"github.com/sunny/technical-analysis/pkg/indicator"
)

// IndicatorHandler handles indicator HTTP endpoints.
type IndicatorHandler struct {
	svc *service.IndicatorService
}

// NewIndicatorHandler creates an IndicatorHandler.
func NewIndicatorHandler(svc *service.IndicatorService) *IndicatorHandler {
	return &IndicatorHandler{svc: svc}
}

// defaultPeriods maps indicator type to its standard default period.
var defaultPeriods = map[string]int{
	"ma": 20, "ema": 12, "rsi": 14, "kd": 9, "bb": 20,
}

// GetIndicator handles GET /api/v1/stocks/:symbol/indicators
func (h *IndicatorHandler) GetIndicator(c *gin.Context) {
	symbol := c.Param("symbol")
	indicatorType := c.Query("type")
	if indicatorType == "" {
		respondError(c, http.StatusBadRequest, "query param 'type' is required")
		return
	}

	period := defaultPeriods[indicatorType]
	if p := c.Query("period"); p != "" {
		parsed, err := strconv.Atoi(p)
		if err != nil || parsed <= 0 {
			respondError(c, http.StatusBadRequest, "invalid period value")
			return
		}
		period = parsed
	}

	result, err := h.svc.Compute(c.Request.Context(), symbol, indicatorType, period)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	respondOK(c, IndicatorResponse{
		Symbol:    result.Symbol,
		Name:      result.Name,
		Indicator: result.Indicator,
		Period:    result.Period,
		Data:      toIndicatorData(indicatorType, result.Data),
	})
}

// toIndicatorData converts indicator results to JSON-serialisable response types.
func toIndicatorData(indicatorType string, data any) any {
	switch indicatorType {
	case "macd":
		r := data.(indicator.MACDResult)
		return MACDDataResponse{
			DIF:       toDataPointResponses(r.DIF),
			Signal:    toDataPointResponses(r.Signal),
			Histogram: toDataPointResponses(r.Histogram),
		}
	case "kd":
		r := data.(indicator.KDResult)
		return KDDataResponse{
			K: toDataPointResponses(r.K),
			D: toDataPointResponses(r.D),
		}
	case "bb":
		r := data.(indicator.BBResult)
		return BBDataResponse{
			Upper: toDataPointResponses(r.Upper),
			Mid:   toDataPointResponses(r.Mid),
			Lower: toDataPointResponses(r.Lower),
		}
	default:
		return toDataPointResponses(data.([]indicator.DataPoint))
	}
}

func toDataPointResponses(pts []indicator.DataPoint) []DataPointResponse {
	resp := make([]DataPointResponse, len(pts))
	for i, p := range pts {
		resp[i] = DataPointResponse{Date: p.Date.Format("2006-01-02"), Value: p.Value}
	}
	return resp
}
