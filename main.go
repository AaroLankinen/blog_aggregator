package main

import (
	"fmt"
	"os"

	config "github.com/AaroLankinen/blog_aggregator/internal/config"
)

type State struct {
	Config   *config.Config
	Commands *Commands
}

type Command struct {
	Name string
	Args []string
}

type Commands struct {
	Commands []Command
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

func (c *Commands) AddCommand(name string, args []string) {
	c.Commands = append(c.Commands, Command{Name: name, Args: args})
}

func handlerLogin(state *State, command Command) error {
	if len(command.Args) < 1 {
		return fmt.Errorf("username is required")
	}
	username := command.Args[0]
	return state.Config.SetUser(username)
}

func main() {
	cfg, err := config.ReadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading config: %v\n", err)
		os.Exit(1)
	}

	cmds := &Commands{
		Handlers: make(map[string]func(*State, Command) error),
	}
	cmds.AddHandler("login", handlerLogin)

	state := &State{
		Config:   &cfg,
		Commands: cmds,
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
