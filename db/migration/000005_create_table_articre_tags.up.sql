CREATE TABLE article_tags
(
    article_id INT REFERENCES articles (id) NOT NULL,
    tag_id     INT REFERENCES tags (id)     NOT NULL,
    UNIQUE (article_id, tag_id)
);
CREATE INDEX article_id_tag_id_idx ON article_tags (article_id,tag_id);
