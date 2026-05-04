-- name: CreateFeedFollow :one
INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListFeedFollowsByUser :many
SELECT * FROM feed_follows WHERE user_id = $1;

-- name: GetFeedFollowsForUser :many
SELECT
    feed_follows.*,
    feeds.name AS feed_name,
    users.name AS user_name
FROM feed_follows
INNER JOIN feeds ON feed_follows.feed_id = feeds.id
INNER JOIN users ON feed_follows.user_id = users.id
WHERE feed_follows.user_id = $1;

-- name: DeleteFeedFollow :exec
DELETE FROM feed_follows WHERE user_id = $1 AND feed_id = $2;

-- name: GetFeedFollowByID :one
SELECT * FROM feed_follows WHERE id = $1;

-- name: GetFeedFollowByFeedID :one
SELECT * FROM feed_follows WHERE feed_id = $1;

-- name: GetFeedFollowByUserID :one
SELECT * FROM feed_follows WHERE user_id = $1;