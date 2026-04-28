package cmd

import "github.com/spf13/cobra"

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Inspect Sanity projects via the Manage API",
}

func init() {
	rootCmd.AddCommand(projectCmd)
}
