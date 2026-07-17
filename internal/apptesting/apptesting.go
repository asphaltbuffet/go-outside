package apptesting

import (
	"testing"

	"github.com/asphaltbuffet/go-outside/internal/app"
	"github.com/asphaltbuffet/go-outside/internal/config"
	"github.com/asphaltbuffet/go-outside/internal/geo"
	"github.com/asphaltbuffet/go-outside/internal/weather"
)

// Comfortable fixture Conditions used by the default test App (well inside
// every Factor's comfortable band, so the Verdict is ok).
const (
	comfortableTempF       = 70.0
	comfortableHumidityPct = 45.0
	comfortableAQI         = 30
	clearWeatherCode       = 0
)

// OpenApp creates an App wired with fake providers for use in tests.
func OpenApp(t *testing.T) *app.App {
	t.Helper()

	cfg := config.GetDefaults()
	a, err := app.New(
		cfg,
		geo.FakeLocator{Loc: geo.Location{PostalCode: "22901"}},
		geo.FakeGeocoder{Loc: geo.Location{PostalCode: "22901"}},
		weather.FakeProvider{Report: weather.Report{
			TempF:       comfortableTempF,
			HumidityPct: comfortableHumidityPct,
			AQI:         comfortableAQI,
			WeatherCode: clearWeatherCode,
		}},
	)
	if err != nil {
		t.Fatalf("apptesting.OpenApp: %v", err)
	}
	return a
}
