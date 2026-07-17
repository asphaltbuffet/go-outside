package apptesting

import (
	"testing"

	"github.com/asphaltbuffet/go-outside/internal/app"
	"github.com/asphaltbuffet/go-outside/internal/config"
	"github.com/asphaltbuffet/go-outside/internal/geo"
	"github.com/asphaltbuffet/go-outside/internal/weather"
)

// OpenApp creates an App wired with fake providers for use in tests.
func OpenApp(t *testing.T) *app.App {
	t.Helper()

	cfg := config.GetDefaults()
	a, err := app.New(
		cfg,
		geo.FakeLocator{Loc: geo.Location{PostalCode: "22901"}},
		geo.FakeGeocoder{Loc: geo.Location{PostalCode: "22901"}},
		weather.FakeProvider{Report: weather.Report{TempF: 70, HumidityPct: 45, AQI: 30, WeatherCode: 0}},
	)
	if err != nil {
		t.Fatalf("apptesting.OpenApp: %v", err)
	}
	return a
}
