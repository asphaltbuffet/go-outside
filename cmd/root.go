package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/asphaltbuffet/go-outside/internal/app"
	"github.com/asphaltbuffet/go-outside/internal/cli"
	"github.com/asphaltbuffet/go-outside/internal/config"
	"github.com/asphaltbuffet/go-outside/internal/geo"
	"github.com/asphaltbuffet/go-outside/internal/score"
	"github.com/asphaltbuffet/go-outside/internal/weather"
)

// appFactory builds the App from a config; overridable in tests to inject fakes.
var appFactory = func(cfg *config.Config) (*app.App, error) {
	return app.New(
		cfg,
		geo.NewIPLocator(nil),
		geo.NewOpenMeteoGeocoder(nil),
		weather.NewOpenMeteo(nil),
	)
}

// errNope is returned (wrapped in a cli.ExitError with code 1) when the Verdict
// is nope, so the process exits 1 per ADR-0002.
var errNope = errors.New("nope")

// NewRootCmd builds the root command with persistent flags and config loading.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "go-outside",
		Short:         "Calculate if you should go outside based on current conditions.",
		SilenceUsage:  true,
		SilenceErrors: true,
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := newApp(cmd.Context())
			if err != nil {
				return err
			}

			override, _ := cmd.Flags().GetString("location")
			assessment, err := a.Assess(cmd.Context(), override)
			if err != nil {
				return err // unclassified -> exit 2 via cli.CodeFor (mapped in Task 8)
			}

			cfg := cli.MustGetConfig(cmd.Context())
			w := cli.NewOutputWriterFromConfig(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg)
			verbose, _ := cmd.Flags().GetBool("verbose")

			switch {
			case w.IsJSON():
				if jErr := w.JSON(map[string]any{
					"verdict":      string(assessment.Verdict),
					"score":        assessment.Score,
					"location":     assessment.Location.PostalCode,
					"temp_f":       assessment.Report.TempF,
					"humidity_pct": assessment.Report.HumidityPct,
					"aqi":          assessment.Report.AQI,
					"weather_code": assessment.Report.WeatherCode,
				}); jErr != nil {
					return jErr
				}
			case verbose:
				w.Success(fmt.Sprintf("%s (%d) [%s]",
					assessment.Verdict, assessment.Score, assessment.Location.PostalCode))
			default:
				w.Success(string(assessment.Verdict))
			}

			if assessment.Verdict == score.Nope {
				return cli.NewExitError(1, errNope)
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().Bool("quiet", false, "suppress non-essential output")
	rootCmd.PersistentFlags().BoolP("json", "j", false, "output as JSON")
	rootCmd.PersistentFlags().String("config", "", "config file path (default: XDG config dirs)")
	rootCmd.PersistentFlags().Bool("no-config", false, "skip config file loading")

	rootCmd.Flags().BoolP("verbose", "v", false, "show score and location: nope (999) [22901]")
	rootCmd.Flags().String("location", "", "postal code to assess instead of auto-detecting")

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

	return appFactory(cfg)
}
