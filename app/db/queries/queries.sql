-- name: DbArticlesLatest5 :many
SELECT 
	a.*, 
	f.title as feed_title 
 FROM articles a 
 INNER JOIN feeds f 
 ON f.id = a.feed_id 
 ORDER BY COALESCE(a.published, a.date_found) 
 DESC LIMIT 0, 5;

-- name: DbArticlesLatest5Starred :many
SELECT 
	a.*, 
	f.title as feed_title 
 FROM articles a 
 INNER JOIN feeds f 
 ON f.id = a.feed_id 
 WHERE a.starred > 0
 ORDER BY starred DESC, COALESCE(a.published, a.date_found) 
 DESC LIMIT 0, 5;

-- name: DbSidebarDataAll :many
SELECT
	f.title AS feed_title,
	f.id AS feed_id,
	COUNT(a.id) AS total_articles,
	COUNT(CASE WHEN a.read <> 0 THEN 1 END) AS articles_read
FROM feeds f
LEFT JOIN articles a ON f.id = a.feed_id
GROUP BY f.id, f.title
ORDER BY feed_title ASC;

-- name: DbArticlesByFeedID :many
SELECT 
	a.*, 
	f.title as feed_title 
FROM articles a 
INNER JOIN feeds f 
ON f.id = a.feed_id 
WHERE feed_id = ? 
ORDER BY COALESCE(a.published, a.date_found) DESC;

-- name: DbArticlesUnreadByFeedID :many
SELECT 
	a.*, 
	f.title as feed_title 
FROM articles a 
INNER JOIN feeds f 
ON f.id = a.feed_id 
WHERE feed_id = ? AND a.read = 0
ORDER BY COALESCE(a.published, a.date_found) DESC;

-- name: DbFeedAndArticletByArticleID :one
SELECT 
	a.id as article_id,
	a.link as article_link, 
	a.title as article_title,
	a.starred as article_stars,
	a.published as article_published,
	f.id as feed_id, 
	f.title as feed_title,
	f.url as feed_url,
	f.css_sel_container as feed_css_sel_container, 
	f.css_sel_start as feed_css_sel_start, 
	f.css_sel_stop as feed_css_sel_stop, 
	f.html_extraction_strategy as feed_html_extraction_strategy   
FROM 
	articles a 
INNER JOIN feeds f 
ON f.id = a.feed_id where a.id = ?;


-- name: DbCachedArticleByLink :one
SELECT * FROM article_cache WHERE link = ?;

-- name: DbCachedArticleCreateNew :exec
INSERT INTO article_cache (
	article_id,
	link, 
	article_content, 
	created
) VALUES(
?,
?,
?, 
CURRENT_TIMESTAMP
);

-- name: DbFeedsAll :many
SELECT * from feeds;	

-- name: DbFeedByID :one
SELECT * FROM feeds where id = ?;

-- name: DbArticlesAddArticle :exec
INSERT OR IGNORE INTO articles (
	feed_id, 
	title, 
	link, 
	published, 
	date_found,
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

-- name: DbArticleSetAsRead :exec
UPDATE articles SET read = 1 WHERE id = ?;

-- name: DbArticleSetStarredValue :exec
UPDATE articles SET starred = ? WHERE id = ?;

-- name: DbArticleSetAnnotation :exec
INSERT INTO annotations (article_id, start_data, end_data, note, snippet, date_added) 
VALUES (?,?,?,?,?, CURRENT_TIMESTAMP);

-- name: DbArticleAnnotationsByID :many
SELECT * FROM annotations WHERE article_id = ?;

-- name: DbArticleContent :one
SELECT article_content FROM article_cache WHERE article_id = ?;