package weather_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/asphaltbuffet/go-outside/internal/geo"
	"github.com/asphaltbuffet/go-outside/internal/weather"
)

func TestFakeProvider(t *testing.T) {
	want := weather.Report{TempF: 70, HumidityPct: 45, AQI: 30, WeatherCode: 0}
	var p weather.Provider = weather.FakeProvider{Report: want}
	got, err := p.Fetch(context.Background(), geo.Location{})
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestOpenMeteoFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "air-quality") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"current": map[string]any{"us_aqi": 42},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"current": map[string]any{
				"temperature_2m":       68.0,
				"relative_humidity_2m": 55.0,
				"weather_code":         61,
			},
		})
	}))
	defer srv.Close()

	p := weather.NewOpenMeteo(resty.New().SetBaseURL(srv.URL))
	got, err := p.Fetch(context.Background(), geo.Location{Lat: 38.03, Lon: -78.47})
	require.NoError(t, err)
	assert.InDelta(t, 68.0, got.TempF, 0.001)
	assert.InDelta(t, 55.0, got.HumidityPct, 0.001)
	assert.Equal(t, 42, got.AQI)
	assert.Equal(t, 61, got.WeatherCode)
}
