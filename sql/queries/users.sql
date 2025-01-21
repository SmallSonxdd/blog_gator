-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, name)
VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE name = $1 LIMIT 1;

-- name: DeleteAllUsers :exec
TRUNCATE TABLE users CASCADE;

-- name: GetUsers :many
SELECT name FROM users;

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

-- name: ListFeed :many
SELECT feeds.name, feeds.url, users.name 
FROM feeds
INNER JOIN users ON feeds.user_id = users.id;

-- name: CreateFeedFollow :many
WITH inserted_feed_follow AS (
    INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
    VALUES (
        $1,
        $2,
        $3,
        $4,
        $5
    )
    RETURNING *
)
SELECT  inserted_feed_follow.*,
        feeds.name AS feed_name,
        users.name AS user_name
FROM inserted_feed_follow
INNER JOIN users ON inserted_feed_follow.user_id = users.id
INNER JOIN feeds ON inserted_feed_follow.feed_id = feeds.id;

-- name: GetFeed :one
SELECT id FROM feeds
WHERE url = $1 LIMIT 1;

-- name: GetFeedFollowsForUser :many
SELECT feed_follows.id, users.name, feeds.name
FROM feed_follows
INNER JOIN users ON feed_follows.user_id = users.id
INNER JOIN feeds ON feed_follows.feed_id = feeds.id
WHERE users.name = $1;

-- name: DeleteFeedFollow :exec
DELETE FROM feed_follows
USING users, feeds
WHERE feed_follows.user_id = users.id AND feed_follows.feed_id = feeds.id AND users.id = $1 AND feeds.url = $2;

-- name: MarkFeedFetched :exec
UPDATE feeds
SET last_fetched_at = NOW(),
    updated_at = NOW()
WHERE id = $1;

-- name: GetNextFeedToFetch :one
SELECT id, url
FROM feeds
ORDER BY last_fetched_at NULLS FIRST
LIMIT 1;

-- name: CreatePost :one
INSERT INTO posts (id, created_at, updated_at, title, url, 
description, published_at, feed_id)
VALUES (
    $1,
    NOW(),
    NOW(),
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: GetPostsForUser :many
SELECT posts.id, posts.created_at, posts.updated_at, posts.title, posts.url,
posts.description, posts.published_at, posts.feed_id FROM posts
INNER JOIN feed_follows ON posts.feed_id = feed_follows.feed_id
INNER JOIN users ON feed_follows.user_id = users.id
WHERE users.name = $1
ORDER BY posts.created_at
LIMIT $2;