# Taiwan Stock Technical Analysis REST API — Design Spec

**Date:** 2026-03-27
**Status:** Approved

---

## Overview

A Go REST API that fetches Taiwan stock market data (TWSE + TPEx) from free public APIs, stores it in PostgreSQL, and exposes technical indicator calculations to a frontend web application.

---

## Requirements Summary

| Item | Decision |
|------|----------|
| Language | Go |
| Application type | REST API (frontend separate) |
| Data source | TWSE + TPEx free public Open APIs |
| Technical indicators | MA, EMA, RSI, MACD, KD, Bollinger Bands, Volume |
| Database | PostgreSQL (Docker) |
| DB access | sqlc (type-safe query generation) |
| Schema migration | golang-migrate |
| HTTP framework | Gin |
| Scheduler | robfig/cron |
| Data sync | Auto daily (weekdays 18:30) + manual trigger endpoint |
| Retry on sync failure | Retry once after 5 minutes |

---

## Project Structure

```
TechnicalAnalysis/
├── cmd/
│   └── api/
│       └── main.go              ← Entry point: init DB, start server + scheduler
├── internal/
│   ├── handler/                 ← HTTP handlers (parse request, return JSON)
│   ├── service/                 ← Business logic (query stocks, calculate indicators)
│   ├── repository/              ← PostgreSQL data access (sqlc generated)
│   ├── syncer/
│   │   ├── syncer.go            ← Unified sync entry point, coordinates twse + tpex
│   │   ├── twse.go              ← Fetch & parse TWSE listed stocks data
│   │   └── tpex.go              ← Fetch & parse TPEx OTC stocks data
│   └── scheduler/               ← Cron scheduler setup
├── pkg/
│   └── indicator/               ← Pure functions: MA, EMA, RSI, MACD, KD, BB, Volume
├── migrations/                  ← golang-migrate SQL files
│   ├── 000001_create_stocks.up.sql
│   ├── 000001_create_stocks.down.sql
│   ├── 000002_create_daily_prices.up.sql
│   ├── 000002_create_daily_prices.down.sql
│   ├── 000003_create_sync_logs.up.sql
│   └── 000003_create_sync_logs.down.sql
├── db/
│   └── query/                   ← sqlc SQL query files
├── docker-compose.yml           ← PostgreSQL container config
└── .env                         ← DB connection string and env vars
```

---

## Data Flow

```
[TWSE / TPEx Open API]
        ↓  syncer fetches & parses
[PostgreSQL daily_prices]
        ↓  repository queries via sqlc
[service layer computes indicators] ← pkg/indicator pure functions
        ↓
[handler returns JSON response]
        ↓
[Frontend web app]
```

---

## Database Schema

```sql
-- Stock basic information
CREATE TABLE stocks (
    symbol      VARCHAR(10) PRIMARY KEY,   -- e.g. "2330"
    name        VARCHAR(100) NOT NULL,     -- e.g. "台積電"
    market      VARCHAR(10) NOT NULL,      -- "TWSE" or "TPEx"
    created_at  TIMESTAMP DEFAULT NOW()
);

-- Daily OHLCV price data
CREATE TABLE daily_prices (
    id          BIGSERIAL PRIMARY KEY,
    symbol      VARCHAR(10) NOT NULL REFERENCES stocks(symbol),
    date        DATE NOT NULL,
    open        NUMERIC(10,2),
    high        NUMERIC(10,2),
    low         NUMERIC(10,2),
    close       NUMERIC(10,2),
    volume      BIGINT,
    UNIQUE (symbol, date)
);

-- Sync operation log
CREATE TABLE sync_logs (
    id          BIGSERIAL PRIMARY KEY,
    triggered   VARCHAR(10) NOT NULL,      -- "auto" or "manual"
    status      VARCHAR(10) NOT NULL,      -- "running", "success", "failed"
    message     TEXT,
    started_at  TIMESTAMP DEFAULT NOW(),
    finished_at TIMESTAMP
);
```

**Notes:**
- Technical indicators are **not stored** in the database. They are computed on-demand from `daily_prices` by `pkg/indicator` functions.
- `UNIQUE(symbol, date)` enables safe upsert during sync.

---

## API Endpoints

### Stocks

```
GET /api/v1/stocks                      ← List all tracked stocks
GET /api/v1/stocks/:symbol              ← Single stock basic info
```

### Price History

