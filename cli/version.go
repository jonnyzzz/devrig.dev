package main

import (
	"fmt"
	"github.com/spf13/cobra"
)

const version = "1.0.0-SNAPSHOT"

func VersionAndBuild() string {
	return version
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the version of the tool",
	Long:  `Show the version of the idew command-line tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Version:", version)
	},
}
