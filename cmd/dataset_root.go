package cmd

import "github.com/spf13/cobra"

var datasetCmd = &cobra.Command{
	Use:   "dataset",
	Short: "Inspect Sanity datasets via the Manage API",
}

func init() {
	rootCmd.AddCommand(datasetCmd)
}
