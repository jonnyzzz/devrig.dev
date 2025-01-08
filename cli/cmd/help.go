package cmd

import (
	"github.com/spf13/cobra"
)

var helpCmd = &cobra.Command{
	Use:   "help",
	Short: "Show help information",
	Long:  `Show help information for the idew command-line tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
