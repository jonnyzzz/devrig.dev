package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"jonnyzzz.com/devrig.dev/config"
	"jonnyzzz.com/devrig.dev/feed"
	initCmd "jonnyzzz.com/devrig.dev/init"
	"jonnyzzz.com/devrig.dev/unpack"
)

var rootCmd = &cobra.Command{
	Use:   "devrig",
	Short: fmt.Sprintf("Devrig v%s - Your development entry point", VersionAndBuild()),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Select subcommand to use devrig")
		fmt.Println("")
		cmd.HelpFunc()(cmd, args)
		os.Exit(11)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd.Cmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

//goland:noinspection GoUnusedFunction
func someOldCode(cmd *cobra.Command) {
	//make it disabled
	if cmd != nil {
		return
	}

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
