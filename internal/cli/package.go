package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"composepack/internal/app"
	"composepack/internal/packager"
)

// NewPackageCommand creates chart archives (.cpack.tgz).
func NewPackageCommand(application *app.Application) *cobra.Command {
	var (
		destination string
		outputName  string
		force       bool
	)

	cmd := &cobra.Command{
		Use:   "package <chart-dir>",
		Short: "Package a chart directory into a .cpack.tgz archive",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := packager.Options{
				ChartPath:   args[0],
				Destination: destination,
				OutputName:  outputName,
				Force:       force,
			}
			path, err := packager.PackageChart(cmd.Context(), application.Runtime.ChartLoader, opts)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", path)
			return nil
		},
	}

	cmd.Flags().StringVarP(&destination, "destination", "d", ".", "output directory for the packaged chart")
	cmd.Flags().StringVarP(&outputName, "output", "o", "", "output filename (defaults to <name>-<version>.cpack.tgz)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing output file")

	return cmd
}
