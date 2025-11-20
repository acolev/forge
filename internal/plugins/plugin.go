package plugins

import (
	"github.com/spf13/cobra"
)

type Plugin interface {
	Name() string
	RegisterCommands(rootCmd *cobra.Command)
}
