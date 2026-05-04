package main

import (
	"context"
	"database/sql"
	"fmt"
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
	cmds.AddHandler("register", handlerRegister) // Register the new handler

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
