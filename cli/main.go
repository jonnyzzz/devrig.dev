package main

import (
	"cli/config"
	"cli/feed"
	"cli/unpack"
	"context"
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
	localConfig, err := config.ResolveConfig()
	if err != nil {
		log.Fatalln("Failed to find localConfig file. ", err)
	}

	fmt.Println("Config file:", localConfig, "IDE name: ", localConfig.GetIDE().Name(), " version ", localConfig.GetIDE().Version())

	remoteIde, err := feed.ResolveRemoteIdeByConfig(localConfig.GetIDE())
	if err != nil {
		log.Fatalln("Failed to find remote IDE. ", err)
	}

	fmt.Printf("Found remote IDE. %v\n", remoteIde)

	downloadedIde, err := feed.DownloadFeedEntry(context.Background(), remoteIde, localConfig)
	if err != nil {
		log.Fatalln("Failed to download remote IDE. ", err)
	}

	fmt.Printf("Downloaded remote IDE to %s\n", downloadedIde.TargetFile())

	loadUnpackedIde, err := unpack.UnpackIde(localConfig, downloadedIde)
	if err != nil {
		log.Fatalln("Failed to unpack remote IDE. ", err)
	}

	fmt.Println("", loadUnpackedIde)
}
