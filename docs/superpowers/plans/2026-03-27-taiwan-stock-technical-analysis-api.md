# Taiwan Stock Technical Analysis REST API — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go REST API that fetches Taiwan stock (TWSE/TPEx) daily price data, stores it in PostgreSQL, and computes technical indicators (MA, EMA, RSI, MACD, KD, Bollinger Bands, Volume) on-demand.

**Architecture:** Layered architecture — handler → service → repository (sqlc-generated). Technical indicator calculation lives in `pkg/indicator` as pure functions. A scheduler + syncer fetches daily data from TWSE/TPEx free Open APIs with automatic retry on failure.

**Tech Stack:** Go 1.22+, Gin, pgx/v5, sqlc, golang-migrate, robfig/cron, godotenv

---

## File Map

```
cmd/api/main.go                           ← wire-up: DB, server, scheduler
internal/
  handler/
    response.go                           ← shared response/error types
    stock.go                              ← GET /stocks, GET /stocks/:symbol, GET /stocks/:symbol/prices
    indicator.go                          ← GET /stocks/:symbol/indicators
    sync.go                               ← POST /sync, POST /sync/:symbol, GET /sync/status
  service/
    stock.go                              ← stock + price business logic
    indicator.go                          ← indicator dispatch + conversion
    sync.go                               ← sync orchestration (calls syncer)
  repository/                             ← sqlc-generated (do not edit manually)
    db.go, models.go, querier.go, *.sql.go
  syncer/
    twse.go                               ← fetch + parse TWSE STOCK_DAY_ALL
    twse_test.go
    tpex.go                               ← fetch + parse TPEx daily close quotes
    tpex_test.go
    syncer.go                             ← SyncAll + SyncAllWithRetry
  scheduler/
    scheduler.go                          ← robfig/cron weekday 18:30 job
pkg/indicator/
  types.go                                ← Price, DataPoint, MACDResult, KDResult, BBResult
  ma.go + ma_test.go                      ← MA, EMA
  rsi.go + rsi_test.go
  macd.go + macd_test.go
  kd.go + kd_test.go
  bb.go + bb_test.go
  volume.go + volume_test.go
db/query/
  stocks.sql
  prices.sql
  sync_logs.sql
migrations/
  000001_create_stocks.{up,down}.sql
  000002_create_daily_prices.{up,down}.sql
  000003_create_sync_logs.{up,down}.sql
docker-compose.yml
.env / .env.example
sqlc.yaml
go.mod
```

---

## Task 1: Project Scaffold

**Files:**
- Create: `go.mod`
- Create: `docker-compose.yml`
- Create: `.env.example`
- Create: `.env`
- Create: `.gitignore`
- Create: `sqlc.yaml`

- [ ] **Step 1: Initialise Go module**

```bash
cd C:/Users/sunny/RiderProjects/TechnicalAnalysis
go mod init github.com/sunny/technical-analysis
```

- [ ] **Step 2: Create `docker-compose.yml`**

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: technicalanalysis
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
```

- [ ] **Step 3: Create `.env.example` and `.env`**

`.env.example`:
```
DATABASE_URL=postgres://postgres:postgres@localhost:5432/technicalanalysis?sslmode=disable
PORT=8080
```

`.env` (same content — local dev values):
```
DATABASE_URL=postgres://postgres:postgres@localhost:5432/technicalanalysis?sslmode=disable
PORT=8080
```

- [ ] **Step 4: Create `.gitignore`**

```gitignore
.env
*.exe
*.out
/tmp
.superpowers/
```

- [ ] **Step 5: Create `sqlc.yaml`**

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "db/query"
    schema: "migrations"
    gen:
      go:
        package: "repository"
        out: "internal/repository"
        sql_package: "pgx/v5"
        overrides:
          - db_type: "date"
            go_type:
              import: "time"
              type: "Time"
          - db_type: "timestamp"
            nullable: false
            go_type:
              import: "time"
              type: "Time"
```

- [ ] **Step 6: Add dependencies**

```bash
go get github.com/gin-gonic/gin
go get github.com/jackc/pgx/v5
go get github.com/jackc/pgx/v5/pgxpool
go get github.com/robfig/cron/v3
go get github.com/joho/godotenv
go mod tidy
```

- [ ] **Step 7: Install tools (run once, not tracked in go.mod)**

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

- [ ] **Step 8: Commit**

```bash
git add go.mod go.sum docker-compose.yml .env.example .gitignore sqlc.yaml
git commit -m "feat: project scaffold — go module, docker-compose, sqlc config"
```

---

## Task 2: Database Migrations

**Files:**
- Create: `migrations/000001_create_stocks.up.sql`
- Create: `migrations/000001_create_stocks.down.sql`
- Create: `migrations/000002_create_daily_prices.up.sql`
- Create: `migrations/000002_create_daily_prices.down.sql`
- Create: `migrations/000003_create_sync_logs.up.sql`
- Create: `migrations/000003_create_sync_logs.down.sql`

- [ ] **Step 1: Create stock migration**

`migrations/000001_create_stocks.up.sql`:
```sql
CREATE TABLE stocks (
    symbol     VARCHAR(10)  PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    market     VARCHAR(10)  NOT NULL,
    created_at TIMESTAMP    DEFAULT NOW()
);
```

`migrations/000001_create_stocks.down.sql`:
```sql
DROP TABLE IF EXISTS stocks;
```

- [ ] **Step 2: Create daily_prices migration**

`migrations/000002_create_daily_prices.up.sql`:
```sql
CREATE TABLE daily_prices (
    id     BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(10)       NOT NULL REFERENCES stocks(symbol),
    date   DATE              NOT NULL,
    open   DOUBLE PRECISION,
    high   DOUBLE PRECISION,
    low    DOUBLE PRECISION,
    close  DOUBLE PRECISION,
    volume BIGINT,
    UNIQUE (symbol, date)
);
```

`migrations/000002_create_daily_prices.down.sql`:
```sql
DROP TABLE IF EXISTS daily_prices;
```

- [ ] **Step 3: Create sync_logs migration**

