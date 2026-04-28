package cmd

import "github.com/spf13/cobra"

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Introspect or display the cached Sanity schema",
}

func init() {
	rootCmd.AddCommand(schemaCmd)
}
