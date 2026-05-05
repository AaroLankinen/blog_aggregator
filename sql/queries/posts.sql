-- name: CreatePost :one
INSERT INTO posts (id, created_at, updated_at, title, url, description, published_at, feed_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListPosts :many
SELECT * FROM posts;

-- name: GetPostByID :one
SELECT * FROM posts WHERE id = $1;

-- name: DeletePost :exec
DELETE FROM posts WHERE id = $1;

-- name: DeletePostsByFeedID :exec
DELETE FROM posts WHERE feed_id = $1;

-- name: GetPostsByFeedID :many
SELECT * FROM posts WHERE feed_id = $1;

-- name: GetPostsByFeedIDs :many
SELECT * FROM posts WHERE feed_id = ANY($1);

-- name: GetPostsByIDs :many
SELECT * FROM posts WHERE id = ANY($1);

-- name: GetPostsByURLs :many
SELECT * FROM posts WHERE url = ANY($1);

-- name: GetPostsForUser :many
SELECT
    p.*,
    feeds.name AS feed_name
FROM posts p
JOIN feed_follows ff ON p.feed_id = ff.feed_id
JOIN feeds ON p.feed_id = feeds.id
WHERE ff.user_id = $1
ORDER BY p.published_at DESC
LIMIT $2 OFFSET $3;

-- name: GetPostsByUserID :many
SELECT p.*
FROM posts p
JOIN feed_follows ff ON p.feed_id = ff.feed_id
WHERE ff.user_id = $1
ORDER BY p.published_at DESC
LIMIT $2 OFFSET $3;