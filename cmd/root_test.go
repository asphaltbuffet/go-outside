package cmd_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/asphaltbuffet/go-outside/cmd"
	"github.com/asphaltbuffet/go-outside/internal/app"
	"github.com/asphaltbuffet/go-outside/internal/cli"
	"github.com/asphaltbuffet/go-outside/internal/config"
	"github.com/asphaltbuffet/go-outside/internal/geo"
	"github.com/asphaltbuffet/go-outside/internal/weather"
)

// runRoot executes the root command with args and returns stdout, the error.
func runRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()
	c := cmd.NewRootCmd()
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs(args)
	err := c.ExecuteContext(context.Background())
	return out.String(), err
}

func TestRootVerboseFormat(t *testing.T) {
	restore := cmd.SetAppFactoryForTest(func(cfg *config.Config) (*app.App, error) {
		return app.New(cfg,
			geo.FakeLocator{Loc: geo.Location{PostalCode: "22901"}},
			geo.FakeGeocoder{Loc: geo.Location{PostalCode: "22901"}},
			weather.FakeProvider{Report: weather.Report{TempF: 70, HumidityPct: 45, AQI: 30, WeatherCode: 0}},
		)
	})
	defer restore()

	out, err := runRoot(t, "--no-config", "-v")
	require.NoError(t, err)
	assert.Equal(t, "ok (0) [22901]", strings.TrimSpace(out))
}

func TestRootNopeExitCode(t *testing.T) {
	restore := cmd.SetAppFactoryForTest(func(cfg *config.Config) (*app.App, error) {
		return app.New(cfg,
			geo.FakeLocator{Loc: geo.Location{PostalCode: "22901"}},
			geo.FakeGeocoder{},
			weather.FakeProvider{Report: weather.Report{TempF: 96, HumidityPct: 45, AQI: 30, WeatherCode: 0}},
		)
	})
	defer restore()

	out, err := runRoot(t, "--no-config")
	require.Error(t, err)
	assert.Equal(t, 1, cli.CodeFor(err))
	assert.Equal(t, "nope", strings.TrimSpace(out))
}

func TestRootJSON(t *testing.T) {
	restore := cmd.SetAppFactoryForTest(func(cfg *config.Config) (*app.App, error) {
		return app.New(cfg,
			geo.FakeLocator{Loc: geo.Location{PostalCode: "22901"}},
			geo.FakeGeocoder{},
			weather.FakeProvider{Report: weather.Report{TempF: 70, HumidityPct: 45, AQI: 30, WeatherCode: 0}},
		)
	})
	defer restore()

	out, err := runRoot(t, "--no-config", "--json")
	require.NoError(t, err)
	assert.Contains(t, out, `"verdict": "ok"`)
	assert.Contains(t, out, `"location": "22901"`)
}
