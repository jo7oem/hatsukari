CREATE TABLE socials (
                         id SERIAL PRIMARY KEY,
                         author_id INT REFERENCES authors(id) NOT NULL ,
                         social_type VARCHAR(50) NOT NULL,
                         content TEXT NOT NULL
);