```
GET /api/v1/stocks/:symbol/prices       ← Daily OHLCV data
                                           Query params: ?from=YYYY-MM-DD&to=YYYY-MM-DD
```

### Technical Indicators

```
GET /api/v1/stocks/:symbol/indicators?type=ma&period=20
GET /api/v1/stocks/:symbol/indicators?type=ema&period=12
GET /api/v1/stocks/:symbol/indicators?type=rsi&period=14
GET /api/v1/stocks/:symbol/indicators?type=macd
GET /api/v1/stocks/:symbol/indicators?type=kd
GET /api/v1/stocks/:symbol/indicators?type=bb&period=20
GET /api/v1/stocks/:symbol/indicators?type=volume
```

**Response example (RSI):**
```json
{
  "symbol": "2330",
  "name": "台積電",
  "indicator": "rsi",
  "period": 14,
  "data": [
    { "date": "2025-03-26", "value": 62.3 },
    { "date": "2025-03-25", "value": 58.1 }
  ]
}
```

**Unified error response:**
```json
{ "error": "symbol not found", "code": 404 }
```

### Data Sync

```
POST /api/v1/sync             ← Manually trigger full sync
POST /api/v1/sync/:symbol     ← Manually trigger single stock sync
GET  /api/v1/sync/status      ← Query last sync time and status
```

---

## Technical Indicators — pkg/indicator

All indicator functions are pure (no side effects, no DB dependency).

```go
type Price struct {
    Date                     time.Time
    Open, High, Low, Close   float64
    Volume                   int64
}

type DataPoint struct {
    Date  time.Time
    Value float64
}

type MACDResult struct {
    DIF       []DataPoint
    Signal    []DataPoint
    Histogram []DataPoint
}

type KDResult struct {
    K []DataPoint
    D []DataPoint
}

type BBResult struct {
    Upper []DataPoint
    Mid   []DataPoint
    Lower []DataPoint
}

func MA(prices []Price, period int) []DataPoint
func EMA(prices []Price, period int) []DataPoint
func RSI(prices []Price, period int) []DataPoint
func MACD(prices []Price) MACDResult
func KD(prices []Price, period int) KDResult
func BollingerBands(prices []Price, period int) BBResult
func Volume(prices []Price) []DataPoint
```

---

## Syncer & Scheduler

### Data Sources (Free, No Auth Required)

- **TWSE listed stocks daily K:**
  `https://openapi.twse.com.tw/v1/exchangeReport/STOCK_DAY?stockNo={symbol}`
- **TPEx OTC stocks daily close:**
  `https://www.tpex.org.tw/openapi/v1/tpex_mainboard_daily_close_quotes`

### Scheduler

Uses `robfig/cron`. Runs weekdays only at 18:30 (after market close at 15:00).

```go
cron.AddFunc("30 18 * * 1-5", func() {
    syncer.SyncAllWithRetry(ctx)
})
```

### Retry on Failure

If sync fails, retry once after 5 minutes using `time.AfterFunc`. Both auto and manual triggers use the same retry logic. All results (success/failure) are recorded in `sync_logs`.

```go
func (s *Syncer) SyncAllWithRetry(ctx context.Context) {
    err := s.SyncAll(ctx)
    if err != nil {
        log.Printf("sync failed: %v, retrying in 5 minutes...", err)
        time.AfterFunc(5*time.Minute, func() {
            if err := s.SyncAll(ctx); err != nil {
                log.Printf("retry failed: %v", err)
                // write sync_logs status = "failed"
            }
        })
    }
}
```

---

## Error Handling

| Layer | Strategy |
|-------|----------|
| HTTP handlers | Uniform JSON error response with status code |
| Syncer | Errors logged and written to sync_logs; server does not crash |
| External API | 10-second timeout; triggers retry on failure |
| DB connection | Fatal on startup if connection unavailable |

---

## Testing Strategy

| Layer | Target | Method |
|-------|--------|--------|
| `pkg/indicator` | Calculation correctness | Unit tests with fixed price sequences |
| `internal/service` | Business logic | Mock repository via interface |
| `internal/handler` | HTTP request/response | `httptest` package |
| `internal/syncer` | Data fetch & parse | Mock HTTP client |

---

## Development Setup

```bash
docker-compose up -d                                              # Start PostgreSQL
migrate -path ./migrations -database $DATABASE_URL up            # Apply schema
go run ./cmd/api                                                  # Start API server
```
