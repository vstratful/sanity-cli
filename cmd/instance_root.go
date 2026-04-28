package cmd

import "github.com/spf13/cobra"

var instanceCmd = &cobra.Command{
	Use:   "instance",
	Short: "Manage named Sanity instances",
	Long:  `Add, list, switch, show, or remove named instances stored in the local config.`,
}

func init() {
	rootCmd.AddCommand(instanceCmd)
}
