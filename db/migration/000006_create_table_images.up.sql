CREATE TABLE images
(
    id   SERIAL PRIMARY KEY,
    article_id INT REFERENCES articles (id)NOT NULL ,
    file_path  TEXT UNIQUE ,
    alt_text   TEXT NOT NULL ,
    caption    TEXT NOT NULL
);