`migrations/000003_create_sync_logs.up.sql`:
```sql
CREATE TABLE sync_logs (
    id          BIGSERIAL   PRIMARY KEY,
    triggered   VARCHAR(10) NOT NULL,
    status      VARCHAR(10) NOT NULL,
    message     TEXT        NOT NULL DEFAULT '',
    started_at  TIMESTAMP   NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMP
);
```

`migrations/000003_create_sync_logs.down.sql`:
```sql
DROP TABLE IF EXISTS sync_logs;
```

- [ ] **Step 4: Start PostgreSQL and apply migrations**

```bash
docker-compose up -d
# Wait ~5 seconds for Postgres to start, then:
migrate -path ./migrations -database "$DATABASE_URL" up
```

Expected output:
```
1/u create_stocks (X ms)
2/u create_daily_prices (X ms)
3/u create_sync_logs (X ms)
```

- [ ] **Step 5: Commit**

```bash
git add migrations/
git commit -m "feat: add database migrations for stocks, daily_prices, sync_logs"
```

---

## Task 3: SQL Queries + sqlc Generate

**Files:**
- Create: `db/query/stocks.sql`
- Create: `db/query/prices.sql`
- Create: `db/query/sync_logs.sql`
- Generated: `internal/repository/` (do not edit)

- [ ] **Step 1: Create `db/query/stocks.sql`**

```sql
-- name: ListStocks :many
SELECT symbol, name, market, created_at
FROM stocks
ORDER BY symbol;

-- name: GetStock :one
SELECT symbol, name, market, created_at
FROM stocks
WHERE symbol = $1;

-- name: UpsertStock :exec
INSERT INTO stocks (symbol, name, market)
VALUES ($1, $2, $3)
ON CONFLICT (symbol) DO UPDATE
    SET name   = EXCLUDED.name,
        market = EXCLUDED.market;
```

- [ ] **Step 2: Create `db/query/prices.sql`**

```sql
-- name: GetDailyPricesBySymbol :many
SELECT symbol, date, open, high, low, close, volume
FROM daily_prices
WHERE symbol = $1
ORDER BY date ASC;

-- name: GetDailyPricesBySymbolAndDateRange :many
SELECT symbol, date, open, high, low, close, volume
FROM daily_prices
WHERE symbol = $1
  AND date >= $2
  AND date <= $3
ORDER BY date ASC;

-- name: UpsertDailyPrice :exec
INSERT INTO daily_prices (symbol, date, open, high, low, close, volume)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (symbol, date) DO UPDATE
    SET open   = EXCLUDED.open,
        high   = EXCLUDED.high,
        low    = EXCLUDED.low,
        close  = EXCLUDED.close,
        volume = EXCLUDED.volume;
```

- [ ] **Step 3: Create `db/query/sync_logs.sql`**

```sql
-- name: CreateSyncLog :one
INSERT INTO sync_logs (triggered, status)
VALUES ($1, $2)
RETURNING id, triggered, status, message, started_at, finished_at;

-- name: UpdateSyncLog :exec
UPDATE sync_logs
SET status      = $2,
    message     = $3,
    finished_at = NOW()
WHERE id = $1;

-- name: GetLastSyncLog :one
SELECT id, triggered, status, message, started_at, finished_at
FROM sync_logs
ORDER BY started_at DESC
LIMIT 1;
```

- [ ] **Step 4: Run sqlc generate**

```bash
sqlc generate
```

Expected: `internal/repository/` directory is created with `db.go`, `models.go`, `querier.go`, `stocks.sql.go`, `prices.sql.go`, `sync_logs.sql.go`. No errors.

- [ ] **Step 5: Commit**

```bash
git add db/ internal/repository/
git commit -m "feat: add sqlc SQL queries and generate repository layer"
```

---

## Task 4: pkg/indicator Types

**Files:**
- Create: `pkg/indicator/types.go`

- [ ] **Step 1: Create `pkg/indicator/types.go`**

```go
package indicator

import "time"

// Price is the unified input type for all indicator functions.
type Price struct {
	Date   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

// DataPoint is a single time-series value (used by MA, EMA, RSI, Volume).
type DataPoint struct {
	Date  time.Time
	Value float64
}

// MACDResult holds the three MACD series.
type MACDResult struct {
	DIF       []DataPoint
	Signal    []DataPoint
	Histogram []DataPoint
}

// KDResult holds K and D series.
type KDResult struct {
	K []DataPoint
	D []DataPoint
}

// BBResult holds Bollinger Bands series.
type BBResult struct {
	Upper []DataPoint
	Mid   []DataPoint
	Lower []DataPoint
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./pkg/indicator/...
```

Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add pkg/indicator/types.go
git commit -m "feat: add indicator types (Price, DataPoint, MACDResult, KDResult, BBResult)"
```

---

## Task 5: MA and EMA

**Files:**
- Create: `pkg/indicator/ma.go`
- Create: `pkg/indicator/ma_test.go`

- [ ] **Step 1: Write the failing tests**

`pkg/indicator/ma_test.go`:
```go
package indicator_test

import (
	"testing"
	"time"

	"github.com/sunny/technical-analysis/pkg/indicator"
)

func makePrices(closes []float64) []indicator.Price {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := make([]indicator.Price, len(closes))
	for i, c := range closes {
		prices[i] = indicator.Price{Date: base.AddDate(0, 0, i), Close: c}
	}
	return prices
}

func TestMA(t *testing.T) {
	prices := makePrices([]float64{10, 11, 12, 13, 14, 15})

	result := indicator.MA(prices, 5)

	if len(result) != 2 {
		t.Fatalf("expected 2 data points, got %d", len(result))
	}
	if result[0].Value != 12.0 {
		t.Errorf("expected 12.0, got %v", result[0].Value)
	}
	if result[1].Value != 13.0 {
		t.Errorf("expected 13.0, got %v", result[1].Value)
	}
}

func TestMA_InsufficientData(t *testing.T) {
	prices := makePrices([]float64{10, 11})
	result := indicator.MA(prices, 5)
	if result != nil {
		t.Errorf("expected nil for insufficient data, got %v", result)
	}
}

