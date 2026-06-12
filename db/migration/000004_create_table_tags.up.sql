CREATE TABLE tags (
                      id SERIAL PRIMARY KEY,
                      tag_name VARCHAR(32) NOT NULL UNIQUE
);