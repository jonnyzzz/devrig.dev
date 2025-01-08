package main

import (
	"cli/config"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "idew",
	Short: fmt.Sprintf("IDE Wrapper v%s is your development entry point", VersionAndBuild()),
	Run:   runMainCommand,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runMainCommand(cmd *cobra.Command, args []string) {
	configPath, err := config.ResolveConfig()
	if err != nil {
		log.Fatalln("Failed to find config file. ", err)
	}
	fmt.Println("Config file:", configPath)
}
