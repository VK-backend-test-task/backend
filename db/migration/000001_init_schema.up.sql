CREATE TABLE containers (
    ip TEXT NOT NULL PRIMARY KEY,
    last_ping TIME,
    last_success TIME
);
