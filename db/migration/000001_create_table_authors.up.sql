CREATE TABLE authors
(
    id  SERIAL PRIMARY KEY,
    author_id  VARCHAR(64) NOT NULL UNIQUE ,
    source_path   TEXT NOT NULL UNIQUE ,
    name       VARCHAR(255) NOT NULL,
    bio        TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    generated  BOOLEAN   DEFAULT FALSE
);