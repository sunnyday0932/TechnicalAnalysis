package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sunny/technical-analysis/internal/service"
)

// SyncHandler handles sync HTTP endpoints.
type SyncHandler struct {
	svc *service.SyncService
}

// NewSyncHandler creates a SyncHandler.
func NewSyncHandler(svc *service.SyncService) *SyncHandler {
	return &SyncHandler{svc: svc}
}

// TriggerFullSync handles POST /api/v1/sync
func (h *SyncHandler) TriggerFullSync(c *gin.Context) {
	h.svc.TriggerFullSync()
	c.JSON(http.StatusAccepted, gin.H{"message": "sync started"})
}

// TriggerSymbolSync handles POST /api/v1/sync/:symbol
func (h *SyncHandler) TriggerSymbolSync(c *gin.Context) {
	symbol := c.Param("symbol")
	if err := h.svc.TriggerSymbolSync(c.Request.Context(), symbol); err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "sync started for " + symbol})
}

// GetStatus handles GET /api/v1/sync/status
func (h *SyncHandler) GetStatus(c *gin.Context) {
	status, err := h.svc.GetStatus(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}
	respondOK(c, status)
}
