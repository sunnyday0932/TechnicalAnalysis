package syncer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sunny/technical-analysis/internal/repository"
)

// Syncer orchestrates TWSE + TPEx data fetching and DB upserts.
type Syncer struct {
	q      repository.Querier
	twse   *TWSEFetcher
	tpex   *TPExFetcher
	appCtx context.Context // application-level context for async retry
}

// NewSyncer creates a Syncer using the default API URLs.
func NewSyncer(ctx context.Context, q repository.Querier) *Syncer {
	return &Syncer{
		q:      q,
		twse:   NewTWSEFetcher(""),
		tpex:   NewTPExFetcher(""),
		appCtx: ctx,
	}
}

// SyncAll fetches all stocks from TWSE and TPEx and upserts into DB.
// Writes a sync_log entry for the operation.
func (s *Syncer) SyncAll(ctx context.Context, triggered string) error {
	logEntry, err := s.q.CreateSyncLog(ctx, repository.CreateSyncLogParams{
		Triggered: triggered,
		Status:    "running",
	})
	if err != nil {
		return fmt.Errorf("create sync log: %w", err)
	}

	syncErr := s.doSync(ctx)

	status, msg := "success", ""
	if syncErr != nil {
		status = "failed"
		msg = syncErr.Error()
	}
	_ = s.q.UpdateSyncLog(ctx, repository.UpdateSyncLogParams{
		ID:      logEntry.ID,
		Status:  status,
		Message: msg,
	})
	return syncErr
}

func (s *Syncer) doSync(ctx context.Context) error {
	twseRecords, err := s.twse.FetchAll()
	if err != nil {
		return fmt.Errorf("twse: %w", err)
	}
	tpexRecords, err := s.tpex.FetchAll()
	if err != nil {
		return fmt.Errorf("tpex: %w", err)
	}

	all := append(twseRecords, tpexRecords...)
	for _, r := range all {
		if err := s.q.UpsertStock(ctx, repository.UpsertStockParams{
			Symbol: r.Symbol,
			Name:   r.Name,
			Market: r.Market,
		}); err != nil {
			log.Printf("upsert stock %s: %v", r.Symbol, err)
			continue
		}
		if err := s.q.UpsertDailyPrice(ctx, repository.UpsertDailyPriceParams{
			Symbol: r.Symbol,
			Date:   r.Date,
			Open:   pgtype.Float8{Float64: r.Open, Valid: true},
			High:   pgtype.Float8{Float64: r.High, Valid: true},
			Low:    pgtype.Float8{Float64: r.Low, Valid: true},
			Close:  pgtype.Float8{Float64: r.Close, Valid: true},
			Volume: pgtype.Int8{Int64: r.Volume, Valid: true},
		}); err != nil {
			log.Printf("upsert price %s %s: %v", r.Symbol, r.Date.Format("2006-01-02"), err)
		}
	}
	return nil
}

// SyncAllWithRetry runs SyncAll and retries once after 5 minutes on failure.
// Uses the application context (not the caller's context) for the retry.
func (s *Syncer) SyncAllWithRetry(triggered string) {
	if err := s.SyncAll(s.appCtx, triggered); err != nil {
		log.Printf("sync failed (%s): %v — retrying in 5 minutes", triggered, err)
		time.AfterFunc(5*time.Minute, func() {
			if err := s.SyncAll(s.appCtx, triggered+"-retry"); err != nil {
				log.Printf("sync retry failed: %v", err)
			}
		})
	}
}
