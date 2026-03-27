package scheduler

import (
	"log"

	"github.com/robfig/cron/v3"
	"github.com/sunny/technical-analysis/internal/syncer"
)

// Scheduler wraps robfig/cron for daily stock data sync.
type Scheduler struct {
	cron   *cron.Cron
	syncer *syncer.Syncer
}

// New creates a Scheduler. Call Start() to begin.
func New(s *syncer.Syncer) *Scheduler {
	return &Scheduler{
		cron:   cron.New(),
		syncer: s,
	}
}

// Start registers the daily sync job and starts the scheduler.
// Runs every weekday at 18:30 (台股收盤後).
func (s *Scheduler) Start() {
	_, err := s.cron.AddFunc("30 18 * * 1-5", func() {
		log.Println("scheduler: starting daily sync")
		s.syncer.SyncAllWithRetry("auto")
	})
	if err != nil {
		log.Fatalf("scheduler: failed to register cron job: %v", err)
	}
	s.cron.Start()
	log.Println("scheduler: started — daily sync at 18:30 weekdays")
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() {
	s.cron.Stop()
}
