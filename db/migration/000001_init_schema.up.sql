CREATE TABLE pings (
    id SERIAL PRIMARY KEY,
    ip TEXT NOT NULL,
    timestamp TIME,
    success BOOLEAN
);
