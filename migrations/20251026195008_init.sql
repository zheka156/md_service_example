-- +goose Up
-- +goose StatementBegin
CREATE TABLE coin (
    ticker VARCHAR(10),
    name VARCHAR(20),
    precision INTEGER,
    UNIQUE (ticker)
);

CREATE TABLE one_hour_price(
    fromsym VARCHAR(10),
    tosym VARCHAR(10),
    last_price NUMERIC(20, 8),
    ts TIMESTAMP
);

CREATE TABLE tg_chat( 
    chatId BIGINT NOT NULL PRIMARY KEY, 
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    prime BOOLEAN,
    is_active BOOLEAN
);

CREATE TABLE chat_coins(
    id UUID PRIMARY KEY,
    chat_id BIGINT,
    coin VARCHAR,
    quantity NUMERIC,
    UNIQUE (chat_id, coin)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS one_hour_price;
DROP TABLE IF EXISTS tg_chat;
DROP TABLE IF EXISTS coin;
DROP TABLE IF EXISTS chat_coins;
-- +goose StatementEnd
