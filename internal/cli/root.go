package cli

import (
	"github.com/spf13/cobra"

	"composepack/internal/app"
)

// NewRootCommand wires together the CLI commands described in PRD/CLAUDE.
func NewRootCommand(application *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "composepack",
		Short:         "Helm-style templating and packaging for Docker Compose",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			releaseDir, err := cmd.Flags().GetString("release-dir")
			if err != nil {
				return err
			}
			if releaseDir != "" {
				application.Runtime.Config.ReleasesBaseDir = releaseDir
			}
			return nil
		},
	}

	cmd.PersistentFlags().String("release-dir", application.Runtime.Config.ReleasesBaseDir, "override default releases base directory")

	cmd.AddCommand(
		NewInstallCommand(application),
		NewTemplateCommand(application),
		NewUpCommand(application),
		NewDownCommand(application),
		NewLogsCommand(application),
		NewPSCommand(application),
		NewVersionCommand(),
		NewInitCommand(),
		NewPackageCommand(application),
	)

	return cmd
}
