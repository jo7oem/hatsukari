CREATE TABLE articles (
                        id serial NOT NULL PRIMARY KEY,
                        src_path varchar(255) UNIQUE NOT NULL,
                        web_path varchar(255) UNIQUE ,
                        need_compile boolean NOT NULL DEFAULT true
);
