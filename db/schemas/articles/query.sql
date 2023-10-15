-- name: GetArticle :one
SELECT * FROM articles
WHERE src_path = $1 LIMIT 1;

-- name: ListArticles :many
SELECT * FROM articles
ORDER BY src_path;