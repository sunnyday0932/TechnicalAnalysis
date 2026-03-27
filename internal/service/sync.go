package service

import (
	"context"
	"fmt"
	"time"

	"github.com/sunny/technical-analysis/internal/repository"
	"github.com/sunny/technical-analysis/internal/syncer"
)

// SyncService exposes sync operations to handlers.
type SyncService struct {
	q      repository.Querier
	syncer *syncer.Syncer
}

// NewSyncService creates a SyncService.
func NewSyncService(q repository.Querier, s *syncer.Syncer) *SyncService {
	return &SyncService{q: q, syncer: s}
}

// TriggerFullSync starts a full sync in the background (non-blocking).
func (s *SyncService) TriggerFullSync() {
	go s.syncer.SyncAllWithRetry("manual")
}

// TriggerSymbolSync fetches only one stock's data. Returns an error if the symbol is unknown.
func (s *SyncService) TriggerSymbolSync(ctx context.Context, symbol string) error {
	_, err := s.q.GetStock(ctx, symbol)
	if err != nil {
		return fmt.Errorf("symbol %q not found", symbol)
	}
	go s.syncer.SyncAll(ctx, "manual-"+symbol)
	return nil
}

// SyncStatus is the JSON-serialisable sync status.
type SyncStatus struct {
	ID         int64      `json:"id"`
	Triggered  string     `json:"triggered"`
	Status     string     `json:"status"`
	Message    string     `json:"message"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

// GetStatus returns the last sync log entry.
func (s *SyncService) GetStatus(ctx context.Context) (SyncStatus, error) {
	row, err := s.q.GetLastSyncLog(ctx)
	if err != nil {
		return SyncStatus{}, fmt.Errorf("no sync log found")
	}

	// finished_at is nullable TIMESTAMPTZ — sqlc generates pgtype.Timestamptz
	var finishedAt *time.Time
	if row.FinishedAt.Valid {
		t := row.FinishedAt.Time
		finishedAt = &t
	}

	return SyncStatus{
		ID:         row.ID,
		Triggered:  row.Triggered,
		Status:     row.Status,
		Message:    row.Message,
		StartedAt:  row.StartedAt,
		FinishedAt: finishedAt,
	}, nil
}
