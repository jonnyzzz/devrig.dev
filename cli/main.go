package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"jonnyzzz.com/devrig.dev/config"
	"jonnyzzz.com/devrig.dev/configservice"
	"jonnyzzz.com/devrig.dev/feed"
	initCmd "jonnyzzz.com/devrig.dev/init"
	"jonnyzzz.com/devrig.dev/install"
	"jonnyzzz.com/devrig.dev/unpack"
	"jonnyzzz.com/devrig.dev/updates"
)

func main() {
	updatesService := updates.NewUpdateService(VersionAndBuild())

	rootCmd := newRootCommand(updatesService)
	rootCmd.AddCommand(NewVersionCommand())
	rootCmd.AddCommand(initCmd.NewInitCommand(updatesService))
	rootCmd.AddCommand(install.NewInstallCommand(VersionAndBuild()))

	var devrigConfigPath string
	// Add global --devrig-config flag
	rootCmd.PersistentFlags().StringVar(&devrigConfigPath, "devrig-config", "", "Path to devrig.yaml configuration file")

	configs := configservice.NewConfigService(ResolveDevrigConfigPath(devrigConfigPath))
	configs.Binaries()

	executeRootCommand(rootCmd)
}

// ResolveDevrigConfigPath resolves the path to devrig.yaml using the following precedence:
// 1. --devrig-config flag
// 2. DEVRIG_CONFIG environment variable
// 3. ./devrig.yaml (current directory)
// Always returns an absolute path.
func ResolveDevrigConfigPath(devrigConfigPath string) string {
	var path string

	// 1. Check command-line flag
	if devrigConfigPath != "" {
		path = devrigConfigPath
	} else if envPath := os.Getenv("DEVRIG_CONFIG"); envPath != "" {
		// 2. Check environment variable
		path = envPath
	} else {
		// 3. Default to current directory
		path = filepath.Join(".", "devrig.yaml")
	}

	// Always return absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		// If we can't resolve, return as-is (shouldn't happen in practice)
		return path
	}
	return absPath
}

func newRootCommand(updatesService updates.UpdateService) *cobra.Command {
	var noUpdates bool
	rootCmd := &cobra.Command{
		Use:   "devrig",
		Short: fmt.Sprintf("Devrig v%s - Your development entry point", VersionAndBuild()),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Select subcommand to use devrig")
			fmt.Println("")
			cmd.HelpFunc()(cmd, args)
			os.Exit(11)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			if !noUpdates {
				go func() {
					//just fetch the update info
					update, err := updatesService.IsUpdateAvailable()
					if err == nil && update {
						fmt.Print("\n\nUpdate available\n\n")
					}
				}()
			}
		},
	}

	rootCmd.Flags().BoolVar(&noUpdates, "no-updates", false, "Do not check for updates")
	return rootCmd
}

func executeRootCommand(rootCmd *cobra.Command) {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	} else {
		os.Exit(0)
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
