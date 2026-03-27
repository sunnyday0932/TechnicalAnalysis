CREATE TABLE sync_logs (
    id          BIGSERIAL    PRIMARY KEY,
    triggered   VARCHAR(10)  NOT NULL,
    status      VARCHAR(10)  NOT NULL,
    message     TEXT         NOT NULL DEFAULT '',
    started_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);
