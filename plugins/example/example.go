package main

import (
	"fmt"
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

// Exported variable Plugin must be of type ExamplePlugin
var Plugin ExamplePlugin
