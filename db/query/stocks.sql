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
