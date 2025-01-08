package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute the run command",
	Long:  `Execute the run command of the jbcli tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running the command...")
		// Add your command logic here
	},
}
