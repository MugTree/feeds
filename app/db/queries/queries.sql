-- name: SelectLatest5Articles :many
SELECT 
	a.*, 
	f.title as feed_title 
 FROM articles a 
 INNER JOIN feeds f 
 ON f.id = a.feed_id 
 ORDER BY COALESCE(a.published, a.date_found) 
 DESC LIMIT 0, 5;

-- name: SelectLatest5StarredArticles :many
SELECT 
	a.*, 
	f.title as feed_title 
 FROM articles a 
 INNER JOIN feeds f 
 ON f.id = a.feed_id 
 WHERE a.starred > 0
 ORDER BY starred DESC, COALESCE(a.published, a.date_found) 
 DESC LIMIT 0, 5;

-- name: SelectSideBarData :many
SELECT
	f.title AS feed_title,
	f.id AS feed_id,
	COUNT(a.id) AS total_articles,
	COUNT(CASE WHEN a.read <> 0 THEN 1 END) AS articles_read
FROM feeds f
LEFT JOIN articles a ON f.id = a.feed_id
GROUP BY f.id, f.title
ORDER BY feed_title ASC;

-- name: SelectArticlesByFeedID :many
SELECT 
	a.*, 
	f.title as feed_title 
FROM articles a 
INNER JOIN feeds f 
ON f.id = a.feed_id 
WHERE feed_id = ? 
ORDER BY COALESCE(a.published, a.date_found) DESC;

-- name: SelectUnreadArticlesByFeedID :many
SELECT 
	a.*, 
	f.title as feed_title 
FROM articles a 
INNER JOIN feeds f 
ON f.id = a.feed_id 
WHERE feed_id = ? AND a.read = 0
ORDER BY COALESCE(a.published, a.date_found) DESC;

-- name: SelectFeedAndArticletByArticleID :one
SELECT 
	a.id as article_id,
	a.link as article_link, 
	a.title as article_title,
	a.starred as article_stars,
	a.published as article_published,
	a.read as article_read,
	a.article_content,
	a.clickable_block_count as article_clickable_block_count,
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

-- name: SelectMarginNotesByArticleID :many
SELECT * FROM margin_notes WHERE article_id = ?;

-- name: SelectMarginNoteByArticleIDAndBlockID :one
SELECT * FROM margin_notes WHERE article_id = ? AND block_id = ?;

-- name: UpdateMarginNoteByArticleIDAndBlockID :exec
UPDATE margin_notes SET note = ? WHERE article_id =? AND block_id = ?;

-- name: InsertAndReturnMarginNote :one
INSERT INTO margin_notes (article_id, block_id, note, date_added ) VALUES (?,?,?, CURRENT_TIMESTAMP) RETURNING *;

-- name: SelectAllFeeds :many
SELECT * from feeds;	

-- name: SelectFeedByID :one
SELECT * FROM feeds where id = ?;

-- name: SelectArticleByID :one
SELECT * FROM articles where id = ?;

-- name: InsertOrIgnoreArticle :exec
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

-- name: UpdateArticleSetAsRead :exec
UPDATE articles SET read = 1 WHERE id = ?;

-- name: UpdateArticleSetStarredValue :exec
UPDATE articles SET starred = ? WHERE id = ?;

-- name: SelectArticlesByFeedIDWithLimit :many
SELECT
   a.id as article_id,
	a.link as article_link,
	a.title as article_title,
	a.starred as article_stars,
	a.published as article_published,
	a.read as article_read,
	a.feed_id as article_feed_id,
    a.article_content AS article_content,
    a.clickable_block_count AS article_clickable_block_count,
    f.title as feed_title
FROM articles a
    INNER JOIN feeds f ON f.id = a.feed_id
WHERE a.feed_id  = ?
ORDER BY published DESC
LIMIT ? OFFSET ?;

-- name: SelectArticleCountByFeedID :one
SELECT COUNT(*) FROM articles WHERE feed_id = ?;