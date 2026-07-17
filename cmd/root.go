package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/asphaltbuffet/go-outside/internal/app"
	"github.com/asphaltbuffet/go-outside/internal/config"
)

// NewRootCmd builds the root command with persistent flags and config loading.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "go-outside",
		Short: "Calculate if you should go outside based on current conditions.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			noConfig, _ := cmd.Flags().GetBool("no-config")

			var cfg *config.Config
			var err error

			if noConfig {
				cfg = config.GetDefaults()
			} else {
				cfg, err = config.New(configPath)
				if err != nil {
					return fmt.Errorf("loading config: %w", err)
				}
			}

			// Override with flags.
			if quiet, _ := cmd.Flags().GetBool("quiet"); quiet {
				cfg.Output.Quiet = 1
			}
			if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
				cfg.Output.DefaultFormat = config.OutputFormatJSON
			}

			cmd.SetContext(context.WithValue(cmd.Context(), config.ConfigKey, cfg))
			return nil
		},
	}

	rootCmd.PersistentFlags().Bool("quiet", false, "suppress non-essential output")
	rootCmd.PersistentFlags().BoolP("json", "j", false, "output as JSON")
	rootCmd.PersistentFlags().String("config", "", "config file path (default: XDG config dirs)")
	rootCmd.PersistentFlags().Bool("no-config", false, "skip config file loading")

	rootCmd.AddCommand(newVersionCmd())

	return rootCmd
}

// Execute runs the root command.
func Execute(ctx context.Context) error {
	return NewRootCmd().ExecuteContext(ctx)
}

// newApp builds an App from the Config stored in ctx by PersistentPreRunE.
func newApp(ctx context.Context) (*app.App, error) {
	cfg, ok := ctx.Value(config.ConfigKey).(*config.Config)
	if !ok {
		return nil, errors.New("config not found in context")
	}

	return app.New(cfg)
}
