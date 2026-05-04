package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings" // Added for error checking
	"time"    // Added for current time

	config "github.com/AaroLankinen/blog_aggregator/internal/config"
	"github.com/AaroLankinen/blog_aggregator/internal/database" // Assuming sqlc generated code is here
	"github.com/google/uuid"                                    // Added for UUID generation

	_ "github.com/lib/pq"
)

type State struct {
	Config   *config.Config
	Commands *Commands
	DB       *sql.DB           // Added: Database connection
	Queries  *database.Queries // Added: SQLC generated queries
}

type Command struct {
	Name string
	Args []string
}

type Commands struct {
	Handlers map[string]func(*State, Command) error
}

type RSSFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel RSSChannel `xml:"channel"`
}

type RSSChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func (c *Commands) AddHandler(name string, handler func(*State, Command) error) {
	c.Handlers[name] = handler
}

func (c *Commands) GetHandler(name string) (func(*State, Command) error, bool) {
	handler, exists := c.Handlers[name]
	return handler, exists
}

func (c *Commands) Handle(state *State, command Command) error {
	handler, exists := c.GetHandler(command.Name)
	if !exists {
		return fmt.Errorf("unknown command: %s", command.Name)
	}
	return handler(state, command)
}

func handlerLogin(state *State, command Command) error {
	if len(command.Args) < 1 {
		return fmt.Errorf("username is required")
	}
	username := command.Args[0]

	user, err := state.Queries.GetUserByName(context.Background(), username)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Fprintf(os.Stderr, "error: user '%s' not found\n", username)
			os.Exit(1)
		}
		return fmt.Errorf("failed to retrieve user '%s': %w", username, err)
	}

	// Set the retrieved user as the current user in the config
	fmt.Printf("Logged in as '%s'.\n", user.Name)
	return state.Config.SetUser(user.Name)
}

// handlerRegister implements the "register" command.
// It creates a new user in the database and sets them as the current user.
func handlerRegister(state *State, command Command) error {
	if len(command.Args) < 1 {
		return fmt.Errorf("usage: register <username>")
	}
	username := command.Args[0]

	now := time.Now().UTC()
	id := uuid.New()

	user, err := state.Queries.CreateUser(context.Background(), database.CreateUserParams{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		Name:      username,
	})
	if err != nil {
		// Check for unique constraint violation on the name field
		if strings.Contains(err.Error(), "duplicate key value") && strings.Contains(err.Error(), "users_name_key") {
			fmt.Fprintf(os.Stderr, "error: user with name '%s' already exists\n", username)
			os.Exit(1)
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Set the newly created user as the current user in the config
	err = state.Config.SetUser(user.Name)
	if err != nil {
		return fmt.Errorf("failed to set current user in config: %w", err)
	}

	fmt.Printf("User '%s' registered successfully and set as current user.\n", user.Name)
	return nil
}

// handlerReset implements the "reset" command. It deletes all users from the users table and resets the current user in the config.
func handlerReset(state *State, command Command) error {
	// Get all users from the database
	users, err := state.Queries.ListUsers(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to retrieve users: %v\n", err)
		os.Exit(1)
	}

	// Delete each user using the DeleteUser query
	for _, user := range users {
		err := state.Queries.DeleteUser(context.Background(), user.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to delete user '%s': %v\n", user.Name, err)
			os.Exit(1)
		}
	}

	// Clear the current user from the config
	err = state.Config.SetUser("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to clear current user in config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Reset successful. All users deleted.\n")
	return nil
}

// handlerUsers implements the "users" command. It displays all users in the database, marking the current user.
func handlerUsers(state *State, command Command) error {
	users, err := state.Queries.ListUsers(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to retrieve users: %v\n", err)
		os.Exit(1)
	}

	if len(users) == 0 {
		fmt.Println("No users found.")
		return nil
	}

	currentUser := state.Config.GetUser()
	for _, user := range users {
		if user.Name == currentUser {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}
	return nil
}

// handlerAddFeed implements the "addfeed" command. It creates a new feed for the current user.
func handlerAddFeed(state *State, command Command) error {
	if len(command.Args) < 2 {
		return fmt.Errorf("usage: addfeed <name> <url>")
	}

	feedName := command.Args[0]
	feedURL := command.Args[1]

	currentUserName := state.Config.GetUser()
	if currentUserName == "" {
		return fmt.Errorf("no current user configured")
	}

	user, err := state.Queries.GetUserByName(context.Background(), currentUserName)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("current user '%s' not found", currentUserName)
		}
		return fmt.Errorf("failed to lookup current user: %w", err)
	}

	now := time.Now().UTC()
	id := uuid.New()

	feed, err := state.Queries.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		Name:      feedName,
		Url:       feedURL,
		UserID:    user.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to create feed: %w", err)
	}

	// Automatically follow the newly created feed for the user
	_, err = state.Queries.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to automatically follow created feed: %w", err)
	}

	fmt.Printf("Feed '%s' created and followed successfully.\n", feed.Name)
	return nil
}

