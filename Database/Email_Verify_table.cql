CREATE TABLE IF NOT EXISTS iot_data.staging_subscribers (
    token text PRIMARY KEY,
    email text,
    created_at timestamp
);
