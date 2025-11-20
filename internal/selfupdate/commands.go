package selfupdate

import "github.com/spf13/cobra"

func Register(rootCmd *cobra.Command, currentVersion string) {
	cmd := &cobra.Command{
		Use:   "self-update",
		Short: "Update forge CLI to the latest release",
		Long: `Check the latest GitHub release and update the current forge binary
in-place (download and replace the executable).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunSelfUpdate(currentVersion)
		},
	}

	rootCmd.AddCommand(cmd)
}
