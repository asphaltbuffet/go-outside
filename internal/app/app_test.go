package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/asphaltbuffet/go-outside/internal/app"
	"github.com/asphaltbuffet/go-outside/internal/config"
	"github.com/asphaltbuffet/go-outside/internal/geo"
	"github.com/asphaltbuffet/go-outside/internal/score"
	"github.com/asphaltbuffet/go-outside/internal/weather"
)

func newApp(t *testing.T, loc geo.Locator, gc geo.Geocoder, wp weather.Provider) *app.App {
	t.Helper()
	a, err := app.New(config.GetDefaults(), loc, gc, wp)
	require.NoError(t, err)
	return a
}

func TestAssessAutoDetectOK(t *testing.T) {
	loc := geo.FakeLocator{Loc: geo.Location{PostalCode: "22901", Lat: 38, Lon: -78}}
	wp := weather.FakeProvider{Report: weather.Report{TempF: 70, HumidityPct: 45, AQI: 30, WeatherCode: 0}}
	a := newApp(t, loc, geo.FakeGeocoder{}, wp)

	got, err := a.Assess(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, score.OK, got.Verdict)
	assert.Equal(t, "22901", got.Location.PostalCode)
}

func TestAssessOverrideGeocodes(t *testing.T) {
	gc := geo.FakeGeocoder{Loc: geo.Location{PostalCode: "90210", Lat: 34, Lon: -118}}
	wp := weather.FakeProvider{Report: weather.Report{TempF: 96, HumidityPct: 45, AQI: 30, WeatherCode: 0}}
	// Locator must NOT be consulted when override is present.
	loc := geo.FakeLocator{Err: errors.New("should not be called")}
	a := newApp(t, loc, gc, wp)

	got, err := a.Assess(context.Background(), "90210")
	require.NoError(t, err)
	assert.Equal(t, score.Nope, got.Verdict)
	assert.Equal(t, "90210", got.Location.PostalCode)
}

func TestAssessDetectErrorPropagates(t *testing.T) {
	loc := geo.FakeLocator{Err: errors.New("network down")}
	a := newApp(t, loc, geo.FakeGeocoder{}, weather.FakeProvider{})

	_, err := a.Assess(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network down")
}

func TestAssessWeatherErrorPropagates(t *testing.T) {
	loc := geo.FakeLocator{Loc: geo.Location{PostalCode: "22901"}}
	wp := weather.FakeProvider{Err: errors.New("api timeout")}
	a := newApp(t, loc, geo.FakeGeocoder{}, wp)

	_, err := a.Assess(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api timeout")
}
