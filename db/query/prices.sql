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