func TestEMA(t *testing.T) {
	// k = 2/(3+1) = 0.5
	// EMA[0]=10, EMA[1]=10.5, EMA[2]=11.25, EMA[3]=12.125, EMA[4]=13.0625
	prices := makePrices([]float64{10, 11, 12, 13, 14})

	result := indicator.EMA(prices, 3)

	if len(result) != 5 {
		t.Fatalf("expected 5 data points, got %d", len(result))
	}
	if result[0].Value != 10.0 {
		t.Errorf("EMA[0]: expected 10.0, got %v", result[0].Value)
	}
	if result[1].Value != 10.5 {
		t.Errorf("EMA[1]: expected 10.5, got %v", result[1].Value)
	}
	if result[4].Value != 13.0625 {
		t.Errorf("EMA[4]: expected 13.0625, got %v", result[4].Value)
	}
}

func TestEMA_Empty(t *testing.T) {
	result := indicator.EMA([]indicator.Price{}, 3)
	if result != nil {
		t.Errorf("expected nil for empty prices")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./pkg/indicator/... -run "TestMA|TestEMA" -v
```

Expected: FAIL — `indicator.MA undefined`

- [ ] **Step 3: Implement `pkg/indicator/ma.go`**

```go
package indicator

// MA returns Simple Moving Average data points.
// Returns nil if len(prices) < period.
func MA(prices []Price, period int) []DataPoint {
	if len(prices) < period {
		return nil
	}
	result := make([]DataPoint, 0, len(prices)-period+1)
	for i := period - 1; i < len(prices); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += prices[j].Close
		}
		result = append(result, DataPoint{
			Date:  prices[i].Date,
			Value: sum / float64(period),
		})
	}
	return result
}

// EMA returns Exponential Moving Average data points.
// Seeds with the first price; returns nil for empty input.
// len(result) == len(prices).
func EMA(prices []Price, period int) []DataPoint {
	if len(prices) == 0 {
		return nil
	}
	k := 2.0 / float64(period+1)
	result := make([]DataPoint, len(prices))
	result[0] = DataPoint{Date: prices[0].Date, Value: prices[0].Close}
	for i := 1; i < len(prices); i++ {
		result[i] = DataPoint{
			Date:  prices[i].Date,
			Value: prices[i].Close*k + result[i-1].Value*(1-k),
		}
	}
	return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/indicator/... -run "TestMA|TestEMA" -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/indicator/ma.go pkg/indicator/ma_test.go
git commit -m "feat: implement MA and EMA indicator functions"
```

---

## Task 6: RSI

**Files:**
- Create: `pkg/indicator/rsi.go`
- Create: `pkg/indicator/rsi_test.go`

- [ ] **Step 1: Write the failing tests**

`pkg/indicator/rsi_test.go`:
```go
package indicator_test

import (
	"testing"
)

func TestRSI_InsufficientData(t *testing.T) {
	prices := makePrices([]float64{10, 11, 12})
	result := indicator.RSI(prices, 14)
	if result != nil {
		t.Errorf("expected nil for insufficient data")
	}
}

func TestRSI_AllGains(t *testing.T) {
	// All prices rising → RSI should be 100
	closes := []float64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}
	prices := makePrices(closes)
	result := indicator.RSI(prices, 14)
	if len(result) == 0 {
		t.Fatal("expected at least one result")
	}
	if result[0].Value != 100.0 {
		t.Errorf("all-gain RSI: expected 100.0, got %v", result[0].Value)
	}
}

func TestRSI_AllLosses(t *testing.T) {
	// All prices falling → RSI should be 0
	closes := []float64{24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10}
	prices := makePrices(closes)
	result := indicator.RSI(prices, 14)
	if len(result) == 0 {
		t.Fatal("expected at least one result")
	}
	if result[0].Value != 0.0 {
		t.Errorf("all-loss RSI: expected 0.0, got %v", result[0].Value)
	}
}

func TestRSI_Length(t *testing.T) {
	// 20 prices, period=14 → expect 20-14=6 results
	closes := make([]float64, 20)
	for i := range closes {
		closes[i] = float64(i + 1)
	}
	prices := makePrices(closes)
	result := indicator.RSI(prices, 14)
	if len(result) != 6 {
		t.Errorf("expected 6 results, got %d", len(result))
	}
}
```

- [ ] **Step 2: Run to verify they fail**

```bash
go test ./pkg/indicator/... -run "TestRSI" -v
```

Expected: FAIL — `indicator.RSI undefined`

- [ ] **Step 3: Implement `pkg/indicator/rsi.go`**

```go
package indicator

// RSI computes Wilder's Relative Strength Index.
// Returns nil if len(prices) <= period.
// len(result) == len(prices) - period.
func RSI(prices []Price, period int) []DataPoint {
	if len(prices) <= period {
		return nil
	}

	// Seed: compute first average gain and loss over [1..period]
	var gainSum, lossSum float64
	for i := 1; i <= period; i++ {
		change := prices[i].Close - prices[i-1].Close
		if change > 0 {
			gainSum += change
		} else {
			lossSum -= change
		}
	}
	avgGain := gainSum / float64(period)
	avgLoss := lossSum / float64(period)

	result := make([]DataPoint, 0, len(prices)-period)
	result = append(result, DataPoint{
		Date:  prices[period].Date,
		Value: rsiValue(avgGain, avgLoss),
	})

	// Wilder smoothing for subsequent values
	for i := period + 1; i < len(prices); i++ {
		change := prices[i].Close - prices[i-1].Close
		gain, loss := 0.0, 0.0
		if change > 0 {
			gain = change
		} else {
			loss = -change
		}
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
		result = append(result, DataPoint{
			Date:  prices[i].Date,
			Value: rsiValue(avgGain, avgLoss),
		})
	}
	return result
}

