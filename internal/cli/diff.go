package cli

import (
	"github.com/spf13/cobra"

	"composepack/internal/app"
)

// NewDiffCommand returns the `composepack diff` command.
func NewDiffCommand(application *app.Application) *cobra.Command {
	var (
		valueFiles   []string
		setValues    []string
		chartSrc     string
		runtimeDir   string
		showFiles    bool
		contextLines int
	)

	cmd := &cobra.Command{
		Use:   "diff <release>",
		Short: "Show what would change if install/up were run now",
		Long: `Compare the currently deployed release with what would be deployed.

This command renders the new configuration and compares it with the current
release to show what containers would be affected and what configuration
changes would be made.

If the release exists, the chart source is auto-resolved from the release
metadata. You can override it with --chart to compare against a different chart.

If the release doesn't exist, --chart is required to show what would be created.

This helps answer: "If I run install now, will it restart my database?"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			overrides, err := parseSetFlags(setValues)
			if err != nil {
				return err
			}

			releaseDir, err := cmd.Flags().GetString("release-dir")
			if err != nil {
				return err
			}

			opts := app.DiffOptions{
				RenderOptions: app.RenderOptions{
					ReleaseName:    args[0],
					ChartSource:    chartSrc,
					ValueFiles:     append([]string{}, valueFiles...),
					SetValues:      overrides,
					RuntimeBaseDir: releaseDir,
					RuntimePath:    runtimeDir,
				},
				ShowFiles:    showFiles,
				ContextLines: contextLines,
			}

			return application.DiffRelease(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&chartSrc, "chart", "", "chart directory or archive to compare (auto-resolved from release if omitted)")
	cmd.Flags().StringArrayVarP(&valueFiles, "values", "f", nil, "values files to include")
	cmd.Flags().StringArrayVar(&setValues, "set", nil, "direct values to set (key=value)")
	cmd.Flags().StringVar(&runtimeDir, "runtime-dir", "", "path to existing release directory (overrides --release-dir, advanced use only)")
	cmd.Flags().BoolVar(&showFiles, "show-files", false, "show diffs for changed files in addition to compose")
	cmd.Flags().IntVarP(&contextLines, "context", "C", 3, "number of context lines in diff output")

	return cmd
}
