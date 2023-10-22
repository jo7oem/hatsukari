CREATE TABLE view_counts
(
    id    SERIAL PRIMARY KEY,
    article_id INT REFERENCES articles (id) NOT NULL UNIQUE,
    view_count INT NOT NULL DEFAULT 0 CHECK (view_count >= 0)
);
