-- name: CreateFeed :one
INSERT INTO feeds (id, created_at, updated_at, name, url, user_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListFeeds :many
SELECT * FROM feeds;

-- name: GetFeedsWithUser :many
SELECT feeds.*, users.name AS user_name
FROM feeds
INNER JOIN users ON feeds.user_id = users.id;

-- name: GetFeedByURL :one
SELECT * FROM feeds WHERE url = $1;

-- name: GetFeedByID :one
SELECT * FROM feeds WHERE id = $1;

-- name: MarkFeedFetched :exec
UPDATE feeds SET last_fetched_at = $1 WHERE id = $2;

-- name: GetNextFeedToFetch :one
SELECT * FROM feeds
WHERE last_fetched_at IS NULL OR last_fetched_at < NOW() - INTERVAL '1 hour'
ORDER BY last_fetched_at ASC
LIMIT 1;

-- name: UpdateFeed :exec
UPDATE feeds SET name = $1, url = $2 WHERE id = $3;

-- name: DeleteFeed :exec
DELETE FROM feeds WHERE id = $1;

-- name: DeleteFeedsByUser :exec
DELETE FROM feeds WHERE user_id = $1;

-- name: GetFeedsByUser :many
SELECT * FROM feeds WHERE user_id = $1;