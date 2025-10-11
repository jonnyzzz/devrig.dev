package main

import (
	"cli/config"
	"cli/feed"
	"cli/unpack"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "idew",
	Short: fmt.Sprintf("IDE Wrapper v%s - Your development entry point", VersionAndBuild()),
	Long: `IDE Wrapper is a command-line tool that helps download, install, manage, 
and configure IDE and development environments for your project.

Simply include the binary and a .idew.yaml config file in your repository,
and contributors can quickly set up their development environment.`,
	Run: runMainCommand,
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
	localConfig, err := config.ResolveConfig()
	if err != nil {
		log.Fatalf("Failed to resolve configuration: %v\n", err)
	}

	fmt.Printf("Configuration loaded from: %s\n", localConfig.ConfigPath())
	fmt.Printf("IDE: %s %s", localConfig.GetIDE().Name(), localConfig.GetIDE().Version())
	if build := localConfig.GetIDE().Build(); build != "" {
		fmt.Printf(" (build: %s)", build)
	}
	fmt.Println()

	remoteIde, err := feed.ResolveRemoteIdeByConfig(localConfig.GetIDE())
	if err != nil {
		log.Fatalf("Failed to resolve remote IDE: %v\n", err)
	}

	fmt.Printf("Found remote IDE: %v\n", remoteIde)

	downloadedIde, err := feed.DownloadFeedEntry(context.Background(), remoteIde, localConfig)
	if err != nil {
		log.Fatalf("Failed to download IDE: %v\n", err)
	}

	fmt.Printf("Downloaded IDE to: %s\n", downloadedIde.TargetFile())

	unpackedIde, err := unpack.UnpackIde(localConfig, downloadedIde)
	if err != nil {
		log.Fatalf("Failed to unpack IDE: %v\n", err)
	}

	fmt.Printf("IDE unpacked successfully: %v\n", unpackedIde)
}
