package main

import (
	"fmt"
	"github.com/spf13/cobra"
)

type TestPlugin struct{}

func (p *TestPlugin) Name() string {
	return "Test Plugin"
}

func (p *TestPlugin) RegisterCommands(rootCmd *cobra.Command) {
	exampleCmd := &cobra.Command{
		Use:   "test",
		Short: "An example command provided by the Example Plugin",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Test command executed")
		},
	}
	rootCmd.AddCommand(exampleCmd)
}

// Exported variable Plugin must be of type TestPlugin
var Plugin TestPlugin
