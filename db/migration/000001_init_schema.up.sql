CREATE TABLE pings (
    id SERIAL PRIMARY KEY,
    container_ip TEXT NOT NULL,
    timestamp TIME,
    success BOOLEAN
);
