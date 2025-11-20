package main

import (
	"context"
	"fmt"
	"forge/internal/hooks"
	"github.com/spf13/cobra"
)

type ExamplePlugin struct{}

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
func (p *ExamplePlugin) Handle(ctx context.Context, e hooks.Event) error {
	fmt.Println("test: ", e.Name)
	if e.Name == "db.migrate.before" {
		fmt.Println("ExamplePlugin: db.migrate.before event received")
		fmt.Println(e.Payload)
	}
	return nil
}

// Exported variable Plugin must be a pointer so it implements the interfaces
var Plugin = ExamplePlugin{}