// handlerFeeds implements the "feeds" command. It displays all feeds in the database.
func handlerFeeds(state *State, command Command) error {
	feeds, err := state.Queries.GetFeedsWithUser(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to retrieve feeds: %v\n", err)
		os.Exit(1)
	}

	if len(feeds) == 0 {
		fmt.Println("No feeds found.")
		return nil
	}

	for _, feed := range feeds {
		fmt.Printf("* %s\n", feed.Name)
		fmt.Printf("  URL: %s\n", feed.Url)
		fmt.Printf("  User: %s\n", feed.UserName)
		fmt.Println()
	}
	return nil
}

// fetchFeed fetches and parses an RSS feed from the given URL.
func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var feed RSSFeed
	err = xml.Unmarshal(body, &feed)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	return &feed, nil
}

// handlerAgg implements the "agg" command. It fetches and displays an RSS feed.
func handlerAgg(state *State, command Command) error {
	feed, err := fetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Feed: %+v\n", feed)
	return nil
}

// handlerFollow implements the "follow" command. It creates a feed follow record for the current user.
func handlerFollow(state *State, command Command) error {
	if len(command.Args) < 1 {
		return fmt.Errorf("usage: follow <url>")
	}

	feedURL := command.Args[0]

	currentUserName := state.Config.GetUser()
	if currentUserName == "" {
		fmt.Fprintln(os.Stderr, "error: no current user configured")
		os.Exit(1)
	}

	user, err := state.Queries.GetUserByName(context.Background(), currentUserName)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Fprintf(os.Stderr, "error: current user '%s' not found\n", currentUserName)
			os.Exit(1)
		}
		return fmt.Errorf("failed to lookup current user: %w", err)
	}

	feed, err := state.Queries.GetFeedByURL(context.Background(), feedURL)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Fprintf(os.Stderr, "error: feed with URL '%s' not found\n", feedURL)
			os.Exit(1)
		}
		return fmt.Errorf("failed to lookup feed: %w", err)
	}

	now := time.Now().UTC()
	id := uuid.New()

	_, err = state.Queries.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		FeedID:    feed.ID,
		UserID:    user.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to create feed follow: %w", err)
	}

	fmt.Printf("Feed '%s' followed by '%s'\n", feed.Name, user.Name)
	return nil
}

// handlerFollowing implements the "following" command. It displays all feeds the current user is following.
func handlerFollowing(state *State, command Command) error {
	currentUserName := state.Config.GetUser()
	if currentUserName == "" {
		return fmt.Errorf("no current user configured")
	}

	user, err := state.Queries.GetUserByName(context.Background(), currentUserName)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("current user '%s' not found", currentUserName)
		}
		return fmt.Errorf("failed to lookup current user: %w", err)
	}

	follows, err := state.Queries.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to retrieve feed follows: %v\n", err)
		os.Exit(1)
	}

	if len(follows) == 0 {
		fmt.Println("No feeds being followed.")
		return nil
	}

	fmt.Printf("Feeds followed by %s:\n", currentUserName)
	for _, follow := range follows {
		fmt.Printf("* %s\n", follow.FeedName)
	}
	return nil
}

// main is the entry point of the application.
func main() {
	cfg, err := config.ReadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading config: %v\n", err)
		os.Exit(1)
	}

	// Initialize database connection
	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close() // Ensure database connection is closed

	queries := database.New(db)

	cmds := &Commands{
		Handlers: make(map[string]func(*State, Command) error),
	}
	cmds.AddHandler("login", handlerLogin)
	cmds.AddHandler("register", handlerRegister)   // Register the new handler
	cmds.AddHandler("reset", handlerReset)         // Register the reset handler
	cmds.AddHandler("users", handlerUsers)         // Register the users handler
	cmds.AddHandler("agg", handlerAgg)             // Register the agg handler
	cmds.AddHandler("addfeed", handlerAddFeed)     // Register the addfeed handler
	cmds.AddHandler("feeds", handlerFeeds)         // Register the feeds handler
	cmds.AddHandler("follow", handlerFollow)       // Register the follow handler
	cmds.AddHandler("following", handlerFollowing) // Register the following handler
	state := &State{
		Config:   &cfg,
		Commands: cmds,
		DB:       db,      // Pass DB connection to state
		Queries:  queries, // Pass queries object to state
	}

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <command> [args...]\n", os.Args[0])
		os.Exit(1)
	}

	cmd := Command{
		Name: os.Args[1],
		Args: os.Args[2:],
	}

	err = state.Commands.Handle(state, cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
