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
