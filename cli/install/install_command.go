package install

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewInstallCommand creates the install command with subcommands
func NewInstallCommand(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install fonts and development tools",
		Long: `Install various fonts and development tools.

Available subcommands:
  jetbrains-mono - Install JetBrains Mono font (latest version)

Examples:
  devrig install jetbrains-mono
`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Please specify a package to install.")
			fmt.Println("")
			cmd.HelpFunc()(cmd, args)
		},
	}

	// Add subcommands
	cmd.AddCommand(NewJetBrainsMonoCommand(version))

	return cmd
}

// NewJetBrainsMonoCommand creates the jetbrains-mono subcommand
func NewJetBrainsMonoCommand(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "jetbrains-mono",
		Short: "Install JetBrains Mono font",
		Long: `Install JetBrains Mono font (latest version).

JetBrains Mono is a free and open-source typeface designed for developers.
It is downloaded from the official JetBrains GitHub repository.

Examples:
  devrig install jetbrains-mono
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return installJetBrainsMono(cmd, args, version)
		},
	}
}

func installJetBrainsMono(cmd *cobra.Command, args []string, version string) error {
	cmd.Println("Installing JetBrains Mono font...")

	installer, err := NewJetBrainsMonoInstaller(version)
	if err != nil {
		return fmt.Errorf("failed to create installer: %w", err)
	}

	if err := installer.Install(cmd); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	cmd.Println("JetBrains Mono font installed successfully!")
	return nil
}
