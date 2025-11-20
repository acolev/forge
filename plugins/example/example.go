package main

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
)

type ExamplePlugin struct{}

type Event struct {
	Name    string
	Payload any
}

func (p *ExamplePlugin) Name() string {
	return "Example Plugin"
}

func (p *ExamplePlugin) RegisterCommands(rootCmd *cobra.Command) {
	exampleCmd := &cobra.Command{
		Use:   "example",
		Short: "An example command provided by the Example Plugin",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Example command executed")
		},
	}
	rootCmd.AddCommand(exampleCmd)
}

// Handle implements hooks.Handler — плагин слушает события.
func (p *ExamplePlugin) Handle(ctx context.Context, name string, payload *any) error {
	fmt.Println(*payload)
	if name == "db.migrate.before" {
		fmt.Println("ExamplePlugin: db.migrate.before event received")
	}
	return nil
}

// Exported variable Plugin must be of type ExamplePlugin
var Plugin ExamplePlugin
