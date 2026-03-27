CREATE TABLE stocks (
    symbol     VARCHAR(10)  PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    market     VARCHAR(10)  NOT NULL,
    created_at TIMESTAMP    DEFAULT NOW()
);
