package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"forge/internal/hooks"
	"github.com/spf13/cobra"
	"os"
)

type ExamplePlugin struct{}

// Exported variable Plugin must be a pointer so it implements the interfaces
var Plugin = &ExamplePlugin{}

var _ = Plugin // used to silence "unused variable" when building as main

// process-mode support: --describe outputs JSON describing the commands,
// otherwise the binary expects first arg to be the command use and executes it.
func main() {
	describe := flag.Bool("describe", false, "output plugin description (JSON)")
	flag.Parse()

	if *describe {
		desc := struct {
			Name     string `json:"name"`
			Commands []struct {
				Use   string `json:"use"`
				Short string `json:"short"`
			} `json:"commands"`
		}{
			Name: "example",
			Commands: []struct {
				Use   string `json:"use"`
				Short string `json:"short"`
			}{
				{Use: "example", Short: "An example command provided by the Example Plugin"},
			},
		}
		enc := json.NewEncoder(os.Stdout)
		_ = enc.Encode(desc)
		return
	}

	// If called as plugin process, first non-flag arg is command name
	args := flag.Args()
	if len(args) == 0 {
		// nothing to run
		return
	}

	switch args[0] {
	case "example":
		fmt.Println("Example command executed")
	default:
		fmt.Fprintf(os.Stderr, "unknown plugin command: %s\n", args[0])
		os.Exit(2)
	}
}

// Handle implements hooks.Handler — плагин слушает события.
func (p *ExamplePlugin) Handle(ctx context.Context, e hooks.Event) error {
	fmt.Println("Context: ", ctx)
	fmt.Println("Event name: ", e.Name)
	if e.Name == "db.migrate.before" {
		fmt.Println("ExamplePlugin: db.migrate.before event received")
		fmt.Println(e.Payload)
	}
	return nil
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
