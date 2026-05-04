-- name: CreateFeed :one
INSERT INTO feeds (id, created_at, updated_at, name, url, user_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: GetFeedByID :one
SELECT * FROM feeds
WHERE id = $1 LIMIT 1;

-- name: ListFeeds :many
SELECT * FROM feeds
ORDER BY name;

-- name: UpdateFeed :exec
UPDATE feeds
SET name = $2, updated_at = $3
WHERE id = $1;

-- name: DeleteFeed :exec
DELETE FROM feeds
WHERE id = $1;

-- name: ListFeedsByUser :many
SELECT * FROM feeds
WHERE user_id = $1
ORDER BY name;