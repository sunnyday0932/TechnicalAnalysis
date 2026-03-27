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
