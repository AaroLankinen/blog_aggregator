# Blog Aggregator

A lightweight Go command-line RSS aggregator. It stores users, feeds, subscriptions, and posts in PostgreSQL, and lets users browse feed posts and run continuous scraping.

## Features

- register and login users
- add RSS feeds and automatically follow them
- follow / unfollow feeds
- list feeds and subscriptions
- browse recent posts from followed feeds
- run an aggregator loop to fetch feeds periodically

## Requirements

- Go 1.26+
- PostgreSQL

## Setup

### 1. Configure the database

Create a PostgreSQL database for the project, for example:

```sh
createdb blog_aggregator
```

Apply the SQL schema in `sql/schema/` in order.
If you use a migration tool, run the migrations there; otherwise apply the SQL files manually.

### 2. Create the config file

Create `~/.gatorconfig.json` with your database URL:

```json
{
  "db_url": "postgres://user:password@localhost:5432/blog_aggregator?sslmode=disable",
  "current_user_name": ""
}
```

`current_user_name` may be empty until you register or login.

### 3. Build or run

Use `go run` during development:

```sh
go run .
```

Or build the binary:

```sh
go build -o blog_aggregator .
./blog_aggregator
```

## Usage

The program now runs as a simple REPL. Start the binary and enter commands at the prompt.

### User commands

- `register <username>` — create a new user and set it as the active user
- `login <username>` — set an existing user as the active user
- `users` — list all users
- `reset` — delete all users and clear the active user

### Feed commands

- `addfeed <name> <url>` — add a new feed and follow it for the current user
- `feeds` — list all feeds and their owners
- `follow <url>` — follow an existing feed by URL
- `unfollow <url>` — unfollow a feed by URL
- `following` — list feeds followed by the current user

### Content commands

- `browse [limit]` — show recent posts from feeds followed by the current user; default limit is `2`
- `agg <interval>` — start the background scraper; it fetches due feeds every interval (for example `10m`)
- `stop` — stop the running background scraper without exiting the REPL
- `exit` — quit the REPL and exit the program
- `usage` — display available commands

## REPL and history

The CLI now runs in REPL mode and supports line editing with arrow-key navigation through history, just like a normal Linux terminal. History is stored in `~/.blog_aggregator_history`.

## Notes

- `addfeed` automatically creates a `feed_follows` record for the current user.
- The scraper uses `last_fetched_at` to select feeds that have not been fetched recently, with a one-hour staleness threshold.
- The config file is stored in the user home directory as `.gatorconfig.json`.
- If you change SQL queries, run `sqlc generate` to regenerate the database client.

## Schema

The database schema includes:

- `users`
- `feeds`
- `feed_follows`
- `posts`

Posts are uniquely keyed by `feed_id` and `url`, so duplicate RSS entries are not stored twice.
