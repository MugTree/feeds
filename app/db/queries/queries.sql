-- name: GetLatest5Articles :many
SELECT 
	a.*, 
	f.title as feed_title 
 FROM articles a 
 INNER JOIN feeds f 
 ON f.id = a.feed_id 
 ORDER BY published 
 DESC LIMIT 0, 5;

-- name: GetSidebarData :many
SELECT
	f.title AS feed_title,
	f.id AS feed_id,
	COUNT(a.id) AS total_articles,
	COUNT(CASE WHEN a.read <> 0 THEN 1 END) AS articles_read
FROM feeds f
LEFT JOIN articles a ON f.id = a.feed_id
GROUP BY f.id, f.title
ORDER BY feed_title ASC;

-- name: GetArticlesByFeedID :many
SELECT 
	a.*, 
	f.title as feed_title 
FROM articles a 
INNER JOIN feeds f 
ON f.id = a.feed_id 
WHERE feed_id = ? 
ORDER BY a.published_parsed DESC;

-- name: GetUnreadByFeedID :many
SELECT 
	a.*, 
	f.title as feed_title 
FROM articles a 
INNER JOIN feeds f 
ON f.id = a.feed_id 
WHERE feed_id = ? AND a.read = 0
ORDER BY a.published_parsed DESC;

-- name: GetFeedDataForArticleByArticleID :one
SELECT 
	a.id,
	a.link, 
	a.title,
	f.id as feed_id, 
	f.title as feed_title,
	f.url as feed_url,
	f.css_sel_container, 
	f.css_sel_start, 
	f.css_sel_stop, 
	f.html_extraction_strategy   
FROM 
	articles a 
INNER JOIN feeds f 
ON f.id = a.feed_id where a.id = ?;


-- name: GetCachedByLink :one
SELECT * FROM article_cache WHERE link = ?;

-- name: AddToArticleCache :exec
INSERT INTO article_cache (
	link, 
	article_content, 
	created
) VALUES(
?,
?, 
CURRENT_TIMESTAMP
);

-- name: GetFeeds :many
SELECT * from feeds;	

-- name: GetFeedByID :one
SELECT * FROM feeds where id = ?;

-- name: AddToArticles :exec
INSERT OR IGNORE INTO articles (
	feed_id, 
	title, 
	link, 
	published, 
	published_parsed, 
	summary, 
	read, 
	starred
) VALUES (
	 ?, 
	 ?, 
	 ?, 
	 ?, 
	 ?, 
	 ?, 
	 ?, 
	 ?
 );

-- name: SetArticleAsRead :exec
 UPDATE articles SET read = 1 WHERE id = ?;