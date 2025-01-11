package main

import (
	"cli/config"
	"cli/feed"
	"cli/layout"
	"errors"
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

	//resolve the local IDE
	localIde, err := layout.ResolveLocallyAvailableIde(localConfig)

	var resolveLocallyAvailableIdeNotFound *layout.ResolveLocallyAvailableIdeNotFound
	if errors.As(err, &resolveLocallyAvailableIdeNotFound) {
		fmt.Println("IDE not found locally. Downloading...")

		remoteIde, err := feed.ResolveRemoteIdeByConfig(localConfig.GetIDE())
		if err != nil {
			log.Fatalln("Failed to find remote IDE. ", err)
		}

		fmt.Printf("Found remote IDE. %v\n", remoteIde)

		//downloadIDE(localConfig)
		localIde, err = layout.ResolveLocallyAvailableIde(localConfig)
	}

	if err != nil {
		log.Fatalln("Failed to find IDE. ", err)
	}

	fmt.Println("IDE:", localIde)
}
