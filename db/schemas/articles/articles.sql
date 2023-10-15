CREATE TABLE articles (
                          src_path varchar(255) NOT NULL PRIMARY KEY ,
                          need_compile boolean NOT NULL DEFAULT true,
                          title varchar(255) NOT NULL
);
CREATE TABLE article_detail (
                          src_path varchar(255) NOT NULL PRIMARY KEY REFERENCES articles(src_path) ON DELETE CASCADE,
                          title varchar(255) NOT NULL
);
