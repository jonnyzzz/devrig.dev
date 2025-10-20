package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

const version = "1.0.0-SNAPSHOT"

func VersionAndBuild() string {
	return version
}

func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the version of the tool",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Version:", version)
		},
	}
}