func rsiValue(avgGain, avgLoss float64) float64 {
	if avgLoss == 0 {
		if avgGain == 0 {
			return 50.0
		}
		return 100.0
	}
	rs := avgGain / avgLoss
	return 100 - 100/(1+rs)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/indicator/... -run "TestRSI" -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/indicator/rsi.go pkg/indicator/rsi_test.go
git commit -m "feat: implement RSI indicator function"
```

---

## Task 7: MACD

**Files:**
- Create: `pkg/indicator/macd.go`
- Create: `pkg/indicator/macd_test.go`

- [ ] **Step 1: Write the failing tests**

`pkg/indicator/macd_test.go`:
```go
package indicator_test

import (
	"testing"
)

func TestMACD_Length(t *testing.T) {
	// Need enough prices for EMA(26). Let's use 30.
	closes := make([]float64, 30)
	for i := range closes {
		closes[i] = float64(i + 10)
	}
	prices := makePrices(closes)

	result := indicator.MACD(prices)

	if len(result.DIF) != 30 {
		t.Errorf("DIF: expected 30, got %d", len(result.DIF))
	}
	if len(result.Signal) != 30 {
		t.Errorf("Signal: expected 30, got %d", len(result.Signal))
	}
	if len(result.Histogram) != 30 {
		t.Errorf("Histogram: expected 30, got %d", len(result.Histogram))
	}
}

func TestMACD_HistogramEquality(t *testing.T) {
	// Histogram[i] == DIF[i] - Signal[i]
	closes := make([]float64, 30)
	for i := range closes {
		closes[i] = float64(i + 10)
	}
	prices := makePrices(closes)

	result := indicator.MACD(prices)

	for i := range result.Histogram {
		expected := result.DIF[i].Value - result.Signal[i].Value
		if abs(result.Histogram[i].Value-expected) > 1e-9 {
			t.Errorf("Histogram[%d]: expected %v, got %v", i, expected, result.Histogram[i].Value)
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
```

- [ ] **Step 2: Run to verify they fail**

```bash
go test ./pkg/indicator/... -run "TestMACD" -v
```

Expected: FAIL — `indicator.MACD undefined`

- [ ] **Step 3: Implement `pkg/indicator/macd.go`**

```go
package indicator

// MACD computes the MACD indicator using EMA(12), EMA(26), and Signal EMA(9).
// All three result slices have the same length as prices.
func MACD(prices []Price) MACDResult {
	ema12 := EMA(prices, 12)
	ema26 := EMA(prices, 26)

	// DIF = EMA12 - EMA26
	dif := make([]DataPoint, len(prices))
	for i := range prices {
		dif[i] = DataPoint{Date: prices[i].Date, Value: ema12[i].Value - ema26[i].Value}
	}

	// Signal = EMA(DIF, 9) — reuse EMA with DIF values as "Close"
	difPrices := make([]Price, len(dif))
	for i, d := range dif {
		difPrices[i] = Price{Date: d.Date, Close: d.Value}
	}
	signal := EMA(difPrices, 9)

	// Histogram = DIF - Signal
	histogram := make([]DataPoint, len(dif))
	for i := range dif {
		histogram[i] = DataPoint{Date: dif[i].Date, Value: dif[i].Value - signal[i].Value}
	}

	return MACDResult{DIF: dif, Signal: signal, Histogram: histogram}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/indicator/... -run "TestMACD" -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/indicator/macd.go pkg/indicator/macd_test.go
git commit -m "feat: implement MACD indicator function"
```

---

## Task 8: KD (Stochastic)

**Files:**
- Create: `pkg/indicator/kd.go`
- Create: `pkg/indicator/kd_test.go`

- [ ] **Step 1: Write the failing tests**

`pkg/indicator/kd_test.go`:
```go
package indicator_test

import (
	"testing"
	"time"

	"github.com/sunny/technical-analysis/pkg/indicator"
)

func makeOHLCPrices(data [][4]float64) []indicator.Price {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := make([]indicator.Price, len(data))
	for i, d := range data {
		prices[i] = indicator.Price{
			Date:  base.AddDate(0, 0, i),
			Open:  d[0],
			High:  d[1],
			Low:   d[2],
			Close: d[3],
		}
	}
	return prices
}

func TestKD_InsufficientData(t *testing.T) {
	prices := makeOHLCPrices([][4]float64{{10, 12, 9, 11}, {11, 13, 10, 12}})
	result := indicator.KD(prices, 9)
	if len(result.K) != 0 || len(result.D) != 0 {
		t.Errorf("expected empty results for insufficient data")
	}
}

func TestKD_FirstValue(t *testing.T) {
	// period=3, initial K=50, initial D=50
	// i=2: highest=14, lowest=10, RSV=(13-10)/(14-10)*100=75
	// K = 50*(2/3) + 75*(1/3) = 33.33+25 = 58.33
	// D = 50*(2/3) + 58.33*(1/3) = 33.33+19.44 = 52.78
	data := [][4]float64{
		{10, 12, 10, 11},
		{11, 13, 11, 12},
		{12, 14, 12, 13},
	}
	prices := makeOHLCPrices(data)
	result := indicator.KD(prices, 3)

	if len(result.K) != 1 {
		t.Fatalf("expected 1 K value, got %d", len(result.K))
	}
	wantK := 50.0*2/3 + 75.0*1/3
	if abs(result.K[0].Value-wantK) > 0.01 {
		t.Errorf("K: expected %.4f, got %.4f", wantK, result.K[0].Value)
	}
	wantD := 50.0*2/3 + wantK*1/3
	if abs(result.D[0].Value-wantD) > 0.01 {
		t.Errorf("D: expected %.4f, got %.4f", wantD, result.D[0].Value)
	}
}

func TestKD_Length(t *testing.T) {
	data := make([][4]float64, 20)
	for i := range data {
		f := float64(i + 10)
		data[i] = [4]float64{f, f + 1, f - 1, f}
	}
	prices := makeOHLCPrices(data)
	result := indicator.KD(prices, 9)
	// 20 prices, period=9 → 20-9+1=12 results
	if len(result.K) != 12 {
		t.Errorf("expected 12 K values, got %d", len(result.K))
	}
}
```

- [ ] **Step 2: Run to verify they fail**

```bash
go test ./pkg/indicator/... -run "TestKD" -v
```

Expected: FAIL — `indicator.KD undefined`

- [ ] **Step 3: Implement `pkg/indicator/kd.go`**

```go
package indicator

// KD computes the Stochastic KD indicator (Taiwan-style).
// Initial K and D values are seeded at 50.
// Uses 1/3 smoothing (K = prevK*2/3 + RSV*1/3).
// Returns empty KDResult if len(prices) < period.
func KD(prices []Price, period int) KDResult {
	if len(prices) < period {
		return KDResult{}
	}

	k, d := 50.0, 50.0
	kValues := make([]DataPoint, 0, len(prices)-period+1)
	dValues := make([]DataPoint, 0, len(prices)-period+1)

	for i := period - 1; i < len(prices); i++ {
		highest := prices[i-period+1].High
		lowest := prices[i-period+1].Low
		for j := i - period + 2; j <= i; j++ {
			if prices[j].High > highest {
				highest = prices[j].High
			}
			if prices[j].Low < lowest {
				lowest = prices[j].Low
			}
		}

		rsv := 50.0
		if highest != lowest {
			rsv = (prices[i].Close - lowest) / (highest - lowest) * 100
		}

		k = k*2/3 + rsv*1/3
		d = d*2/3 + k*1/3

		kValues = append(kValues, DataPoint{Date: prices[i].Date, Value: k})
		dValues = append(dValues, DataPoint{Date: prices[i].Date, Value: d})
	}

	return KDResult{K: kValues, D: dValues}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/indicator/... -run "TestKD" -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/indicator/kd.go pkg/indicator/kd_test.go
git commit -m "feat: implement KD stochastic indicator function"
```

---

## Task 9: Bollinger Bands

**Files:**
- Create: `pkg/indicator/bb.go`
- Create: `pkg/indicator/bb_test.go`

- [ ] **Step 1: Write the failing tests**

`pkg/indicator/bb_test.go`:
```go
package indicator_test

import (
	"math"
	"testing"
)

func TestBollingerBands_Length(t *testing.T) {
	prices := makePrices([]float64{10, 11, 12, 13, 14, 15})
	result := indicator.BollingerBands(prices, 3)
	// 6 prices, period=3 → 4 results
	if len(result.Mid) != 4 || len(result.Upper) != 4 || len(result.Lower) != 4 {
		t.Errorf("expected 4 values each, got Mid=%d Upper=%d Lower=%d",
			len(result.Mid), len(result.Upper), len(result.Lower))
	}
}

func TestBollingerBands_FirstValues(t *testing.T) {
	// prices=[10,11,12], period=3
	// Mid = (10+11+12)/3 = 11.0
	// std = sqrt(((10-11)^2+(11-11)^2+(12-11)^2)/3) = sqrt(2/3)
	// Upper = 11 + 2*sqrt(2/3), Lower = 11 - 2*sqrt(2/3)
	prices := makePrices([]float64{10, 11, 12, 13})
	result := indicator.BollingerBands(prices, 3)

	wantMid := 11.0
	wantStd := math.Sqrt(2.0 / 3.0)
	wantUpper := wantMid + 2*wantStd
	wantLower := wantMid - 2*wantStd

	if abs(result.Mid[0].Value-wantMid) > 1e-9 {
		t.Errorf("Mid: expected %v, got %v", wantMid, result.Mid[0].Value)
	}
	if abs(result.Upper[0].Value-wantUpper) > 1e-9 {
		t.Errorf("Upper: expected %v, got %v", wantUpper, result.Upper[0].Value)
	}
	if abs(result.Lower[0].Value-wantLower) > 1e-9 {
		t.Errorf("Lower: expected %v, got %v", wantLower, result.Lower[0].Value)
	}
}

func TestBollingerBands_SymmetryAroundMid(t *testing.T) {
	prices := makePrices([]float64{10, 11, 12, 13, 14})
	result := indicator.BollingerBands(prices, 3)
	for i := range result.Mid {
		upperDiff := result.Upper[i].Value - result.Mid[i].Value
		lowerDiff := result.Mid[i].Value - result.Lower[i].Value
		if abs(upperDiff-lowerDiff) > 1e-9 {
			t.Errorf("band[%d] not symmetric: upper-mid=%v mid-lower=%v", i, upperDiff, lowerDiff)
		}
	}
}
```

- [ ] **Step 2: Run to verify they fail**

```bash
go test ./pkg/indicator/... -run "TestBollingerBands" -v
```

Expected: FAIL — `indicator.BollingerBands undefined`

- [ ] **Step 3: Implement `pkg/indicator/bb.go`**

```go
package indicator

import "math"

// BollingerBands computes Bollinger Bands: Mid ± 2 standard deviations.
// Returns empty BBResult if len(prices) < period.
func BollingerBands(prices []Price, period int) BBResult {
	mas := MA(prices, period)
	if mas == nil {
		return BBResult{}
	}

	upper := make([]DataPoint, len(mas))
	lower := make([]DataPoint, len(mas))

	for i, ma := range mas {
		priceIdx := i + period - 1
		sum := 0.0
		for j := priceIdx - period + 1; j <= priceIdx; j++ {
			diff := prices[j].Close - ma.Value
			sum += diff * diff
		}
		std := math.Sqrt(sum / float64(period))
		upper[i] = DataPoint{Date: ma.Date, Value: ma.Value + 2*std}
		lower[i] = DataPoint{Date: ma.Date, Value: ma.Value - 2*std}
	}

	return BBResult{Upper: upper, Mid: mas, Lower: lower}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/indicator/... -run "TestBollingerBands" -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/indicator/bb.go pkg/indicator/bb_test.go
git commit -m "feat: implement Bollinger Bands indicator function"
```

---

## Task 10: Volume

**Files:**
- Create: `pkg/indicator/volume.go`
- Create: `pkg/indicator/volume_test.go`

- [ ] **Step 1: Write the failing tests**

`pkg/indicator/volume_test.go`:
```go
package indicator_test

import (
	"testing"
	"time"

	"github.com/sunny/technical-analysis/pkg/indicator"
)

func TestVolume(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := []indicator.Price{
		{Date: base, Volume: 1000},
		{Date: base.AddDate(0, 0, 1), Volume: 2000},
		{Date: base.AddDate(0, 0, 2), Volume: 3000},
	}

	result := indicator.Volume(prices)

	if len(result) != 3 {
		t.Fatalf("expected 3 data points, got %d", len(result))
	}
	if result[0].Value != 1000 {
		t.Errorf("expected 1000, got %v", result[0].Value)
	}
	if result[2].Value != 3000 {
		t.Errorf("expected 3000, got %v", result[2].Value)
	}
}

func TestVolume_Empty(t *testing.T) {
	result := indicator.Volume([]indicator.Price{})
	if len(result) != 0 {
		t.Errorf("expected empty result for empty input")
	}
}
```

- [ ] **Step 2: Run to verify they fail**

```bash
go test ./pkg/indicator/... -run "TestVolume" -v
```

Expected: FAIL — `indicator.Volume undefined`

- [ ] **Step 3: Implement `pkg/indicator/volume.go`**

```go
package indicator

// Volume converts price Volume fields into DataPoint slices.
func Volume(prices []Price) []DataPoint {
	result := make([]DataPoint, len(prices))
	for i, p := range prices {
		result[i] = DataPoint{Date: p.Date, Value: float64(p.Volume)}
	}
	return result
}
```

- [ ] **Step 4: Run all indicator tests**

```bash
go test ./pkg/indicator/... -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/indicator/volume.go pkg/indicator/volume_test.go
git commit -m "feat: implement Volume indicator function; all indicator tests passing"
```

---

## Task 11: Stock Service

**Files:**
- Create: `internal/service/stock.go`

- [ ] **Step 1: Create `internal/service/stock.go`**

```go
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/sunny/technical-analysis/internal/repository"
	"github.com/sunny/technical-analysis/pkg/indicator"
)

// StockService handles stock and price business logic.
type StockService struct {
	q repository.Querier
}

// NewStockService creates a StockService.
func NewStockService(q repository.Querier) *StockService {
	return &StockService{q: q}
}

// ListStocks returns all stocks.
func (s *StockService) ListStocks(ctx context.Context) ([]repository.Stock, error) {
	return s.q.ListStocks(ctx)
}

// GetStock returns a single stock by symbol.
func (s *StockService) GetStock(ctx context.Context, symbol string) (repository.Stock, error) {
	stock, err := s.q.GetStock(ctx, symbol)
	if err != nil {
		return repository.Stock{}, fmt.Errorf("stock %q not found", symbol)
	}
	return stock, nil
}

// GetPrices returns OHLCV data for a symbol, optionally filtered by date range.
// from and to are optional (zero time means no filter).
func (s *StockService) GetPrices(ctx context.Context, symbol string, from, to time.Time) ([]repository.DailyPrice, error) {
	if !from.IsZero() && !to.IsZero() {
		return s.q.GetDailyPricesBySymbolAndDateRange(ctx, repository.GetDailyPricesBySymbolAndDateRangeParams{
			Symbol: symbol,
			Date:   from,
			Date_2: to,
		})
	}
	return s.q.GetDailyPricesBySymbol(ctx, symbol)
}

// PricesToIndicatorPrices converts repository DailyPrice rows to indicator.Price slice.
func PricesToIndicatorPrices(rows []repository.DailyPrice) []indicator.Price {
	prices := make([]indicator.Price, len(rows))
	for i, r := range rows {
		prices[i] = indicator.Price{
			Date:   r.Date,
			Open:   r.Open,
			High:   r.High,
			Low:    r.Low,
			Close:  r.Close,
			Volume: r.Volume,
		}
	}
	return prices
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/service/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/service/stock.go
git commit -m "feat: add stock service (list, get, prices)"
```

---

## Task 12: Indicator Service

**Files:**
- Create: `internal/service/indicator.go`

- [ ] **Step 1: Create `internal/service/indicator.go`**

```go
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/sunny/technical-analysis/pkg/indicator"
)

// IndicatorResult is the unified return type for any indicator computation.
type IndicatorResult struct {
	Symbol    string
	Name      string
	Indicator string
	Period    int
	Data      any // []indicator.DataPoint | indicator.MACDResult | indicator.KDResult | indicator.BBResult
}

// IndicatorService computes technical indicators.
type IndicatorService struct {
	stock *StockService
}

// NewIndicatorService creates an IndicatorService.
func NewIndicatorService(stock *StockService) *IndicatorService {
	return &IndicatorService{stock: stock}
}

// Compute fetches prices for symbol and computes the requested indicator.
// indicatorType: ma, ema, rsi, macd, kd, bb, volume
// period: used for ma, ema, rsi, kd, bb (ignored for macd, volume)
func (s *IndicatorService) Compute(ctx context.Context, symbol, indicatorType string, period int) (IndicatorResult, error) {
	stock, err := s.stock.GetStock(ctx, symbol)
	if err != nil {
		return IndicatorResult{}, err
	}

	rows, err := s.stock.GetPrices(ctx, symbol, time.Time{}, time.Time{})
	if err != nil {
		return IndicatorResult{}, fmt.Errorf("failed to fetch prices for %s: %w", symbol, err)
	}
	if len(rows) == 0 {
		return IndicatorResult{}, fmt.Errorf("no price data for symbol %q", symbol)
	}

	prices := PricesToIndicatorPrices(rows)

	var data any
	switch indicatorType {
	case "ma":
		data = indicator.MA(prices, period)
	case "ema":
		data = indicator.EMA(prices, period)
	case "rsi":
		data = indicator.RSI(prices, period)
	case "macd":
		data = indicator.MACD(prices)
	case "kd":
		data = indicator.KD(prices, period)
	case "bb":
		data = indicator.BollingerBands(prices, period)
	case "volume":
		data = indicator.Volume(prices)
	default:
		return IndicatorResult{}, fmt.Errorf("unknown indicator type %q", indicatorType)
	}

	return IndicatorResult{
		Symbol:    stock.Symbol,
		Name:      stock.Name,
		Indicator: indicatorType,
		Period:    period,
		Data:      data,
	}, nil
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/service/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/service/indicator.go
git commit -m "feat: add indicator service with dispatch for all indicator types"
```

---

## Task 13: Handler — Response Types + Stock Endpoints

**Files:**
- Create: `internal/handler/response.go`
- Create: `internal/handler/stock.go`

- [ ] **Step 1: Create `internal/handler/response.go`**

```go
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
```

- [ ] **Step 2: Create `internal/handler/stock.go`**

```go
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sunny/technical-analysis/internal/repository"
	"github.com/sunny/technical-analysis/internal/service"
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

	rows, err := h.svc.GetPrices(c.Request.Context(), symbol, from, to)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(c, toPriceResponses(rows))
}

func parseDateRange(c *gin.Context) (time.Time, time.Time) {
	const layout = "2006-01-02"
	from, _ := time.Parse(layout, c.Query("from"))
	to, _ := time.Parse(layout, c.Query("to"))
	return from, to
}

func toPriceResponses(rows []repository.DailyPrice) []PriceResponse {
	resp := make([]PriceResponse, len(rows))
	for i, r := range rows {
		resp[i] = PriceResponse{
			Date:   r.Date.Format("2006-01-02"),
			Open:   r.Open,
			High:   r.High,
			Low:    r.Low,
			Close:  r.Close,
			Volume: r.Volume,
		}
	}
	return resp
}
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./internal/handler/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/handler/response.go internal/handler/stock.go
git commit -m "feat: add stock handler (list stocks, get stock, get prices)"
```

---

## Task 14: Handler — Indicator Endpoint

**Files:**
- Create: `internal/handler/indicator.go`

- [ ] **Step 1: Create `internal/handler/indicator.go`**

```go
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
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/handler/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/handler/indicator.go
git commit -m "feat: add indicator handler with type dispatch and period defaulting"
```

---

## Task 15: TWSE Syncer

**Files:**
- Create: `internal/syncer/twse.go`
- Create: `internal/syncer/twse_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/syncer/twse_test.go`:
```go
package syncer_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunny/technical-analysis/internal/syncer"
)

func TestTWSEFetcher_ParsesResponse(t *testing.T) {
	sample := []map[string]string{
		{
			"Date":         "20250326",
			"Code":         "2330",
			"Name":         "台積電",
			"TradeVolume":  "23,456,789",
			"OpeningPrice": "1000.0",
			"HighestPrice": "1010.0",
			"LowestPrice":  "995.0",
			"ClosingPrice": "1005.0",
		},
	}
	body, _ := json.Marshal(sample)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer server.Close()

	fetcher := syncer.NewTWSEFetcher(server.URL)
	records, err := fetcher.FetchAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	r := records[0]
	if r.Symbol != "2330" {
		t.Errorf("Symbol: expected 2330, got %s", r.Symbol)
	}
	if r.Name != "台積電" {
		t.Errorf("Name: expected 台積電, got %s", r.Name)
	}
	if r.Close != 1005.0 {
		t.Errorf("Close: expected 1005.0, got %v", r.Close)
	}
	if r.Volume != 23456789 {
		t.Errorf("Volume: expected 23456789, got %v", r.Volume)
	}
}

func TestTWSEFetcher_HandlesEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	fetcher := syncer.NewTWSEFetcher(server.URL)
	records, err := fetcher.FetchAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}
```

- [ ] **Step 2: Run to verify they fail**

```bash
go test ./internal/syncer/... -run "TestTWSE" -v
```

Expected: FAIL — `syncer.NewTWSEFetcher undefined`

- [ ] **Step 3: Create `internal/syncer/twse.go`**

```go
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
		date, err := time.Parse("20060102", r.Date)
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/syncer/... -run "TestTWSE" -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/syncer/twse.go internal/syncer/twse_test.go
git commit -m "feat: add TWSE fetcher with parse tests"
```

---

## Task 16: TPEx Syncer

**Files:**
- Create: `internal/syncer/tpex.go`
- Create: `internal/syncer/tpex_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/syncer/tpex_test.go`:
```go
package syncer_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunny/technical-analysis/internal/syncer"
)

func TestTPExFetcher_ParsesResponse(t *testing.T) {
	// TPEx uses ROC year (e.g. 114 = 2025) and "Volumn" (their typo)
	sample := []map[string]string{
		{
			"Date":                    "114/03/26",
			"SecuritiesCompanyCode":   "6505",
			"CompanyName":             "台塑化",
			"Open":                    "80.00",
			"High":                    "81.00",
			"Low":                     "79.50",
			"Close":                   "80.50",
			"Volumn":                  "1234567",
		},
	}
	body, _ := json.Marshal(sample)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer server.Close()

	fetcher := syncer.NewTPExFetcher(server.URL)
	records, err := fetcher.FetchAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	r := records[0]
	if r.Symbol != "6505" {
		t.Errorf("Symbol: expected 6505, got %s", r.Symbol)
	}
	if r.Market != "TPEx" {
		t.Errorf("Market: expected TPEx, got %s", r.Market)
	}
	if r.Close != 80.50 {
		t.Errorf("Close: expected 80.50, got %v", r.Close)
	}
	if r.Volume != 1234567 {
		t.Errorf("Volume: expected 1234567, got %v", r.Volume)
	}
	if r.Date.Year() != 2025 || r.Date.Month() != 3 || r.Date.Day() != 26 {
		t.Errorf("Date: expected 2025-03-26, got %v", r.Date)
	}
}
```

- [ ] **Step 2: Run to verify they fail**

```bash
go test ./internal/syncer/... -run "TestTPEx" -v
```

Expected: FAIL — `syncer.NewTPExFetcher undefined`

- [ ] **Step 3: Create `internal/syncer/tpex.go`**

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/syncer/... -run "TestTPEx" -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/syncer/tpex.go internal/syncer/tpex_test.go
git commit -m "feat: add TPEx fetcher with ROC date parsing and tests"
```

---

## Task 17: Unified Syncer with Retry

**Files:**
- Create: `internal/syncer/syncer.go`

- [ ] **Step 1: Create `internal/syncer/syncer.go`**

```go
package syncer

import (
	"context"
	"fmt"
	"log"
	"time"

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
		Message: msg, // TEXT NOT NULL — plain string
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
			Open:   r.Open,
			High:   r.High,
			Low:    r.Low,
			Close:  r.Close,
			Volume: r.Volume,
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
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/syncer/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/syncer/syncer.go
git commit -m "feat: add unified syncer with 5-minute retry on failure"
```

---

## Task 18: Sync Service + Sync Handler

**Files:**
- Create: `internal/service/sync.go`
- Create: `internal/handler/sync.go`

- [ ] **Step 1: Create `internal/service/sync.go`**

```go
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
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
// started_at is NOT NULL → time.Time (via sqlc override).
// finished_at is nullable → pgtype.Timestamp (no override for nullable columns).
func (s *SyncService) GetStatus(ctx context.Context) (SyncStatus, error) {
	row, err := s.q.GetLastSyncLog(ctx)
	if err != nil {
		return SyncStatus{}, fmt.Errorf("no sync log found")
	}

	// finished_at is nullable TIMESTAMP — sqlc generates pgtype.Timestamp
	var finishedAt *time.Time
	if row.FinishedAt.Valid {
		t := row.FinishedAt.Time
		finishedAt = &t
	}

	return SyncStatus{
		ID:         row.ID,
		Triggered:  row.Triggered,
		Status:     row.Status,
		Message:    row.Message, // TEXT NOT NULL DEFAULT '' → string
		StartedAt:  row.StartedAt, // TIMESTAMP NOT NULL → time.Time via sqlc override
		FinishedAt: finishedAt,
	}, nil
}

// ensure pgtype is used (for FinishedAt)
var _ = pgtype.Timestamp{}
```

- [ ] **Step 2: Create `internal/handler/sync.go`**

```go
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
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./internal/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/service/sync.go internal/handler/sync.go
git commit -m "feat: add sync service and sync handler (trigger + status)"
```

---

## Task 19: Scheduler

**Files:**
- Create: `internal/scheduler/scheduler.go`

- [ ] **Step 1: Create `internal/scheduler/scheduler.go`**

```go
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
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/scheduler/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/scheduler/scheduler.go
git commit -m "feat: add cron scheduler for daily 18:30 weekday sync"
```

---

## Task 20: main.go — Wire Everything Together

**Files:**
- Create: `cmd/api/main.go`

- [ ] **Step 1: Create `cmd/api/main.go`**

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/sunny/technical-analysis/internal/handler"
	"github.com/sunny/technical-analysis/internal/repository"
	"github.com/sunny/technical-analysis/internal/scheduler"
	"github.com/sunny/technical-analysis/internal/service"
	"github.com/sunny/technical-analysis/internal/syncer"
)

func main() {
	_ = godotenv.Load() // load .env; ignore error if not found

	ctx := context.Background()

	// --- Database ---
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("cannot connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("database connected")

	// --- Repository ---
	q := repository.New(pool)

	// --- Services ---
	stockSvc := service.NewStockService(q)
	indicatorSvc := service.NewIndicatorService(stockSvc)
	syncerInst := syncer.NewSyncer(ctx, q)
	syncSvc := service.NewSyncService(q, syncerInst)

	// --- Scheduler ---
	sched := scheduler.New(syncerInst)
	sched.Start()
	defer sched.Stop()

	// --- Handlers ---
	stockH := handler.NewStockHandler(stockSvc)
	indicatorH := handler.NewIndicatorHandler(indicatorSvc)
	syncH := handler.NewSyncHandler(syncSvc)

	// --- Router ---
	r := gin.Default()
	v1 := r.Group("/api/v1")
	{
		v1.GET("/stocks", stockH.ListStocks)
		v1.GET("/stocks/:symbol", stockH.GetStock)
		v1.GET("/stocks/:symbol/prices", stockH.GetPrices)
		v1.GET("/stocks/:symbol/indicators", indicatorH.GetIndicator)

		v1.POST("/sync", syncH.TriggerFullSync)
		v1.POST("/sync/:symbol", syncH.TriggerSymbolSync)
		v1.GET("/sync/status", syncH.GetStatus)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("server starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
```

- [ ] **Step 2: Tidy dependencies**

```bash
go mod tidy
```

- [ ] **Step 3: Build the full project**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Start the server and verify it responds**

```bash
# Terminal 1 — ensure postgres is running
docker-compose up -d

# Terminal 2 — run the server
go run ./cmd/api

# Terminal 3 — smoke test
curl http://localhost:8080/api/v1/stocks
# Expected: [] (empty array — no data synced yet)

curl -X POST http://localhost:8080/api/v1/sync
# Expected: {"message":"sync started"}

curl http://localhost:8080/api/v1/sync/status
# Expected: JSON with triggered="manual", status="running" or "success"
```

- [ ] **Step 5: Commit**

```bash
git add cmd/api/main.go go.mod go.sum
git commit -m "feat: wire up main.go — all layers connected, server ready"
```

---

## Task 21: Final Build Verification

- [ ] **Step 1: Run all tests**

```bash
go test ./... -v
```

Expected: all PASS

- [ ] **Step 2: Build release binary**

```bash
go build -o ta-api ./cmd/api
```

Expected: `ta-api` binary created, no errors.

- [ ] **Step 3: Run end-to-end smoke test**

```bash
# Trigger sync and wait ~30 seconds for data
curl -X POST http://localhost:8080/api/v1/sync
# Wait for sync to complete, then:

# Query a stock (2330 = TSMC)
curl "http://localhost:8080/api/v1/stocks/2330"

# Query prices
curl "http://localhost:8080/api/v1/stocks/2330/prices"

# Query RSI
curl "http://localhost:8080/api/v1/stocks/2330/indicators?type=rsi&period=14"

# Query MACD
curl "http://localhost:8080/api/v1/stocks/2330/indicators?type=macd"
```

- [ ] **Step 4: Final commit**

```bash
git add .
git commit -m "feat: complete Taiwan stock technical analysis REST API implementation"
```
