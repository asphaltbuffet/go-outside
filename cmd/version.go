package cmd

import (
	"github.com/spf13/cobra"

	"github.com/asphaltbuffet/go-outside/internal/cli"
	"github.com/asphaltbuffet/go-outside/internal/versioncmd"
)

// newVersionCmd builds the "version" subcommand.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Ensure the App can be constructed from the loaded config.
			if _, err := newApp(cmd.Context()); err != nil {
				return err
			}

			cfg := cli.MustGetConfig(cmd.Context())
			w := cli.NewOutputWriterFromConfig(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg)

			if w.IsJSON() {
				return w.JSON(map[string]string{
					"version": versioncmd.ShortVersion(),
					"commit":  versioncmd.BuildInfo(),
					"full":    versioncmd.FullVersion(),
				})
			}

			w.Success(versioncmd.FullVersion())
			return nil
		},
	}
}
