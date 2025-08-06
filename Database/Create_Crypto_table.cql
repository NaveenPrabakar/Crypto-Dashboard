CREATE TABLE iot_data.crypto_price_by_coin (
    coin_id text,
    timestamp timestamp,
    price_usd double,
    PRIMARY KEY (coin_id, timestamp)
) WITH CLUSTERING ORDER BY (timestamp DESC);
